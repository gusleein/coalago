package pools

import (
	"github.com/coalalib/coalago/pools/StoragePools"
)

type activeRequestPool struct {
	storage *StoragePools.Storage
}

func (h *activeRequestPool) Set(id string) {
	h.storage.Set(ACTIVE_REQUEST_NAME, id, true)
}

func (h *activeRequestPool) Get(id string) interface{} {
	b := h.storage.Get(ACTIVE_REQUEST_NAME, id)
	if b == nil {
		return nil
	}
	return b
}

func (h *activeRequestPool) Delete(id string) {
	h.storage.Delete(ACTIVE_REQUEST_NAME, id)
	return
}
