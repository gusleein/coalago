package coalago

import (
	"errors"
	"net"
	"sync"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/queue"
)

const (
	DEFAULT_WINDOW_SIZE = 70
	MAX_PAYLOAD_SIZE    = 512
)

type layerARQ struct {
	coala        *Coala
	rxStates     *ARQStatesPool
	txStates     *ARQStatesPool
	messagePool  *queue.Queue
	callbackPool *sync.Map
	emptyAcks    *sync.Map
}

func newLayerARQ(coala *Coala) layerARQ {
	l := layerARQ{
		coala:        coala,
		rxStates:     NewARQStatesPool(),
		txStates:     NewARQStatesPool(),
		messagePool:  queue.New(),
		callbackPool: &sync.Map{},
		emptyAcks:    &sync.Map{},
	}

	go messagePoolSender(coala, l.messagePool, l.callbackPool)
	return l
}

func (l layerARQ) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	return l.ARQReceiveHandler(message)
}

func (l layerARQ) OnSend(coala *Coala, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	return l.ARQSendHandler(message, address), nil
}

func (l layerARQ) sendARQmessage(message *m.CoAPMessage, address *net.UDPAddr, callback CoalaCallback) {
	l.coala.sendMessage(message, address, callback, l.messagePool, l.callbackPool)
}

func (l layerARQ) sendMoreData(token string, windowSize int) {
	state := l.txStates.Get(token)
	if state == nil {
		return
	}

	for {
		msg := state.PopBlock(windowSize)
		if msg == nil {
			break
		}

		l.sendARQmessage(msg, msg.Recipient, func(rsp *m.CoAPMessage, err error) {
			if err != nil {
				l.txStates.Delete(token)
				clb, ok := l.coala.reciverPool.Load(state.origMessage.GetMessageIDString() + msg.Recipient.String())
				if ok {
					clb.(CoalaCallback)(nil, errors.New("arq "+err.Error()))
				}
				l.coala.reciverPool.Delete(token)

				return
			}
		})
	}
}
