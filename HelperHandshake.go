package coalago

import (
	"bytes"
	"errors"
	"net"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network/session"
)

const (
	ERR_KEYS_NOT_MATCH = "Expected and current public keys do not match"
)

func handshake(coala *Coala, message *m.CoAPMessage, session *session.SecuredSession, address net.Addr) error {
	// We skip handshake if session already exists
	if session.AEAD != nil {
		return nil
	}

	// Sending my Public Key.
	// Receiving Peer's Public Key as a Response!
	peerPublicKey, err := outgoingHandshake(coala, message, session.Curve.GetPublicKey(), address)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if len(peerPublicKey) == 0 {
		log.Error("Empty public key for message: ", message.GetMessageIDString())
	}

	// assign new value
	session.PeerPublicKey = peerPublicKey

	signature, err := session.GetSignature()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return session.Verify(signature)
}

func outgoingHandshake(coala *Coala, origMessage *m.CoAPMessage, myPublicKey []byte, address net.Addr) ([]byte, error) {
	message := m.NewCoAPMessage(m.CON, m.GET)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypeClientHello)
	message.Payload = m.NewBytesPayload(myPublicKey)
	message.Token = m.GenerateToken(6)
	message.CloneOptions(origMessage, m.OptionProxyURI)

	var peerPublicKey []byte

	respMsg, err := coala.Send(message, address)
	if err != nil {
		log.Error("Cannot send HELLO", err)
		return nil, err
	}
	if respMsg == nil {
		return nil, nil
	}

	optHandshake := respMsg.GetOption(m.OptionHandshakeType)
	if optHandshake != nil {
		if optHandshake.IntValue() == m.CoapHandshakeTypePeerHello {
			peerPublicKey = respMsg.Payload.Bytes()
		}
	}

	if len(origMessage.PublicKey) > 1 && !bytes.Equal(peerPublicKey, origMessage.PublicKey) {
		return nil, errors.New(ERR_KEYS_NOT_MATCH)
	}
	origMessage.PublicKey = peerPublicKey

	return peerPublicKey, nil
}

func incomingHandshake(coala *Coala, publicKey []byte, origMessage *m.CoAPMessage) error {
	message := m.NewCoAPMessageId(m.ACK, m.CoapCodeContent, origMessage.MessageID)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypePeerHello)
	message.Payload = m.NewBytesPayload(publicKey)
	message.Token = origMessage.Token

	_, err := coala.Send(message, origMessage.Sender)
	if err != nil {
		log.Error("Can't send HELLO", err)
		return err
	}

	return nil
}
