package pools

import (
	"net"

	"github.com/coalalib/coalago/pools/StoragePools"
)

type proxyPool struct {
	storage *StoragePools.Storage
}

func (h *proxyPool) Set(id string, address net.Addr) {
	h.storage.Set(PROXY_POOL_NAME, id, address)
}

func (h *proxyPool) Get(id string) net.Addr {
	obj := h.storage.Get(PROXY_POOL_NAME, id)
	if obj == nil {
		return nil
	}

	return obj.(net.Addr)
}

func (h *proxyPool) Delete(id string) {
	h.storage.Delete(PROXY_POOL_NAME, id)
}

func (h *proxyPool) Pop(id string) net.Addr {
	obj := h.storage.Pop(PROXY_POOL_NAME, id)
	if obj == nil {
		return nil
	}
	return obj.(net.Addr)
}

func (h *proxyPool) Count() int {
	return h.storage.Count(PROXY_POOL_NAME)
}
