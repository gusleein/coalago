package StoragePools

import (
	"github.com/op/go-logging"
	"github.com/patrickmn/go-cache"
)

var (
	log = logging.MustGetLogger("StoragePools")
)

type Storage struct {
	pools map[string]*cache.Cache
}

func NewStoragePool() *Storage {
	d := new(Storage)
	d.pools = make(map[string]*cache.Cache)
	return d
}

func (d *Storage) setPool(name string, cacheObject *cache.Cache) {
	d.pools[name] = cacheObject
}

func (d *Storage) write(poolName, id string, obj interface{}) {
	d.pools[poolName].SetDefault(id, obj)
}

func (d *Storage) read(poolName, id string) interface{} {
	obj, _ := d.pools[poolName].Get(id)
	if obj != nil {
		d.write(poolName, id, obj)
	}
	return obj
}

func (d *Storage) delete(poolName, id string) {
	d.pools[poolName].Delete(id)
}

func (d *Storage) readAll(poolName string) map[string]interface{} {
	items := d.pools[poolName].Items()
	objectsMap := make(map[string]interface{})
	for k, v := range items {
		objectsMap[k] = v.Object
		d.write(poolName, k, v.Object)
	}
	return objectsMap
}

func (s *Storage) count(poolName string) int {
	return s.pools[poolName].ItemCount()
}
