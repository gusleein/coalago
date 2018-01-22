package Coala

import (
	"net"

	m "github.com/coalalib/coalago/message"
)

type ReliabilityLayer struct{}

func (layer *ReliabilityLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if message.Type == m.ACK || message.Type == m.RST {
		return true
	}

	coala.Pools.ActiveRequests.Set(message.GetMessageIDString() + message.Sender.String())
	return true
}

func (layer *ReliabilityLayer) OnSend(coala *Coala, message *m.CoAPMessage, address *net.UDPAddr) bool {
	return true
}
