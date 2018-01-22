package byteBuffer

import (
	"errors"
	"sync"

	m "github.com/coalalib/coalago/message"

	logging "github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("coala.ARQ.byteBuffer")
)

const FRAME_SIZE = 512

type ARQResponse struct {
	Message *m.CoAPMessage
	IsNext  bool
}

type Buffer struct {
	sync.RWMutex
	lenght         int
	count          int
	frames         map[int]*Frame
	windowSelector int
	readableFrame  int
	lastBlock      int
	ackCount       int
	IsAllACK       bool
}

type Frame struct {
	Body []byte
	ACK  bool
}

func NewBuffer() *Buffer {
	return &Buffer{
		frames:         make(map[int]*Frame),
		windowSelector: 0,
		readableFrame:  0,
		lastBlock:      -1,
		count:          0,
		lenght:         0,
		ackCount:       -1,
	}
}

func (b *Buffer) Byte() []byte {
	b.Lock()
	buf := make([]byte, b.lenght)

	for k, v := range b.frames {
		copy(buf[k*FRAME_SIZE:], v.Body)
	}
	b.Unlock()
	return buf
}

//////////////////////////////////Output methods/////////////////////////////////////

func (b *Buffer) NextFrame() (num int, frame *Frame, next bool) {
	b.RLock()
	defer b.RUnlock()

	frame, ok := b.frames[b.readableFrame]
	if !ok {
		return b.readableFrame, nil, false
	}
	delete(b.frames, b.readableFrame)
	_, next = b.frames[b.readableFrame+1]

	b.readableFrame++
	b.count--
	b.windowSelector = b.readableFrame

	return b.readableFrame - 1, frame, next
}

func (b *Buffer) Window(size int) (window []*Frame, last bool) {
	b.Lock()
	defer b.Unlock()
	var i int
	for i = b.windowSelector; i < b.windowSelector+size; i++ {
		frame, exists := b.frames[i]

		if !exists {
			delete(b.frames, b.windowSelector)
			b.windowSelector++
			return window, true
		}

		window = append(window, frame)
	}

	delete(b.frames, b.windowSelector)

	b.readableFrame = b.windowSelector + size

	b.count--
	b.windowSelector++

	_, exist := b.frames[i]
	return window, !exist
}

/////////////////////////////////Input methods///////////////////////////////////////
func (b *Buffer) Write(object []byte) {
	b.Lock()
	b.frames = make(map[int]*Frame)
	b.lenght = len(object)

	if b.lenght == 0 {
		b.frames[0] = &Frame{
			Body: object[0:0],
		}
		b.count++
	}

	for firstByte := 0; firstByte < b.lenght; firstByte += FRAME_SIZE {
		lastByte := firstByte + FRAME_SIZE
		if lastByte > b.lenght {
			lastByte = b.lenght
		}
		b.frames[b.count] = &Frame{
			Body: object[firstByte:lastByte],
		}
		b.count++
	}
	b.Unlock()
}

func (b *Buffer) WriteToWindow(size int, index int, body []byte) (isFull bool, err error) {
	b.Lock()
	defer b.Unlock()

	if index < b.windowSelector || index > b.windowSelector+size {
		log.Error("index outside the window", index, b.windowSelector, size)
		return b.count == b.lastBlock, errors.New("index outside the window")
	}

	if _, ok := b.frames[index]; ok {
		return b.count-1 == b.lastBlock, nil
	}

	b.frames[index] = &Frame{
		Body: body,
	}
	b.lenght += len(body)
	b.count++

	if index == b.windowSelector {
		b.windowSelector++

		for i := index + 1; ; i++ {
			if _, ok := b.frames[i]; ok {
				b.windowSelector++
			} else {
				break
			}
		}
	}

	return b.count-1 == b.lastBlock, nil
}

/////////////////////////////Others methods/////////////////////////////////////
func (b *Buffer) SetLastBlock(index int) {
	b.Lock()
	b.lastBlock = index
	b.Unlock()
}

func (b *Buffer) IsFirstWindow() bool {
	return b.windowSelector == 0
}

func (b *Buffer) FirstWindowFrame() int {
	b.Lock()
	defer b.Unlock()
	return b.windowSelector
}

func (b *Buffer) AddACK(blockNum int) {
	b.Lock()
	frame, ok := b.frames[blockNum]
	if ok {
		if !frame.ACK {
			frame.ACK = true
			b.ackCount++
		}
	}
	b.IsAllACK = b.ackCount == b.lastBlock
	b.Unlock()
}
