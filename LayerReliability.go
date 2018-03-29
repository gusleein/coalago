package coalago

import (
	"fmt"
	"net"

	m "github.com/coalalib/coalago/message"
)

type ReliabilityLayer struct{}

func (layer *ReliabilityLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if message.Type == m.ACK || message.Type == m.RST {
		return true
	}

	key := fmt.Sprintf("%s%s%s", message.GetTokenString(), message.GetMessageIDString(), message.Sender.String())

	if _, ok := coala.InProcessingsRequests.Get(key); !ok {
		return false
	}

	coala.InProcessingsRequests.SetDefault(key, struct{}{})
	return true
}

func (layer *ReliabilityLayer) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}
