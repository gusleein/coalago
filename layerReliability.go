package coalago

import (
	"fmt"
	"net"

	cache "github.com/patrickmn/go-cache"
)

type reliabilityLayer struct {
	inProcess *cache.Cache
}

func (layer *reliabilityLayer) OnReceive(message *CoAPMessage) bool {
	if message.Type == ACK || message.Type == RST {
		return true
	}

	key := fmt.Sprintf("%s%s", message.GetTokenString(), message.Sender.String())

	if _, ok := layer.inProcess.Get(key); ok {
		return false
	}

	layer.inProcess.SetDefault(key, struct{}{})
	return true
}

func (layer *reliabilityLayer) OnSend(message *CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}
