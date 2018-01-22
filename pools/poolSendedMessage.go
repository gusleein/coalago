package pools

import (
	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/pools/StoragePools"
)

type sendedMessagePool struct {
	storage *StoragePools.Storage
}

func (h *sendedMessagePool) Set(id string, message interface{}) {
	h.storage.Set(SENDEDMESSAGES_POOL_NAME, id, message)
}

func (h *sendedMessagePool) Get(id string) *m.CoAPMessage {
	message := h.storage.Get(SENDEDMESSAGES_POOL_NAME, id)
	if message == nil {
		return nil
	}
	return message.(*m.CoAPMessage)
}

func (h *sendedMessagePool) Delete(id string) {
	h.storage.Delete(SENDEDMESSAGES_POOL_NAME, id)
}
