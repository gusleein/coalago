package SecurityLayer

import (
	"bytes"
	"errors"
	"net"

	"github.com/coalalib/coalago/common"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network/session"
)

const (
	ERR_KEYS_NOT_MATCH = "Expected and current encryption keys do not match"
)

func handshake(coala common.SenderIface, message *m.CoAPMessage, session *session.SecuredSession, address *net.UDPAddr) error {
	// We skip handshake if session already exists
	if session.AEAD != nil {
		return nil
	}

	// Sending my Public Key.
	// Receiving Peer's Public Key as a Response!
	// log.Debug("Handshake start! Sending PK", len(session.Curve.GetPublicKey()))
	peerPublicKey, err := outgoingHandshake(coala, message, session.Curve.GetPublicKey(), address)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Debugf("Received Peer PK  len: %v", len(peerPublicKey))

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

	coala.GetMetrics().SuccessfulHandshakes.Inc()
	return session.Verify(signature)
}

func outgoingHandshake(coala common.SenderIface, origMessage *m.CoAPMessage, myPublicKey []byte, address *net.UDPAddr) ([]byte, error) {
	message := m.NewCoAPMessage(m.CON, m.GET)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypeClientHello)
	message.Payload = m.NewBytesPayload(myPublicKey)
	message.Token = m.GenerateToken(6)
	message.CloneOptions(origMessage, m.OptionProxyURI)

	log.Debugf("\n\nHello: %s, to: %s\n\n", message.ToReadableString(), address.String())

	// serialize the message

	var peerPublicKey []byte

	key := message.GetMessageIDString() + message.GetTokenString() + address.String()

	coala.GetAllPools().ExpectedHandshakePool.NewElement(key)
	if err := coala.GetAllPools().ExpectedHandshakePool.Set(key, message); err != nil {
		log.Error("Error adding message to the handshake pool. Error: ", err, message.ToReadableString())
	}

	respMsg, err := coala.Send(message, address)
	if err != nil {
		coala.GetAllPools().ExpectedHandshakePool.Delete(key)
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

	if origMessage.KeysOpts.IsStaticKey && !bytes.Equal(peerPublicKey, origMessage.KeysOpts.ExpectedKey) {
		return nil, errors.New(ERR_KEYS_NOT_MATCH)
	}
	origMessage.PublicKey = peerPublicKey

	return peerPublicKey, nil
}

func incomingHandshake(coala common.SenderIface, publicKey []byte, origMessage *m.CoAPMessage) error {
	message := m.NewCoAPMessageId(m.ACK, m.CoapCodeContent, origMessage.MessageID)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypePeerHello)
	message.Payload = m.NewBytesPayload(publicKey)
	message.Token = origMessage.Token

	log.Debugf("\n\nHello: %s, from: %s\n\n", message.ToReadableString(), origMessage.Sender.String())

	_, err := coala.Send(message, origMessage.Sender)
	if err != nil {
		log.Error("Can't send HELLO", err)
		return err
	}

	return nil
}
