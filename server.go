package coalago

import (
	"fmt"
	"net"
	"strings"
	"sync"
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

	for {
		readBuf := make([]byte, MTU+1)
	start:
		n, senderAddr, err := s.sr.conn.Listen(readBuf)
		if err != nil {
			panic(err)
		}
		if n == 0 || n > MTU {
			goto start
		}

		message, err := preparationReceivingBufferForStorageLocalStates("receiveOnce", readBuf[:n], senderAddr)
		if err != nil {
			goto start
		}

		fn, ok := StorageLocalStates.Get(senderAddr.String())
		if !ok {
			fn = MakeLocalStateFn(s, s.sr, nil)
			StorageLocalStates.SetDefault(senderAddr.String(), fn)
		}

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
	s.sr.ReceiveMessage(message, func(message *CoAPMessage, err error) {
		if err != nil {
			return
		}
		requestOnReceive(s.getResourceForPathAndMethod(message.GetURIPath(), message.GetMethod()), globalSessions, s.sr, message)
	})
}

func (s *Server) addResource(res *CoAPResource) {
	key := res.Path + fmt.Sprint(res.Method)
	s.resources.Store(key, res)
}

func (s *Server) AddGETResource(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodGet, path, handler))
}

func (s *Server) AddPOSTResource(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodPost, path, handler))
}

func (s *Server) AddPUTResource(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodPut, path, handler))
}

func (s *Server) AddDELETEResource(path string, handler CoAPResourceHandler) {
	s.addResource(NewCoAPResource(CoapMethodDelete, path, handler))
}

func (s *Server) getResourceForPathAndMethod(path string, method CoapMethod) *CoAPResource {
	path = strings.Trim(path, "/ ")
	key := path + fmt.Sprint(method)
	res, ok := s.resources.Load(key)
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
