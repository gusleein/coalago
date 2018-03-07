package queue

import (
	"container/list"
	"sync"
)

type Queue struct {
	l  *list.List
	m  *sync.Map
	mx sync.Mutex
}

func New() *Queue {
	return &Queue{
		l: list.New(),
		m: &sync.Map{},
	}
}

func (q *Queue) Push(key string, elem interface{}) {
	q.mx.Lock()
	e := q.l.PushBack(elem)
	q.m.Store(key, e)
	q.mx.Unlock()

}

func (q *Queue) Pop() interface{} {
	q.mx.Lock()

	e := q.l.Front()
	if e == nil {
		q.mx.Unlock()
		return nil
	}

	q.l.MoveToBack(e)
	q.mx.Unlock()

	return e.Value
}

func (q *Queue) Delete(key string) {
	q.mx.Lock()
	e, ok := q.m.Load(key)
	if ok {
		q.l.Remove(e.(*list.Element))
		q.m.Delete(key)
	}
	q.mx.Unlock()
}
