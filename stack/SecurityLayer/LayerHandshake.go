package SecurityLayer

import (
	"net"
	"time"

	"github.com/coalalib/coalago/common"

	m "github.com/coalalib/coalago/message"
	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("SecurityLayer")

type HandshakeLayer struct{}

func (layer *HandshakeLayer) OnReceive(coala common.SenderIface, message *m.CoAPMessage) bool {
	if message.IsProxies {
		return true
	}
	option := message.GetOption(m.OptionHandshakeType)
	if option == nil {
		return true
	}

	value := option.IntValue()
	if value != m.CoapHandshakeTypeClientSignature && value != m.CoapHandshakeTypeClientHello {
		return true
	}

	peerSession := coala.GetSessionForAddress(message.Sender)

	if value == m.CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		err := incomingHandshake(coala, peerSession.Curve.GetPublicKey(), message)
		if err != nil {
			log.Error(err)
		} else {
			signature, err := peerSession.GetSignature()
			if err != nil {
				log.Error(err)
			} else {
				peerSession.PeerVerify(signature)
			}
		}
	}

	peerSession.UpdatedAt = int(time.Now().Unix())
	coala.SetSessionForAddress(peerSession, message.Sender)

	return false
}

func (layer *HandshakeLayer) OnSend(coala common.SenderIface, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	return true, nil
}
