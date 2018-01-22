package coalago

import (
	"net"

	"github.com/coalalib/coalago/common"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/SecurityLayer"
)

type Layer interface {
	OnReceive(coala common.SenderIface, message *m.CoAPMessage) bool
	OnSend(coala common.SenderIface, message *m.CoAPMessage, address *net.UDPAddr) (bool, error)
}

type LayersStack struct {
	coala *Coala
	stack []Layer
}

func NewReceiveLayersStack(coala *Coala) *LayersStack {
	stack := [...]Layer{
		&ResponseLayer{},
		&SecurityLayer.HandshakeLayer{},
		&SecurityLayer.SecurityLayer{},

		&layerARQ{},

		&ResourceDiscoveryLayer{},
		&RequestLayer{},
	}

	return &LayersStack{stack: stack[:], coala: coala}
}

func NewSendLayersStack(coala *Coala) *LayersStack {
	stack := [...]Layer{
		&SecurityLayer.SecurityLayer{},
	}

	return &LayersStack{stack: stack[:], coala: coala}
}

func (stack *LayersStack) OnReceive(message *m.CoAPMessage) {
	for _, layer := range stack.stack {
		shouldContinue := layer.OnReceive(stack.coala, message)
		if !shouldContinue {
			break
		}
	}
}

func (stack *LayersStack) OnSend(message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	for _, layer := range stack.stack {
		shouldContinue, err := layer.OnSend(stack.coala, message, address)
		if !shouldContinue || err != nil {
			return false, err
		}
	}
	return true, nil
}
