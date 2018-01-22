package pools

import (
	"errors"
	"sync"

	m "github.com/coalalib/coalago/message"
)

type ExpectedHandshakerPool struct {
	sync.RWMutex
	m map[string][]*m.CoAPMessage
}

func newExpectedHandshakePool() *ExpectedHandshakerPool {
	p := new(ExpectedHandshakerPool)
	p.m = make(map[string][]*m.CoAPMessage)
	return p
}

func (e *ExpectedHandshakerPool) IsEmpty(key string) bool {
	e.Lock()
	defer e.Unlock()
	if e.m[key] == nil {
		return true
	}
	return false
}

func (e *ExpectedHandshakerPool) Set(key string, msg *m.CoAPMessage) error {
	e.Lock()
	defer e.Unlock()
	if e.m[key] == nil {
		return errors.New("Queue for set not exists")
	}
	e.m[key] = append(e.m[key], msg)

	return nil
}

func (e *ExpectedHandshakerPool) Pull(key string) (msg *m.CoAPMessage) {
	e.Lock()
	defer e.Unlock()
	if e.m[key] == nil {
		return nil
	}

	if len(e.m[key]) == 0 {
		delete(e.m, key)
		return nil
	}
	msg, e.m[key] = e.m[key][0], e.m[key][1:] //FIFO
	return msg
}

func (e *ExpectedHandshakerPool) Delete(key string) {
	e.Lock()
	defer e.Unlock()
	delete(e.m, key)
}

func (e *ExpectedHandshakerPool) NewElement(key string) {
	e.Lock()
	defer e.Unlock()
	e.m[key] = []*m.CoAPMessage{}
}
