package pools

import (
	"time"

	"github.com/coalalib/coalago/pools/StoragePools"
)

const (
	SESSIONS_POOL_NAME       = "sessions"
	SENDEDMESSAGES_POOL_NAME = "sendmessages"
	PROXY_POOL_NAME          = "proxy"
)

var (
	CLEANING_INTERVAL = time.Second * 1

	SESSIONS_POOL_EXPIRATION  = time.Second * 60 * 5
	SENDEDMESSAGES_EXPIRATION = time.Second * 4
	PROXY_EXPIRATION          = time.Second * 4

	Pools AllPools
)

var (
	ListPools = map[string]time.Duration{
		SESSIONS_POOL_NAME:       SESSIONS_POOL_EXPIRATION,
		SENDEDMESSAGES_POOL_NAME: SENDEDMESSAGES_EXPIRATION,
		PROXY_POOL_NAME:          PROXY_EXPIRATION,
	}
)

type AllPools struct {
	storage               *StoragePools.Storage
	Sessions              sessionsPool
	ExpectedHandshakePool *ExpectedHandshakerPool
	ProxyPool             proxyPool
	SendMessages          sendedMessagePool
}

func NewPools() *AllPools {
	s := StoragePools.NewStoragePool()
	a := &AllPools{
		storage:               s,
		Sessions:              sessionsPool{storage: s},
		ExpectedHandshakePool: newExpectedHandshakePool(),
		SendMessages:          sendedMessagePool{storage: s},
		ProxyPool:             proxyPool{storage: s},
	}

	for k, v := range ListPools {
		a.storage.AddPool(k, v, CLEANING_INTERVAL)
	}

	return a
}
