package coalago

import (
	"errors"
	"net"
	"sync"

	m "github.com/coalalib/coalago/message"
)

const (
	DEFAULT_WINDOW_SIZE = 70
	MAX_PAYLOAD_SIZE    = 512
)

type layerARQ struct {
	coala     *Coala
	rxStates  *ARQStatesPool
	txStates  *ARQStatesPool
	emptyAcks *sync.Map
}

func newLayerARQ(coala *Coala) layerARQ {
	l := layerARQ{
		coala:    coala,
		rxStates: NewARQStatesPool(),
		txStates: NewARQStatesPool(),
		// callbackPool: &sync.Map{},
		emptyAcks: &sync.Map{},
	}

	return l
}

func (l layerARQ) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	return l.ARQReceiveHandler(message)
}

func (l layerARQ) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	return l.ARQSendHandler(message, address), nil
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

		_, err := l.coala.sendMessage(msg, msg.Recipient)
		if err != nil {
			l.txStates.Delete(token)
			clb, ok := l.coala.reciverPool.Load(state.origMessage.GetMessageIDString() + msg.Recipient.String())
			if ok {
				clb.(CoalaCallback)(nil, errors.New("arq "+err.Error()))
			}
			l.coala.reciverPool.Delete(token)

			return
		}
	}
}
