package coalago

import (
	"net"

	"github.com/coalalib/coalago/common"

	m "github.com/coalalib/coalago/message"
)

type ResourceDiscoveryResult struct {
	Message       *m.CoAPMessage
	SenderAddress *net.UDPAddr
}

type ResourceDiscoveryLayer struct{}

func (layer *ResourceDiscoveryLayer) OnReceive(coala common.SenderIface, message *m.CoAPMessage) bool {
	return true
}

func (layer *ResourceDiscoveryLayer) OnSend(coala common.SenderIface, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	return true, nil
}
