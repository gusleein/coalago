package Coala

import (
	"net"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/observer"
)

func (coala *Coala) Subscribe(message *m.CoAPMessage, address *net.UDPAddr) (respMessage *m.CoAPMessage, responseChannel chan *m.CoAPMessage, err error) {
	responseChannel = make(chan *m.CoAPMessage, 32)
	coala.observeMessages[message.GetTokenString()+address.String()] = responseChannel

	message.AddOption(m.OptionObserve, 0)
	respMessage, err = coala.Send(message, address)
	return
}

func (coala *Coala) Unsubscribe(token []byte, address *net.UDPAddr) (resp *m.CoAPMessage, err error) {
	msg := m.NewCoAPMessage(m.RST, m.GET)
	msg.Token = token
	resp, err = coala.Send(msg, address)
	return
}

type ObserverLayer struct{}

func (layer *ObserverLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if message.Type == m.ACK {
		return true
	}

	if message.Type == m.RST {
		coala.GetAllPools().Observers.Delete(message.Sender.String() + "|" + message.GetTokenString())
	}

	option := message.GetOption(m.OptionObserve)
	if option == nil {
		// check Response Code Error (if any)
		if message.Code.IsCommonError() {
			// any 4.XX Codes
			coala.GetAllPools().Observers.Delete(message.Sender.String() + "|" + message.GetTokenString())
		}
		return true
	}

	if option.IntValue() == 0 { // Register!

		condition := coala.GetObserverCondition(message.GetURIPath())
		if condition == nil {
			return true
		}
		callback := observer.NewObserverCallback(message, condition)
		coala.Pools.Observers.Set(callback.Key, callback)
		return true
	}
	if option.IntValue() == 1 { // Unregister!
		coala.Pools.Observers.Delete(message.Sender.String() + "|" + message.GetTokenString())
		return true
	}

	respChannel, ok := coala.observeMessages[message.GetTokenString()+message.Sender.String()]

	if ok {
		sendObserverACK(coala, message)
		respChannel <- message
		return false
	}

	return true
}

func (layer *ObserverLayer) OnSend(coala *Coala, message *m.CoAPMessage, address *net.UDPAddr) bool {
	return true
}

func sendObserverACK(coala *Coala, msg *m.CoAPMessage) {
	r := m.NewCoAPMessage(m.ACK, m.CoapCodeEmpty)
	r.MessageID = msg.MessageID
	r.Token = msg.Token
	r.CloneOptions(msg, m.OptionObserve, m.OptionBlock1, m.OptionBlock2, m.OptionSelectiveRepeatWindowSize)
	coala.Send(r, msg.Sender)
}
