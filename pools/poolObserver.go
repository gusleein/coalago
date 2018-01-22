package pools

import "github.com/coalalib/coalago/pools/StoragePools"

type observersPool struct {
	storage *StoragePools.Storage
}

func (h *observersPool) Set(id string, obsrv interface{}) {
	h.storage.Set(OBSERVERS_POOL_NAME, id, obsrv)
}

func (h *observersPool) Get(id string) interface{} {
	obsrv := h.storage.Get(OBSERVERS_POOL_NAME, id)
	if obsrv == nil {
		return nil
	}
	return obsrv
}

func (h *observersPool) GetAll() map[string]interface{} {
	return h.storage.GetAll(OBSERVERS_POOL_NAME)
}

func (h *observersPool) Delete(id string) {
	h.storage.Delete(OBSERVERS_POOL_NAME, id)
}

func (h *observersPool) Pop(id string) interface{} {
	obsrv := h.storage.Pop(OBSERVERS_POOL_NAME, id)
	if obsrv == nil {
		return nil
	}
	return obsrv
}

func (h *observersPool) Count() int {
	return h.storage.Count(OBSERVERS_POOL_NAME)
}
