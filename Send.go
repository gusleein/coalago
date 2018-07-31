package coalago

import (
	"net"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network"
)

func (coala *Coala) Send(message *m.CoAPMessage, address net.Addr) (response *m.CoAPMessage, err error) {
	chErr := make(chan error)

	var once sync.Once
	if message.Type == m.CON {
		callback := func(r *m.CoAPMessage, e error) {
			once.Do(func() {
				response = r
				select {
				case <-time.After(time.Millisecond * 10):
				case chErr <- e:
				}
			})

		}

		coala.sendMessage(message, address, callback, coala.pendingsMessage, coala.acknowledgePool)
		err = <-chErr
	} else {
		coala.sendMessage(message, address, nil, coala.pendingsMessage, coala.acknowledgePool)
	}
	return
}

func (coala *Coala) sendMessage(message *m.CoAPMessage, address net.Addr, callback CoalaCallback, messagePool chan *m.CoAPMessage, callbackPool *ackPool) {
	address = network.NewAddress(address.String())
	message.Recipient = address

	if callback != nil {
		callbackPool.Load(newPoolID(message.MessageID, message.Token, message.Recipient), callback)
	}

	shouldContinue, err := coala.sendLayerStack.OnSend(message, address)
	if err != nil {
		if callback != nil {
			callbackPool.Delete(newPoolID(message.MessageID, message.Token, message.Recipient))
			callback(nil, err)
		}

		return
	}
	if !shouldContinue {
		return
	}

	messagePool <- message
	return
}

func sendToSocket(coala *Coala, message *m.CoAPMessage, address net.Addr) error {
	data, err := m.Serialize(message)
	if err != nil {
		return err
	}

	// fmt.Printf("\n|-----> %s\t%v\n\n", address, message.ToReadableString())
	_, err = coala.connection.WriteTo(data, address)
	if err != nil {
		coala.Metrics.SentMessageError.Inc()
	}
	coala.Metrics.SentMessages.Inc()
	return err
}
