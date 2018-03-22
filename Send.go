package coalago

import (
	"errors"
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network"
)

func (coala *Coala) Send(message *m.CoAPMessage, address net.Addr) (response *m.CoAPMessage, err error) {
	return coala.sendMessage(message, address)
}

func (coala *Coala) sendMessage(message *m.CoAPMessage, address net.Addr) (response *m.CoAPMessage, err error) {
	var (
		callback CoalaCallback
		errChan  = make(chan error)
	)
	if message.Type == m.CON {
		callback = func(r *m.CoAPMessage, e error) {
			response = r
			errChan <- err
		}
	}
	address = network.NewAddress(address.String())
	message.Recipient = address

	if callback != nil {
		coala.reciverPool.Store(message.GetMessageIDString()+message.Recipient.String(), callback)
	}

	shouldContinue, err := coala.sendLayerStack.OnSend(message, address)
	if err != nil {
		coala.reciverPool.Delete(message.GetMessageIDString() + message.Recipient.String())
		return nil, err
	}
	if !shouldContinue {
		return nil, nil
	}

	message.LastSent = time.Now()
	if message.Attempts > 1 {
		coala.Metrics.Retransmissions.Inc()
	}

LabelRetransmit:
	message.Attempts++
	sendToSocket(coala, message, message.Recipient)

	if message.Type == m.ACK {
		return nil, nil
	}

LabelRepeatWaiting:
	select {
	case <-time.Tick(time.Second * 3):
		if time.Since(message.LastSent) < 3 {
			goto LabelRepeatWaiting
		}

		if message.Attempts >= 3 {
			coala.reciverPool.Delete(message.GetMessageIDString() + message.Recipient.String())
			coala.Metrics.ExpiredMessages.Inc()
			err = errors.New("Max attempts")
			return nil, err
		}

		goto LabelRetransmit
	case err = <-errChan:
		return response, err
	}
}

func sendToSocket(coala *Coala, message *m.CoAPMessage, address net.Addr) error {
	data, err := m.Serialize(message)
	if err != nil {
		return err
	}

	// fmt.Printf("\n|-----> %v\t%v\tPAYLOAD: %v\n\n", address, message.ToReadableString(), "message.Payload.String()")
	_, err = coala.connection.WriteTo(data, address)
	if err != nil {
		coala.Metrics.SentMessageError.Inc()
	}
	coala.Metrics.SentMessages.Inc()
	return err
}

func getBufferKeyForReceive(msg *m.CoAPMessage) string {
	return msg.Sender.String() + msg.GetTokenString()
}

func getBufferKeyForSend(msg *m.CoAPMessage, address net.Addr) string {
	return address.String() + msg.GetTokenString()
}
