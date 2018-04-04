package coalago

import (
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network"
)

func (coala *Coala) Send(message *m.CoAPMessage, address net.Addr) (response *m.CoAPMessage, err error) {
	var (
		callback CoalaCallback
		chErr    = make(chan error)
	)
	if message.Type == m.CON {
		callback = func(r *m.CoAPMessage, e error) {
			response = r
			select {
			case <-time.After(time.Millisecond * 1):
			case chErr <- err:
			}

		}

		coala.sendMessage(message, address, callback, coala.pendingsMessage, coala.acknowledgePool)
		err = <-chErr
	} else {
		coala.sendMessage(message, address, callback, coala.pendingsMessage, coala.acknowledgePool)
	}

	return
}

func (coala *Coala) sendMessage(message *m.CoAPMessage, address net.Addr, callback CoalaCallback, messagePool *Queue, callbackPool *ackPool) {
	address = network.NewAddress(address.String())
	message.Recipient = address

	if callback != nil {
		callbackPool.Load(newPoolID(message.MessageID, message.Token, message.Recipient), callback)
	}

	shouldContinue, err := coala.sendLayerStack.OnSend(message, address)
	if err != nil {
		callbackPool.Delete(newPoolID(message.MessageID, message.Token, message.Recipient))
		callback(nil, err)
		return
	}
	if !shouldContinue {
		return
	}

	messagePool.Push(newPoolID(message.MessageID, message.Token, address), message)
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
