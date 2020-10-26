package coalago

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/coalalib/coalago/session"
	"github.com/patrickmn/go-cache"
)

type sessionStorage interface {
	Set(sender string, receiver string, proxy string, sess *session.SecuredSession)
	Get(sender string, receiver string, proxy string) *session.SecuredSession
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

func (s *sessionStorageImpl) Set(sender string, receiver string, proxy string, sess *session.SecuredSession) {
	if len(proxy) != 0 {
		sender = ""
	}
	s.storage.SetDefault(sender+receiver+proxy, sess)
}

func (s *sessionStorageImpl) Get(sender string, receiver string, proxy string) *session.SecuredSession {
	if len(proxy) != 0 {
		sender = ""
	}
	v, ok := s.storage.Get(sender + receiver + proxy)
	if ok {
		return v.(*session.SecuredSession)
	}
	return nil
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

// For local state

type localStateSessionStorageImpl struct {
	count   int32
	storage sync.Map
}

func newLocalStateSessionStorageImpl() *localStateSessionStorageImpl {
	s := new(localStateSessionStorageImpl)
	return s
}

func (s *localStateSessionStorageImpl) Set(sender string, receiver string, proxy string, sess *session.SecuredSession) {
	if len(proxy) != 0 {
		sender = ""
	}
	atomic.AddInt32(&s.count, 1)
	s.storage.Store(sender+receiver+proxy, sess)
}

func (s *localStateSessionStorageImpl) Get(sender string, receiver string, proxy string) *session.SecuredSession {
	if len(proxy) != 0 {
		sender = ""
	}
	v, ok := s.storage.Load(sender + receiver + proxy)
	if ok {
		return v.(*session.SecuredSession)
	}
	return nil
}

func (s *localStateSessionStorageImpl) Delete(sender string, receiver string, proxy string) {
	if len(proxy) != 0 {
		sender = ""
	}
	atomic.AddInt32(&s.count, -1)
	s.storage.Delete(sender + receiver + proxy)
}

func (s *localStateSessionStorageImpl) ItemCount() int {
	v := atomic.LoadInt32(&s.count)
	return int(v)
}
