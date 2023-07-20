package coalaServer

import (
	"time"

	"github.com/gusleein/coalago/session"
	"github.com/patrickmn/go-cache"
)

const (
	sessionLifetime = time.Minute*4 + time.Second*9
)

type sessionState struct {
	key string
	est time.Time
}

type securitySessionStorage struct {
	// rwmx sync.RWMutex
	// m       map[string]session.SecuredSession
	// indexes map[string]time.Time
	// est     []sessionState
	seccache *cache.Cache
}

func newSecuritySessionStorage() *securitySessionStorage {
	s := &securitySessionStorage{
		seccache: cache.New(sessionLifetime, time.Second),
	}

	return s
}

func (s *securitySessionStorage) Set(k string, v session.SecuredSession) {
	s.seccache.SetDefault(k, v)
}

func (s *securitySessionStorage) Delete(k string) {
	s.seccache.Delete(k)
}

func (s *securitySessionStorage) Update(k string, sess session.SecuredSession) {
	s.seccache.SetDefault(k, sess)
}

func (s *securitySessionStorage) Get(k string) (sess session.SecuredSession, ok bool) {
	v, ok := s.seccache.Get(k)
	if !ok {
		return sess, ok
	}
	sess = v.(session.SecuredSession)
	return sess, ok
}
