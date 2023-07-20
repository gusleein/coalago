package coalago

import (
	"time"

	"github.com/gusleein/coalago/session"
	"github.com/patrickmn/go-cache"
)

type sessionStorage interface {
	Set(sender string, receiver string, proxy string, sess *session.SecuredSession)
	Get(sender string, receiver string, proxy string) (*session.SecuredSession, bool)
	Delete(sender string, receiver string, proxy string)
	ItemCount() int
}

type sessionStorageImpl struct {
	storage *cache.Cache
}

func newSessionStorageImpl() *sessionStorageImpl {
	s := new(sessionStorageImpl)
	s.storage = cache.New(SESSIONS_POOL_EXPIRATION, time.Second*1)
	return s
}

func (s *sessionStorageImpl) Set(sender string, receiver string, proxy string, sess session.SecuredSession) {
	if len(proxy) != 0 {
		sender = "" //TODO понять надо ли это зануление
	}
	s.storage.SetDefault(sender+receiver+proxy, sess)
}

func (s *sessionStorageImpl) Get(sender string, receiver string, proxy string) (session.SecuredSession, bool) {
	if len(proxy) != 0 {
		sender = ""
	}
	v, ok := s.storage.Get(sender + receiver + proxy)
	if ok {
		return v.(session.SecuredSession), true
	}
	return session.SecuredSession{}, false
}

func (s *sessionStorageImpl) Delete(sender string, receiver string, proxy string) {
	if len(proxy) != 0 {
		sender = ""
	}
	s.storage.Delete(sender + receiver + proxy)
}

func (s *sessionStorageImpl) ItemCount() int {
	return s.storage.ItemCount()
}
