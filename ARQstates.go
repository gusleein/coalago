package coalago

import (
	"bytes"
	"math"
	"sync"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/window"
)

type ARQStatesPool struct {
	pool *sync.Map
}

func NewARQStatesPool() *ARQStatesPool {
	return &ARQStatesPool{
		pool: &sync.Map{},
	}
}

func (a *ARQStatesPool) Set(key string, state *ARQState) {
	a.pool.Store(key, state)
}

func (a *ARQStatesPool) Get(key string) *ARQState {
	s, ok := a.pool.Load(key)
	if ok {
		return s.(*ARQState)
	}
	return nil
}

func (a *ARQStatesPool) Delete(key string) {
	a.pool.Delete(key)
}

type ARQState struct {
	origMessage *m.CoAPMessage
	initMessage *m.CoAPMessage
	Callbacks   *sync.Map

	blockSize   int
	blockOption m.OptionCode

	window      *window.SlidingWindow
	accumulator *bytes.Buffer

	lastBlockNumber int

	mx sync.Mutex
}

func NewSendState(data []byte, windowSize int, blockOption m.OptionCode, blockSize int, origMessage *m.CoAPMessage) *ARQState {
	a := new(ARQState)
	a.accumulator = bytes.NewBuffer(data)
	a.blockSize = blockSize
	a.origMessage = origMessage
	a.blockOption = blockOption

	totalBlocks := math.Ceil(float64(len(data)) / float64(blockSize))
	windowSize = int(math.Min(float64(windowSize), float64(totalBlocks)))

	a.window = window.NewSlidingWindow(windowSize, 0-windowSize)

	if windowSize <= 0 {
		return a
	}

	for i := 0 - windowSize; i <= -1; i++ {
		a.window.Set(i, true)
	}
	return a
}

func (a *ARQState) GetWindowSize() int {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.window.GetSize()
}

func (a *ARQState) PopBlock(windowSize int) *m.CoAPMessage {
	a.mx.Lock()
	defer a.mx.Unlock()
	if a.window.Advance() == nil {
		return nil
	}

	blockNumber := a.window.Tail()
	rangeStart := blockNumber * a.blockSize
	rangeStop := int(math.Min(float64(rangeStart+a.blockSize), float64(a.accumulator.Len())))

	if rangeStart >= rangeStop {
		return nil
	}

	msg := newBlockingMessage(
		a.origMessage,
		a.origMessage.Recipient,
		a.accumulator.Bytes()[rangeStart:rangeStop],
		a.blockOption,
		blockNumber,
		windowSize,
		rangeStop != a.accumulator.Len(),
	)
	if proxxyuri := a.origMessage.GetOptionProxyURIasString(); len(proxxyuri) > 0 {
		msg.SetProxyURI(proxxyuri)
	}
	return msg
}

func (a *ARQState) GetOriginalMessage() *m.CoAPMessage {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.origMessage
}

func (a *ARQState) IsCompleted() bool {
	a.mx.Lock()
	defer a.mx.Unlock()
	lastDeliveredBlock := a.window.GetOffset()
	index := 0

	for {
		if index < a.window.GetSize() && a.window.GetValue(index) != nil && a.window.GetValue(index).(bool) {
			lastDeliveredBlock++
			index++
		} else {
			break
		}
	}
	return lastDeliveredBlock*a.blockSize >= a.accumulator.Len()
}

func (a *ARQState) didTransmit(blockNumber int) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.window.Set(blockNumber, true)
}

//---- receive state

func NewReceiveState(windowSize int, initMessage *m.CoAPMessage) *ARQState {

	a := new(ARQState)
	a.window = window.NewSlidingWindow(windowSize, 0)
	a.initMessage = initMessage
	a.accumulator = bytes.NewBuffer([]byte{})
	return a
}

func (a *ARQState) GetInitiatingMessage() *m.CoAPMessage {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.initMessage
}

func (a *ARQState) GetData() []byte {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.accumulator.Bytes()
}

func (a *ARQState) IsTransferCompleted() bool {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.lastBlockNumber > 0 && a.window.GetOffset()-1 == a.lastBlockNumber
}

func (a *ARQState) DidReceiveBlock(blockNumber int, isMore bool, data []byte, windowSize int, code m.CoapCode) error {
	a.mx.Lock()
	defer a.mx.Unlock()
	if code != m.CoapCodeContinue {
		a.initMessage.Code = code
	}

	a.window.Set(blockNumber, data)

	if !isMore {
		a.lastBlockNumber = blockNumber
	}

	var b []byte

	for {
		fm := a.window.Advance()
		if fm == nil {
			break
		}
		b = fm.([]byte)
		a.accumulator.Write(b)
	}
	return nil
}
