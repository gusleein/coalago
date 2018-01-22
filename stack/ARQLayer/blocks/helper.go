package blocks

import (
	"net"

	m "github.com/coalalib/coalago/message"
)

func getBufferKeyForReceive(msg *m.CoAPMessage) string {
	return msg.Sender.String() + msg.GetTokenString()
}

func getBufferKeyForSend(msg *m.CoAPMessage, address *net.UDPAddr) string {
	return address.String() + msg.GetTokenString()
}
