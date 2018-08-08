package coalago

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
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

func (s *Server) Listen(addr string) (err error) {
	conn, err := newListener(addr)
	if err != nil {
		return err
	}

	s.sr = newtransport(conn)

	for {
		s.sr.ReceiveOnce(func(message *m.CoAPMessage, err error) {
			if err != nil {
				return
			}
			requestOnReceive(s, message)
		})
	}
}

func (s *Server) addResource(res *resource.CoAPResource) {
	key := res.Path + fmt.Sprint(res.Method)
	s.resources.Store(key, res)
}

func (s *Server) AddGETResource(path string, handler resource.CoAPResourceHandler) {
	s.addResource(resource.NewCoAPResource(m.CoapMethodGet, path, handler))
}

func (s *Server) AddPOSTResource(path string, handler resource.CoAPResourceHandler) {
	s.addResource(resource.NewCoAPResource(m.CoapMethodPost, path, handler))
}

func (s *Server) AddPUTResource(path string, handler resource.CoAPResourceHandler) {
	s.addResource(resource.NewCoAPResource(m.CoapMethodPut, path, handler))
}

func (s *Server) AddDELETEResource(path string, handler resource.CoAPResourceHandler) {
	s.addResource(resource.NewCoAPResource(m.CoapMethodDelete, path, handler))
}

func (s *Server) getResourceForPathAndMethod(path string, method m.CoapMethod) *resource.CoAPResource {
	path = strings.Trim(path, "/ ")
	key := path + fmt.Sprint(method)
	res, ok := s.resources.Load(key)
	if ok {
		return res.(*resource.CoAPResource)
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
