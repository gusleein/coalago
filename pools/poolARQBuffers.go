package pools

import (
	"github.com/coalalib/coalago/pools/StoragePools"
)

type arqBuffersPool struct {
	storage *StoragePools.Storage
}

func (h *arqBuffersPool) Set(id string, buffer interface{}) {
	h.storage.Set(ARQBUFFERS_POOL_NAME, id, buffer)
}

func (h *arqBuffersPool) Get(id string) interface{} {
	buf := h.storage.Get(ARQBUFFERS_POOL_NAME, id)
	if buf == nil {
		return nil
	}
	return buf
}

func (h *arqBuffersPool) Delete(id string) {
	h.storage.Delete(ARQBUFFERS_POOL_NAME, id)
}
