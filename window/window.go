package window

import "sync/atomic"

type SlidingWindow struct {
	offset int32
	values []interface{}
}

func NewSlidingWindow(size int, offset int) *SlidingWindow {
	return &SlidingWindow{
		offset: int32(offset),
		values: make([]interface{}, size),
	}
}

func (s *SlidingWindow) GetOffset() int {
	return int(atomic.LoadInt32(&s.offset))
}

func (s *SlidingWindow) GetSize() int {
	return len(s.values)
}

func (s *SlidingWindow) Set(number int, value interface{}) {
	windowIndex := number - int(atomic.LoadInt32(&s.offset))

	if windowIndex > len(s.values)-1 {
		return
	} else if windowIndex < 0 {
		return
	}

	s.values[windowIndex] = value
}

func (s *SlidingWindow) Advance() interface{} {
	if len(s.values) == 0 {
		return nil
	}

	firstBlock := s.values[0]
	if firstBlock == nil {
		return nil
	}

	copy(s.values, s.values[1:])
	s.values[len(s.values)-1] = nil

	atomic.AddInt32(&s.offset, 1)

	return firstBlock
}

func (s *SlidingWindow) GetValue(windowIndex int) interface{} {
	return s.values[windowIndex]
}

func (s *SlidingWindow) Tail() int {
	return int(atomic.LoadInt32(&s.offset)) + len(s.values) - 1
}
