package pools

import (
	"github.com/coalalib/coalago/network/session"
	"github.com/coalalib/coalago/pools/StoragePools"
)

type sessionsPool struct {
	storage *StoragePools.Storage
}

func (h *sessionsPool) Set(id string, privateKey []byte, obj *session.SecuredSession) {
	if obj == nil {
		obj, _ = session.NewSecuredSession(privateKey)
	}
	h.storage.Set(SESSIONS_POOL_NAME, id, obj)
}

func (h *sessionsPool) Get(id string) *session.SecuredSession {
	obj := h.storage.Get(SESSIONS_POOL_NAME, id)
	if obj == nil {
		return nil
	}

	h.storage.Set(SESSIONS_POOL_NAME, id, obj)
	return obj.(*session.SecuredSession)
}

func (h *sessionsPool) Delete(id string) {
	h.storage.Delete(SESSIONS_POOL_NAME, id)
}

func (h *sessionsPool) Pop(id string) *session.SecuredSession {
	obj := h.storage.Pop(SESSIONS_POOL_NAME, id)
	if obj == nil {
		return nil
	}
	return obj.(*session.SecuredSession)
}

func (h *sessionsPool) Count() int {
	return h.storage.Count(SESSIONS_POOL_NAME)
}
