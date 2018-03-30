package pools

import (
	"time"

	"github.com/coalalib/coalago/pools/StoragePools"
)

const (
	PROXY_POOL_NAME = "proxy"
)

var (
	CLEANING_INTERVAL = time.Second * 1
	PROXY_EXPIRATION  = time.Second * 4

	Pools AllPools
)

var (
	ListPools = map[string]time.Duration{
		PROXY_POOL_NAME: PROXY_EXPIRATION,
	}
)

type AllPools struct {
	storage   *StoragePools.Storage
	ProxyPool proxyPool
}

func NewPools() *AllPools {
	s := StoragePools.NewStoragePool()
	a := &AllPools{
		storage:   s,
		ProxyPool: proxyPool{storage: s},
	}

	for k, v := range ListPools {
		a.storage.AddPool(k, v, CLEANING_INTERVAL)
	}

	return a
}
