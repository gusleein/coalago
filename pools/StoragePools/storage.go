package StoragePools

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

func (s *Storage) AddPool(poolName string, expiration, interval time.Duration) {
	log.Info("Add pool: ", poolName, expiration, interval)
	cacheObject := cache.New(expiration, interval)
	s.setPool(poolName, cacheObject)
}

func (s *Storage) Set(poolName, id string, obj interface{}) {
	s.write(poolName, id, obj)
}

func (s *Storage) Get(poolName string, id string) interface{} {
	return s.read(poolName, id)
}

func (s *Storage) GetAll(poolName string) map[string]interface{} {
	return s.readAll(poolName)
}

func (s *Storage) Delete(poolName string, id string) {
	s.delete(poolName, id)
}

func (s *Storage) Pop(poolName string, id string) interface{} {
	obj := s.read(poolName, id)
	s.delete(poolName, id)
	return obj
}
func (s *Storage) Count(poolName string) int {
	return s.count(poolName)
}
