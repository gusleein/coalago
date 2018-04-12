package coalago

import (
	"net"

	m "github.com/coalalib/coalago/message"
)

type Layer interface {
	OnReceive(coala *Coala, message *m.CoAPMessage) bool
	OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error)
}

type LayersStack struct {
	coala *Coala
	stack []Layer
}

func NewLayersStacks(coala *Coala) (receiveStack *LayersStack, sendStack *LayersStack) {
	arqLayer := newLayerARQ(coala)
	secLayer := newSecurityLayer()

	stackReceive := [...]Layer{
		&ProxyLayer{},
		&ReliabilityLayer{},
		secLayer,
		arqLayer,
		&ResourceDiscoveryLayer{},
		&RequestLayer{},
	}

	stackSend := [...]Layer{
		&ProxyLayer{},
		&ReliabilityLayer{},
		arqLayer,
		secLayer,

		//&ResponseLayer{},
	}

	return &LayersStack{stack: stackReceive[:], coala: coala}, &LayersStack{stack: stackSend[:], coala: coala}
}

func (stack *LayersStack) OnReceive(message *m.CoAPMessage) bool {
	for _, layer := range stack.stack {
		if !layer.OnReceive(stack.coala, message) {
			return false
		}
	}
	return true
}

func (stack *LayersStack) OnSend(message *m.CoAPMessage, address net.Addr) (bool, error) {
	for _, layer := range stack.stack {
		if shouldContinue, err := layer.OnSend(stack.coala, message, address); !shouldContinue {
			return false, err
		}
	}
	return true, nil
}
