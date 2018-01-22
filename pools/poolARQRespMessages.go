package pools

import (
	"github.com/coalalib/coalago/pools/StoragePools"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
)

type arqRespMessages struct {
	storage *StoragePools.Storage
}

func (h *arqRespMessages) Set(id string, resp chan *byteBuffer.ARQResponse) {
	h.storage.Set(ARQRESP_POOL_NAME, id, resp)
}

func (h *arqRespMessages) Get(id string) chan *byteBuffer.ARQResponse {
	msg := h.storage.Get(ARQRESP_POOL_NAME, id)
	if msg == nil {
		return nil
	}
	return msg.(chan *byteBuffer.ARQResponse)
}

func (h *arqRespMessages) Delete(id string) {
	h.storage.Delete(ARQRESP_POOL_NAME, id)
}
