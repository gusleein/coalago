package coalago

import (
	"net"

	m "github.com/coalalib/coalago/message"
)

type ResponseLayer struct{}

func (layer *ResponseLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	return true
}

func (layer *ResponseLayer) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}
