package coalago

import (
	"sync"
)

type Element struct {
	next, prev *Element
	queue      *Queue
	key        string
	Value      interface{}
}

type Queue struct {
	root     Element
	m        map[string]*Element
	len      int
	mx       sync.Mutex
	callback func()
}

func (q *Queue) Init() *Queue {
	q.root.next = &q.root
	q.root.prev = &q.root
	q.len = 0
	q.m = make(map[string]*Element)
	q.mx = sync.Mutex{}
	return q
}

func NewQueue() *Queue { return new(Queue).Init() }

func (q *Queue) Len() int { return q.len }

func (q *Queue) Pop() *Element {
	q.mx.Lock()
	defer q.mx.Unlock()
	if q.len == 0 {
		var wg sync.WaitGroup
		wg.Add(1)
		q.callback = func() {
			wg.Done()
			q.callback = nil
		}

		q.mx.Unlock()

		wg.Wait()

		q.mx.Lock()
	}
	e := q.root.next
	if e == nil {
		return nil
	}
	q.moveToBack(e)
	return e
}

func (q *Queue) Push(key string, v interface{}) {
	q.mx.Lock()
	q.lazyInit()
	e := q.insertValue(v, q.root.prev)
	e.key = key
	q.m[key] = e
	if q.callback != nil {
		q.callback()
	}
	q.mx.Unlock()
}

func (q *Queue) moveToBack(e *Element) {
	if e.queue != q || q.root.prev == e {
		return
	}
	q.insert(q.remove(e), q.root.prev)
}

func (q *Queue) RemoveByKey(key string) *Element {
	q.mx.Lock()
	defer q.mx.Unlock()
	e, ok := q.m[key]
	if !ok {
		return nil
	}
	if e.queue == q {
		delete(q.m, key)
		q.remove(e)
	}
	return e
}

func (q *Queue) Remove(e *Element) *Element {
	q.mx.Lock()
	defer q.mx.Unlock()
	if e.queue == q {
		delete(q.m, e.key)
		q.remove(e)
	}
	return e
}

func (q *Queue) lazyInit() {
	if q.root.next == nil {
		q.Init()
	}
}

func (q *Queue) insert(e, at *Element) *Element {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	e.queue = q
	q.len++
	return e
}

func (q *Queue) insertValue(v interface{}, at *Element) *Element {
	return q.insert(&Element{Value: v}, at)
}

func (q *Queue) remove(e *Element) *Element {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.queue = nil
	q.len--
	return e
}
