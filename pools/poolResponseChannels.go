package pools

import (
	m "github.com/coalalib/coalago/message"

	"github.com/coalalib/coalago/pools/StoragePools"
)

type responsesPool struct {
	storage *StoragePools.Storage
}

func (h *responsesPool) Set(id string, msg chan *m.CoAPMessage) {
	h.storage.Set(RESPONSES_POOL_NAME, id, msg)
}

func (h *responsesPool) Get(id string) chan *m.CoAPMessage {
	msg := h.storage.Get(RESPONSES_POOL_NAME, id)
	if msg == nil {
		return nil
	}
	return msg.(chan *m.CoAPMessage)
}

func (h *responsesPool) Delete(id string) {
	h.storage.Delete(RESPONSES_POOL_NAME, id)
}

func (h *responsesPool) Pop(id string) chan *m.CoAPMessage {
	msg := h.storage.Pop(RESPONSES_POOL_NAME, id)
	if msg == nil {
		return nil
	}
	return msg.(chan *m.CoAPMessage)
}

func (h *responsesPool) GetAll() map[string]interface{} {
	msgs := h.storage.GetAll("responses")
	return msgs
}
