package coalago

import (
	"net"

	"github.com/coalalib/coalago/common"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/ARQLayer"
)

type layerARQ struct {
}

func (l *layerARQ) OnReceive(coala common.SenderIface, message *m.CoAPMessage) bool {
	return ARQLayer.OnReceive(coala, message)
}

func (l *layerARQ) OnSend(coala common.SenderIface, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	return true, nil
}
