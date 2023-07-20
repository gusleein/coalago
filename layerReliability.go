package coalago

import (
	"fmt"
	"net"

	m "github.com/gusleein/coalago/message"
	cache "github.com/patrickmn/go-cache"
)

type reliabilityLayer struct {
	inProcess *cache.Cache
}

func (layer *reliabilityLayer) OnReceive(message *m.CoAPMessage) bool {
	if message.Type == m.ACK || message.Type == m.RST {
		return true
	}

	key := fmt.Sprintf("%s%s", message.GetTokenString(), message.Sender.String())

	if _, ok := layer.inProcess.Get(key); ok {
		return false
	}

	layer.inProcess.SetDefault(key, struct{}{})
	return true
}

func (layer *reliabilityLayer) OnSend(message *m.CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}
