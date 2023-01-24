package coalago

import (
	"fmt"
	"net"
	"strings"
	"sync"

	log "github.com/ndmsystems/golog"
)

type rawData struct {
	buff   []byte
	sender net.Addr
}

type Server struct {
	proxyEnable bool
	sr          *transport
	resources   sync.Map
	privatekey  []byte
}

func NewServer() *Server {
	s := new(Server)
	return s
}

func NewServerWithPrivateKey(privatekey []byte) *Server {
	s := NewServer()
	s.privatekey = privatekey

	return s
}

type Resourcer interface {
	getResourceForPathAndMethod(path string, method CoapMethod) *CoAPResource
}

func (s *Server) Listen(addr string) (err error) {
	conn, err := newListener(addr)
	if err != nil {
		return err
	}

	s.sr = newtransport(conn)
	s.sr.privateKey = s.privatekey
	log.Info(fmt.Sprintf(
		"COALAServer start ADDR: %s, WS: %d, MinWS: %d, MaxWS: %d, Retransmit:%d, timeWait:%d, poolExpiration:%d",
		addr, DEFAULT_WINDOW_SIZE, MIN_WiNDOW_SIZE, MAX_WINDOW_SIZE, maxSendAttempts, timeWait, SESSIONS_POOL_EXPIRATION))
	for {
		readBuf := make([]byte, MTU+1)
	start:
		n, senderAddr, err := s.sr.conn.Listen(readBuf)
		if err != nil {
			panic(err)
		}
		if n == 0 {
			goto start
		}

		message, err := preparationReceivingBufferForStorageLocalStates(readBuf[:n], senderAddr)
		if err != nil {
			goto start
		}

		id := senderAddr.String() + message.GetTokenString()
		fn, ok := StorageLocalStates.Get(id)
		if !ok {
			fn = MakeLocalStateFn(s, s.sr, nil, func() {
				StorageLocalStates.Delete(id)
			})
		}
		StorageLocalStates.SetDefault(id, fn)

		go fn.(LocalStateFn)(message)
	}
}

func (s *Server) Serve(conn *net.UDPConn) {
	c := new(connection)
	c.conn = conn
	s.sr = newtransport(c)
	s.sr.privateKey = s.privatekey

}

func (s *Server) ServeMessage(message *CoAPMessage) {
	id := message.Sender.String() + message.GetTokenString()
	fn, ok := StorageLocalStates.Get(id)
	if !ok {
		fn = MakeLocalStateFn(s, s.sr, nil, func() {
			StorageLocalStates.Delete(id)
		})
		StorageLocalStates.SetDefault(id, fn)
	}

	go fn.(LocalStateFn)(message)
}

func (s *Server) addResource(res *CoAPResource) {
	key := res.Path + fmt.Sprint(res.Method)
	s.resources.Store(key, res)
}

func (s *Server) GET(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodGet, path, handler))
}

func (s *Server) POST(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodPost, path, handler))
}

func (s *Server) AddPUTResource(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodPut, path, handler))
}

func (s *Server) DELETE(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodDelete, path, handler))
}

func (s *Server) getResourceForPathAndMethod(path string, method CoapMethod) *CoAPResource {
	path = strings.Trim(path, "/ ")
	key := path + fmt.Sprint(method)
	res, ok := s.resources.Load("*" + fmt.Sprint(method))
	if ok {
		return res.(*CoAPResource)
	}
	res, ok = s.resources.Load(key)
	if ok {
		return res.(*CoAPResource)
	}
	return nil
}

func (s *Server) EnableProxy() {
	s.proxyEnable = true
}

func (s *Server) DisableProxy() {
	s.proxyEnable = false
}

func (s *Server) SetPrivateKey(privateKey []byte) {
	s.privatekey = privateKey
}

func (s *Server) GetPrivateKey() []byte {
	return s.privatekey
}

func (s *Server) SendToSocket(message *CoAPMessage, addr string) error {
	b, err := Serialize(message)
	if err != nil {
		return err
	}
	_, err = s.sr.conn.WriteTo(b, addr)
	return err
}
