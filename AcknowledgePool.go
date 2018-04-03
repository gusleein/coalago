package coalago

import "sync"

type ackPool struct {
	m      map[poolID]CoalaCallback
	locker sync.Mutex
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

func (a *ackPool) GetAndDelete(key poolID) CoalaCallback {
	a.locker.Lock()
	v, _ := a.m[key]
	delete(a.m, key)
	a.locker.Unlock()
	return v
}
