package pools

import (
	"sync"
	"time"

	"github.com/coalalib/coalago/pools/StoragePools"
)

const (
	OBSERVERS_POOL_NAME      = "observers"
	BLOCKWISE_POOL_NAME      = "blockwise"
	SESSIONS_POOL_NAME       = "sessions"
	ACTIVE_REQUEST_NAME      = "activereq"
	RESPONSES_POOL_NAME      = "responses"
	ARQBUFFERS_POOL_NAME     = "arqbuffers"
	ARQRESP_POOL_NAME        = "arqresp"
	SENDEDMESSAGES_POOL_NAME = "sendmessages"
)

var (
	CLEANING_INTERVAL = time.Second * 1

	OBSERVERS_EXPIRATION      = time.Second * 60
	SESSIONS_POOL_EXPIRATION  = time.Second * 60 * 5
	ACTIVE_REQUEST_EXPIRATION = time.Second * 4
	RESPONSES_EXPIRATION      = time.Second * 4
	ARQBUFFERS_EXPIRATION     = time.Second * 4
	ARQRESP_EXPIRATION        = time.Second * 4
	SENDEDMESSAGES_EXPIRATION = time.Second * 4

	Pools AllPools
)

var (
	ListPools = map[string]time.Duration{
		OBSERVERS_POOL_NAME:      OBSERVERS_EXPIRATION,
		SESSIONS_POOL_NAME:       SESSIONS_POOL_EXPIRATION,
		ACTIVE_REQUEST_NAME:      ACTIVE_REQUEST_EXPIRATION,
		RESPONSES_POOL_NAME:      RESPONSES_EXPIRATION,
		ARQBUFFERS_POOL_NAME:     ARQBUFFERS_EXPIRATION,
		ARQRESP_POOL_NAME:        ARQRESP_EXPIRATION,
		SENDEDMESSAGES_POOL_NAME: SENDEDMESSAGES_EXPIRATION,
	}
)

type AllPools struct {
	storage               *StoragePools.Storage
	Observers             observersPool
	Sessions              sessionsPool
	ActiveRequests        activeRequestPool
	Responses             responsesPool
	ARQBuffers            arqBuffersPool
	ARQRespMessages       arqRespMessages
	ExpectedHandshakePool *ExpectedHandshakerPool
	ProxySessions         sync.Map
	SendMessages          sendedMessagePool
}

func NewPools() *AllPools {
	s := StoragePools.NewStoragePool()
	a := &AllPools{
		storage:               s,
		Observers:             observersPool{storage: s},
		Sessions:              sessionsPool{storage: s},
		ActiveRequests:        activeRequestPool{storage: s},
		Responses:             responsesPool{storage: s},
		ARQBuffers:            arqBuffersPool{storage: s},
		ARQRespMessages:       arqRespMessages{storage: s},
		ExpectedHandshakePool: newExpectedHandshakePool(),
		SendMessages:          sendedMessagePool{storage: s},
	}

	for k, v := range ListPools {
		a.storage.AddPool(k, v, CLEANING_INTERVAL)
	}

	return a
}
