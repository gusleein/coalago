package coalago

import (
	"net"
	"sync"

	m "github.com/coalalib/coalago/message"
)

const (
	DEFAULT_WINDOW_SIZE = 70
	MAX_PAYLOAD_SIZE    = 1024
)

type layerARQ struct {
	coala     *Coala
	rxStates  *ARQStatesPool
	txStates  *ARQStatesPool
	emptyAcks *sync.Map
}

func newLayerARQ(coala *Coala) *layerARQ {
	l := &layerARQ{
		coala:     coala,
		rxStates:  NewARQStatesPool(),
		txStates:  NewARQStatesPool(),
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
func (l layerARQ) sendARQmessage(message *m.CoAPMessage, address net.Addr, callback CoalaCallback) {
	l.coala.sendMessage(message, address, callback, l.coala.pendingsMessage, l.coala.acknowledgePool)
}

func (l layerARQ) sendMoreData(token string, windowSize int, sendState *ARQState) {
	for {
		msg := sendState.PopBlock(windowSize)
		if msg == nil {
			break
		}

		var once sync.Once
		l.sendARQmessage(msg, msg.Recipient, func(rsp *m.CoAPMessage, err error) {
			once.Do(func() {
				if err != nil {
					l.txStates.Delete(token)
					go l.coala.acknowledgePool.DoDelete(
						newPoolID(sendState.origMessage.MessageID, sendState.origMessage.Token, sendState.origMessage.Recipient),
						rsp,
						err,
					)
					return
				}
			})
		})
	}
}
