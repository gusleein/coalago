package coalago

import (
	"net"
	"sync"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/queue"
)

func (coala *Coala) Send(message *m.CoAPMessage, address *net.UDPAddr) (response *m.CoAPMessage, err error) {
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

	coala.sendMessage(message, address, callback, coala.senderPool, coala.reciverPool)
	wg.Wait()

	return
}

func (coala *Coala) sendMessage(message *m.CoAPMessage, address *net.UDPAddr, callback CoalaCallback, messagePool *queue.Queue, callbackPool *sync.Map) {
	message.Recipient = address

	if callback != nil {
		// fmt.Println("CALLBACK SAVE -----------------> ", message.GetMessageIDString()+message.Recipient.String())
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

func sendToSocket(coala *Coala, message *m.CoAPMessage, address *net.UDPAddr) error {
	// fmt.Printf("\n|-----> %v\t%v\n\n", message.Recipient.String(), message.ToReadableString())

	data, err := m.Serialize(message)
	if err != nil {
		return err
	}

	_, err = coala.connection.WriteTo(data, address)
	return err
}

func getBufferKeyForReceive(msg *m.CoAPMessage) string {
	return msg.Sender.String() + msg.GetTokenString()
}

func getBufferKeyForSend(msg *m.CoAPMessage, address *net.UDPAddr) string {
	return address.String() + msg.GetTokenString()
}
