package coalago

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
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
	sessions    *cache.Cache
}

func NewServer() *Server {
	s := new(Server)
	s.sessions = cache.New(SESSIONS_POOL_EXPIRATION, time.Second*10)

	return s
}

func NewServerWithPrivateKey(privatekey []byte) *Server {
	s := NewServer()
	s.privatekey = privatekey

	return s
}

func (s *Server) Listen(addr string) (err error) {
	conn, err := newListener(addr)
	if err != nil {
		return err
	}

	s.sr = newtransport(conn)
	s.sr.privateKey = s.privatekey
	for {
		s.sr.ReceiveOnce(func(message *CoAPMessage, err error) {
			if err != nil {
				return
			}
			requestOnReceive(s, message)
		})
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
		requestOnReceive(s, message)
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
