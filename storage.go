package coalago

import (
	"time"

	"github.com/coalalib/coalago/session"
	"github.com/patrickmn/go-cache"
)

type sessionStorage struct {
	storage *cache.Cache
}

func newSessionStorage() *sessionStorage {
	s := new(sessionStorage)
	s.storage = cache.New(SESSIONS_POOL_EXPIRATION, time.Second*1)

	return s
}

func (s *sessionStorage) Set(sender string, receiver string, proxy string, sess *session.SecuredSession) {
	s.storage.SetDefault(sender+receiver+proxy, sess)
}

func (s *sessionStorage) Get(sender string, receiver string, proxy string) *session.SecuredSession {
	v, ok := s.storage.Get(sender + receiver + proxy)
	if ok {
		return v.(*session.SecuredSession)
	}
	return nil
}

func (s *sessionStorage) Delete(sender string, receiver string, proxy string) {
	s.storage.Delete(sender + receiver + proxy)
}

func (s *sessionStorage) ItemCount() int {
	return s.storage.ItemCount()
}
