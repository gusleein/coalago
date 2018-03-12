package coalago

import (
	"net"

	m "github.com/coalalib/coalago/message"
)

type ResourceDiscoveryResult struct {
	Message       *m.CoAPMessage
	SenderAddress net.Addr
}

type ResourceDiscoveryLayer struct{}

func (layer *ResourceDiscoveryLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	return true
}

func (layer *ResourceDiscoveryLayer) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}
