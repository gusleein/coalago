package coalago

import (
	"sync"

	m "github.com/coalalib/coalago/message"
)

type ackPool struct {
	m      map[poolID]CoalaCallback
	locker sync.RWMutex
}

func newAckPool() *ackPool {
	return &ackPool{
		m: make(map[poolID]CoalaCallback),
	}
}

func (a *ackPool) Load(key poolID, v CoalaCallback) {
	a.locker.Lock()
	a.m[key] = v
	a.locker.Unlock()
}

func (a *ackPool) Delete(key poolID) {
	a.locker.Lock()
	delete(a.m, key)
	a.locker.Unlock()
}

func (a *ackPool) DoDelete(key poolID, message *m.CoAPMessage, err error) {
	a.locker.Lock()
	v, ok := a.m[key]
	delete(a.m, key)
	a.locker.Unlock()
	if ok {
		v(message, err)
	}
}

func (a *ackPool) IsExists(key poolID) bool {
	a.locker.RLock()
	_, ok := a.m[key]
	a.locker.RUnlock()
	return ok
}
