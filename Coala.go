package coalago

import (
	"bytes"
	"net"
	"strconv"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network"
	"github.com/coalalib/coalago/pools"
	"github.com/coalalib/coalago/resource"
	"github.com/op/go-logging"
	cache "github.com/patrickmn/go-cache"
)

const (
	MESSAGE_EXPIRATION        = 3 // sec
	MESSAGE_CHECKING_INTERVAL = 1 // sec
)

var (
	log = logging.MustGetLogger("coala")
)

type Coala struct {
	connection network.UDPConnection
	resources  []*resource.CoAPResource // We don't need it to be concurrent safe

	dataChannel *DataChannel

	proxyEnabled bool

	receiveLayerStack *LayersStack
	sendLayerStack    *LayersStack
	Metrics           *MetricsList

	Pools                 *pools.AllPools
	Sessions              *cache.Cache
	ProxySessions         *cache.Cache
	InProcessingsRequests *cache.Cache

	pendingsMessage *Queue

	reciverPool *sync.Map

	privatekey []byte
}

func NewListen(port int) *Coala {
	var err error

	coala := new(Coala)
	coala.Pools = pools.NewPools()
	coala.Sessions = cache.New(SESSIONS_POOL_EXPIRATION, time.Second)
	coala.ProxySessions = cache.New(SESSIONS_POOL_EXPIRATION, time.Second)
	coala.InProcessingsRequests = cache.New(10*time.Second, time.Second)

	coala.reciverPool = &sync.Map{}

	coala.dataChannel = &DataChannel{
		Handshake:   make(chan *m.CoAPMessage),
		StopSend:    make(chan bool, 1),
		StopReceive: make(chan bool, 1),
	}

	coala.pendingsMessage = NewQueue()

	for i := 0; i < 4; i++ {
		go pendingMessagesReader(coala, coala.pendingsMessage, coala.reciverPool)
	}

	// Default values
	coala.proxyEnabled = false

	coala.receiveLayerStack, coala.sendLayerStack = NewLayersStacks(coala)
	coala.Metrics = NewMetricList(coala)

	// Init Resource Discovery
	coala.initResourceDiscovery()

	//TODO  remove on production
	coala.initResourceTestsMirror()
	coala.initResourceTestsBlock2()

	// Init Message Dispatching

	coala.connection, err = network.NewUDPConnection(port)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	go coala.listenConnection()
	return coala
}

func (coala *Coala) Close() {
	coala.connection.Close()
}

func (coala *Coala) RunResourceDiscovery() []*ResourceDiscoveryResult {
	// init the data channel first
	coala.dataChannel.Discovery = make(chan *ResourceDiscoveryResult)

	var list []*ResourceDiscoveryResult

	message := m.NewCoAPMessage(m.CON, m.GET)
	message.SetURIPath("/.well-known/core")
	message.SetSchemeCOAP()
	address, _ := net.ResolveUDPAddr("udp", "224.0.0.187:5683")
	coala.Send(message, address)

LabelTimeout: // <- for break by timeout
	select {
	case result := <-coala.dataChannel.Discovery:
		list = append(list, result)
	case <-time.After(3 * time.Second):
		log.Notice("ResourceDiscovery TIMEOUT!")
		break LabelTimeout
	}

	// rest the data channel
	close(coala.dataChannel.Discovery)
	coala.dataChannel.Discovery = nil

	return list
}

type CoalaCallback func(*m.CoAPMessage, error)

func (coala *Coala) AddGETResource(path string, handler resource.CoAPResourceHandler) {
	coala.AddResource(resource.NewCoAPResource(m.CoapMethodGet, path, handler))
}
func (coala *Coala) AddPOSTResource(path string, handler resource.CoAPResourceHandler) {
	coala.AddResource(resource.NewCoAPResource(m.CoapMethodPost, path, handler))
}
func (coala *Coala) AddPUTResource(path string, handler resource.CoAPResourceHandler) {
	coala.AddResource(resource.NewCoAPResource(m.CoapMethodPut, path, handler))
}
func (coala *Coala) AddDELETEResource(path string, handler resource.CoAPResourceHandler) {
	coala.AddResource(resource.NewCoAPResource(m.CoapMethodDelete, path, handler))
}

func (coala *Coala) AddResource(resource *resource.CoAPResource) {
	coala.resources = append(coala.resources, resource)
}

func (coala *Coala) GetResourcesForPath(path string) []*resource.CoAPResource {
	var result []*resource.CoAPResource

	for _, resource := range coala.resources {
		if resource.DoesMatchPath(path) {
			result = append(result, resource)
		}
	}

	return result
}

func (coala *Coala) GetResourcesForPathAndMethod(path string, method m.CoapMethod) []*resource.CoAPResource {
	var result []*resource.CoAPResource
	for _, resource := range coala.resources {
		if resource.DoesMatchPathAndMethod(path, method) {
			result = append(result, resource)
		}
	}

	return result
}

func (coala *Coala) RemoveResourceByHash(hash string) *Coala {
	for r, res := range coala.resources {
		if res.Hash == hash {
			coala.resources[r] = coala.resources[len(coala.resources)-1]
			coala.resources[len(coala.resources)-1] = nil
			coala.resources = coala.resources[:len(coala.resources)-1]
			break
		}
	}
	return coala
}

func (coala *Coala) EnableProxy() {
	coala.proxyEnabled = true
}
func (coala *Coala) DisableProxy() {
	coala.proxyEnabled = false
}

func (coala *Coala) SessionCount() int {
	return coala.Sessions.ItemCount()
}

func (coala *Coala) Stop() {
	log.Info("Stopping Coala...")
	coala.dataChannel.stop()

	if coala.connection != nil {
		coala.connection.Close()
	}
}

func (coala *Coala) initResourceDiscovery() {
	coala.AddGETResource("/.well-known/core", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		var buf bytes.Buffer
		for i, r := range coala.resources {
			if r.Path != ".well-known/core" {
				i++

				buf.WriteString("</")
				buf.WriteString(r.Path)
				buf.WriteString(">")

				// Media Types
				lenMt := len(r.MediaTypes)
				if lenMt > 0 {
					buf.WriteString(";ct=")
					for idx, mt := range r.MediaTypes {

						buf.WriteString(strconv.Itoa(int(mt)))
						if idx+1 < lenMt {
							buf.WriteString(" ")
						}
					}
				}

				// no commas at the end for the last element
				if i != len(coala.resources) {
					buf.WriteString(",")
				}
			}
		}

		handlerResult := resource.NewResponse(m.NewBytesPayload(buf.Bytes()), m.CoapCodeContent)
		handlerResult.MediaType = m.MediaTypeApplicationLinkFormat

		return handlerResult
	})
}

type DataChannel struct {
	StopSend    chan bool
	StopReceive chan bool
	Handshake   chan *m.CoAPMessage
	Discovery   chan *ResourceDiscoveryResult
}

func (dc DataChannel) stop() {
	dc.StopSend <- true
	dc.StopReceive <- true
	if dc.StopReceive != nil {
		close(dc.StopReceive)
	}
	if dc.StopSend != nil {
		close(dc.StopSend)
	}
	if dc.Handshake != nil {
		close(dc.Handshake)
	}
	if dc.Discovery != nil {
		close(dc.Discovery)
	}
}

func (coala *Coala) GetAllPools() *pools.AllPools {
	return coala.Pools
}

func (coala *Coala) IsProxyMode() bool {
	return coala.proxyEnabled
}

func (coala *Coala) StaticPrivateKeyEnable(privateKey []byte) {
	coala.privatekey = privateKey
}

func (coala *Coala) GetPrivateKey() []byte {
	return coala.privatekey
}

func (coala *Coala) GetMetrics() *MetricsList {
	return coala.Metrics
}
