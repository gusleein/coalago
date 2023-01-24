package coalago

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type proxySessionStorage struct {
	storage *cache.Cache
}

func newProxySessionStorage() *proxySessionStorage {
	s := new(proxySessionStorage)
	s.storage = cache.New(time.Minute, time.Second*1)
	return s
}

func (s *proxySessionStorage) Set(key string, value interface{}) {
	s.storage.SetDefault(key, value)
}

func (s *proxySessionStorage) Get(key string) (interface{}, bool) {
	return s.storage.Get(key)
}

func (s *proxySessionStorage) Delete(key string) {
	s.storage.Delete(key)
}
