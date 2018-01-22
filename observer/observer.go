package observer

import (
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
)

type Publisher struct {
	observers map[string]chan *m.CoAPMessage
	sync.RWMutex
}

func New() *Publisher {
	return &Publisher{observers: make(map[string]chan *m.CoAPMessage)}
}

func (p *Publisher) Subscribe(key string) (message *m.CoAPMessage) {
	ch := make(chan *m.CoAPMessage)
	p.Lock()
	p.observers[key] = ch
	p.Unlock()

	message, _ = <-ch
	return
}

func (p *Publisher) Delete(key string) {
	p.Lock()

	if ch := p.observers[key]; ch != nil {
		close(ch)
	}
	p.Unlock()
	return
}

func (p *Publisher) Publish(key string, message *m.CoAPMessage) {
	defer func() {
		recover()
	}()

	p.RLock()
	ch := p.observers[key]
	p.RUnlock()

	if ch != nil {
		select {
		case ch <- message:
			break
		case <-time.After(time.Second):
			break
		}
	}

	close(ch)
	delete(p.observers, key)
}
