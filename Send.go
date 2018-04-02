package coalago

import (
	"net"
	"sync"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network"
)

func (coala *Coala) Send(message *m.CoAPMessage, address net.Addr) (response *m.CoAPMessage, err error) {
	var (
		callback CoalaCallback
		wg       sync.WaitGroup
	)
	if message.Type == m.CON {
		wg.Add(1)
		callback = func(r *m.CoAPMessage, e error) {
			response = r
			err = e
			wg.Done()
		}
	}

	coala.sendMessage(message, address, callback, coala.pendingsMessage, coala.reciverPool)
	wg.Wait()
	return
}

func (coala *Coala) sendMessage(message *m.CoAPMessage, address net.Addr, callback CoalaCallback, messagePool *Queue, callbackPool *sync.Map) {
	address = network.NewAddress(address.String())
	message.Recipient = address

	if callback != nil {
		message.Callback = callback
		callbackPool.Store(message.GetMessageIDString()+message.Recipient.String(), callback)
	}

	shouldContinue, err := coala.sendLayerStack.OnSend(message, address)
	if err != nil {
		callbackPool.Delete(message.GetMessageIDString() + message.Recipient.String())
		callback(nil, err)
		return
	}
	if !shouldContinue {
		return
	}

	messagePool.Push(message.GetMessageIDString()+address.String(), message)
	return
}

func sendToSocket(coala *Coala, message *m.CoAPMessage, address net.Addr) error {
	data, err := m.Serialize(message)
	if err != nil {
		return err
	}

	// fmt.Printf("\n|-----> %v\t%v\n\n", address, message.ToReadableString())
	_, err = coala.connection.WriteTo(data, address)
	if err != nil {
		coala.Metrics.SentMessageError.Inc()
	}
	coala.Metrics.SentMessages.Inc()
	return err
}
