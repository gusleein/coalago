package ARQLayer

import (
	"net"

	"github.com/coalalib/coalago/pools"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
)

func getBufferKeyForReceive(msg *m.CoAPMessage) string {
	return msg.Sender.String() + msg.GetTokenString()
}

func getBufferKeyForSend(msg *m.CoAPMessage, address *net.UDPAddr) string {
	return address.String() + msg.GetTokenString()
}

type Sender interface {
	Send(*m.CoAPMessage, *net.UDPAddr) (*m.CoAPMessage, error)
}

func emptyACKmessage(message *m.CoAPMessage, windowSize int) *m.CoAPMessage {
	ACKmsg := m.NewCoAPMessage(m.ACK, m.CoapCodeEmpty)
	ACKmsg.MessageID = message.MessageID
	ACKmsg.Token = message.Token

	if proxyURI := message.GetOptionProxyURIasString(); proxyURI != "" {
		ACKmsg.SetProxyURI(proxyURI)
	}

	if obsrv := message.GetOption(m.OptionObserve); obsrv != nil {
		ACKmsg.AddOption(m.OptionObserve, obsrv.Value)
		maxAge := message.GetOption(m.OptionMaxAge)
		if maxAge != nil {
			ACKmsg.AddOption(m.OptionMaxAge, maxAge.Value)
		}
	}

	if block := message.GetBlock1(); block != nil {
		ACKmsg.AddOption(m.OptionBlock1, block.ToInt())
	}

	ACKmsg.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)

	return ACKmsg
}

func checkAndSetARQChan(pools *pools.AllPools, message *m.CoAPMessage) {
	if arqRespChan := pools.ARQRespMessages.Get(getBufferKeyForReceive(message)); arqRespChan == nil {
		arqRespChan = make(chan *byteBuffer.ARQResponse)
		pools.ARQRespMessages.Set(getBufferKeyForReceive(message), arqRespChan)
		pools.ARQBuffers.Set(getBufferKeyForReceive(message), byteBuffer.NewBuffer())
	}
}

func isBigPayload(message *m.CoAPMessage) bool {
	return message.Payload != nil && message.Payload.Length() > MAX_PAYLOAD_SIZE
}
