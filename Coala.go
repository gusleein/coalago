package coalago

import (
	"bytes"
	"net"
	"strconv"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/metrics"
	"github.com/coalalib/coalago/network"
	"github.com/coalalib/coalago/network/session"
	"github.com/coalalib/coalago/observer"
	"github.com/coalalib/coalago/pools"
	"github.com/coalalib/coalago/resource"
	"github.com/op/go-logging"
)

const (
	MESSAGE_EXPIRATION        = 3 // sec
	MESSAGE_CHECKING_INTERVAL = 1 // sec
)

var (
	log = logging.MustGetLogger("coala")
)

type Coala struct {
	connection         network.UDPConnection
	resources          []*resource.CoAPResource                   // We don't need it to be concurrent safe
	observerConditions map[string]*observer.CoAPObserverCondition // key = address of resource. Without concurency

	dataChannel *DataChannel

	proxyEnabled bool

	receiveLayerStack *LayersStack
	sendLayerStack    *LayersStack
	Metrics           *metrics.MetricsList

	incomingMessages sync.Map

	observeMessages map[string]chan *m.CoAPMessage

	Pools *pools.AllPools

	privatekey []byte
}

func NewCoala() *Coala {
	var err error

	coala := new(Coala)
	coala.Pools = pools.NewPools()

	coala.connection, err = network.NewUDPConnection(0)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	go coala.listenConnection()

	coala.observerConditions = make(map[string]*observer.CoAPObserverCondition)

	coala.dataChannel = &DataChannel{
		Handshake:   make(chan *m.CoAPMessage),
		StopSend:    make(chan bool, 1),
		StopReceive: make(chan bool, 1),
	}
	// Default values
	coala.proxyEnabled = false

	coala.receiveLayerStack = NewReceiveLayersStack(coala)
	coala.sendLayerStack = NewSendLayersStack(coala)
	coala.Metrics = new(metrics.MetricsList)
	coala.observeMessages = make(map[string]chan *m.CoAPMessage)

	return coala
}

func NewListen(port int) *Coala {
	log.Info("Start Listen Coala on port:", port)

	var err error

	coala := new(Coala)
	coala.Pools = pools.NewPools()

	coala.observerConditions = make(map[string]*observer.CoAPObserverCondition)

	coala.dataChannel = &DataChannel{
		Handshake:   make(chan *m.CoAPMessage),
		StopSend:    make(chan bool, 1),
		StopReceive: make(chan bool, 1),
	}
	// Default values
	coala.proxyEnabled = false

	coala.receiveLayerStack = NewReceiveLayersStack(coala)
	coala.sendLayerStack = NewSendLayersStack(coala)
	coala.Metrics = new(metrics.MetricsList)
	coala.observeMessages = make(map[string]chan *m.CoAPMessage)

	// Init Resource Discovery
	coala.initResourceDiscovery()

	//TODO  remove on production
	coala.initResourceTestsMirror()
	coala.initResourceTestsBlock2()

	// Init Message Dispatching

	observer.StartObserver(coala, coala.Pools)

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

func (coala *Coala) AddConditionOfObserving(resource string, condition *observer.CoAPObserverCondition) {
	coala.observerConditions[resource] = condition
}

func (coala *Coala) GetObserverCondition(key string) *observer.CoAPObserverCondition {
	return coala.observerConditions[key]
}

func (coala *Coala) EnableProxy() {
	coala.proxyEnabled = true
}
func (coala *Coala) DisableProxy() {
	coala.proxyEnabled = false
}

func (coala *Coala) SessionCount() int {
	return coala.Pools.Sessions.Count()
}

func (coala *Coala) Stop() {
	log.Info("Stopping Coala...")
	coala.dataChannel.stop()

	if coala.connection != nil {
		coala.connection.Close()
	}
}

func (coala *Coala) GetSessionForAddress(udpAddr *net.UDPAddr) *session.SecuredSession {
	securedSession := coala.Pools.Sessions.Get(udpAddr.String())
	var err error
	if securedSession == nil || securedSession.Curve == nil {
		securedSession, err = session.NewSecuredSession(coala.GetPrivateKey())
		if err != nil {
			log.Error(err)
			return nil
		}
		coala.Metrics.Sessions.Inc()
		coala.SetSessionForAddress(securedSession, udpAddr)
	}

	return securedSession
}

func (coala *Coala) SetSessionForAddress(securedSession *session.SecuredSession, udpAddr *net.UDPAddr) {
	coala.Pools.Sessions.Set(udpAddr.String(), coala.privatekey, securedSession)
	coala.Metrics.Sessions.Set(int64(coala.Pools.Sessions.Count()))
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

func (coala *Coala) GetMetrics() *metrics.MetricsList {
	return coala.Metrics
}
