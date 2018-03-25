package coalago

import (
	"net"

	m "github.com/coalalib/coalago/message"
)

type ResponseLayer struct{}

func (layer *ResponseLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if message.Code == m.CoapCodeEmpty && message.Type == m.CON {
		// Ping Message! Send RST!
		rst := m.NewCoAPMessageId(m.RST, m.CoapCodeEmpty, message.MessageID)
		coala.Send(rst, message.Sender)
		return false
	}

	return true
}

func (layer *ResponseLayer) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}
