package coalago

import (
	"bytes"
	"errors"
	"net"
	"time"

	"github.com/coalalib/coalago/session"
)

var (
	SESSIONS_POOL_EXPIRATION = time.Minute * 5
)

func securityClientSend(tr *transport, message *CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != COAPS_SCHEME {
		return nil
	}

	currentSession := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addr.String())

	if currentSession == nil {
		err := errors.New("Cannot encrypt: no session, message: %v  from: %v")
		return err
	}

	// Perform the Handshake (if necessary)
	err := handshake(tr, message, currentSession, addr)
	if err != nil {
		return err
	}

	MetricSuccessfulHandhshakes.Inc()

	// Encrypt message payload
	err = encrypt(message, addr, currentSession.AEAD)
	if err != nil {
		return err
	}

	return nil
}

func getSessionForAddress(tr *transport, senderAddr, receiverAddr string) *session.SecuredSession {
	s, _ := globalSessions.Get(senderAddr + receiverAddr)
	var (
		err            error
		securedSession *session.SecuredSession
	)

	if s == nil {
		securedSession = nil
	} else {
		securedSession = s.(*session.SecuredSession)
	}
	if securedSession == nil || securedSession.Curve == nil {
		securedSession, err = session.NewSecuredSession(tr.privateKey)
		if err != nil {
			return nil
		}

		setSessionForAddress(tr.privateKey, securedSession, senderAddr, receiverAddr)
	}

	return securedSession
}

func setSessionForAddress(privatekey []byte, securedSession *session.SecuredSession, senderAddr, receiverAddr string) {
	if securedSession == nil {
		securedSession, _ = session.NewSecuredSession(privatekey)
	}
	globalSessions.SetDefault(senderAddr+receiverAddr, securedSession)
	MetricSessionsRate.Inc()
	MetricSessionsCount.Set(int64(globalSessions.ItemCount()))
}

func securityReceive(tr *transport, message *CoAPMessage) bool {
	if !receiveHandshake(tr, tr.privateKey, message) {
		return false
	}
	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == COAPS_SCHEME {
		var addressSession string

		// a, _ := coala.ProxySessions.Get(message.Sender.String())

		// if a == nil {
		addressSession = message.Sender.String()
		// } else {
		// addressSession = a.(string)
		// }

		currentSession := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addressSession)

		if currentSession == nil || currentSession.AEAD == nil {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false
		}

		// Decrypt message payload
		err := decrypt(message, currentSession.AEAD)
		if err != nil {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false
		}

		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(OptionSessionNotFound)
	sessionExpired := message.GetOption(OptionSessionExpired)
	if message.Code == CoapCodeUnauthorized && (sessionNotFound != nil || sessionExpired != nil) {
		globalSessions.Delete(message.Sender.String())
		return false
	}

	return true
}

func receiveHandshake(tr *transport, privatekey []byte, message *CoAPMessage) bool {
	if message.IsProxies {
		return true
	}
	option := message.GetOption(OptionHandshakeType)
	if option == nil {
		return true
	}

	value := option.IntValue()
	if value != CoapHandshakeTypeClientSignature && value != CoapHandshakeTypeClientHello {
		return true
	}

	peerSession := getSessionForAddress(tr, tr.conn.LocalAddr().String(), message.Sender.String())

	if value == CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		err := incomingHandshake(tr, peerSession.Curve.GetPublicKey(), message)
		if err != nil {
			return false
		}
		if signature, err := peerSession.GetSignature(); err == nil {
			peerSession.PeerVerify(signature)
		}

	}
	MetricSuccessfulHandhshakes.Inc()

	peerSession.UpdatedAt = int(time.Now().Unix())
	setSessionForAddress(privatekey, peerSession, tr.conn.LocalAddr().String(), message.Sender.String())

	return false
}

const (
	ERR_KEYS_NOT_MATCH = "Expected and current public keys do not match"
)

func handshake(tr *transport, message *CoAPMessage, session *session.SecuredSession, address net.Addr) error {
	// We skip handshake if session already exists
	if session.AEAD != nil {
		return nil
	}

	// Sending my Public Key.
	// Receiving Peer's Public Key as a Response!
	peerPublicKey, err := sendHelloFromClient(tr, message, session.Curve.GetPublicKey(), address)
	if err != nil {
		return err
	}

	// assign new value
	session.PeerPublicKey = peerPublicKey

	signature, err := session.GetSignature()
	if err != nil {
		return err
	}

	return session.Verify(signature)
}

func sendHelloFromClient(tr *transport, origMessage *CoAPMessage, myPublicKey []byte, address net.Addr) ([]byte, error) {
	var peerPublicKey []byte
	message := newClientHelloMessage(origMessage, myPublicKey)

	respMsg, err := tr.Send(message)
	if err != nil {
		return nil, err
	}

	if respMsg == nil {
		return nil, nil
	}

	optHandshake := respMsg.GetOption(OptionHandshakeType)
	if optHandshake != nil {
		if optHandshake.IntValue() == CoapHandshakeTypePeerHello {
			peerPublicKey = respMsg.Payload.Bytes()
		}
	}

	if len(origMessage.PublicKey) > 1 && !bytes.Equal(peerPublicKey, origMessage.PublicKey) {
		return nil, errors.New(ERR_KEYS_NOT_MATCH)
	}
	origMessage.PublicKey = peerPublicKey
	return peerPublicKey, nil
}

func newClientHelloMessage(origMessage *CoAPMessage, myPublicKey []byte) *CoAPMessage {
	message := NewCoAPMessage(CON, POST)
	message.AddOption(OptionHandshakeType, CoapHandshakeTypeClientHello)
	message.Payload = NewBytesPayload(myPublicKey)
	message.Token = generateToken(6)
	message.CloneOptions(origMessage, OptionProxyURI)
	return message
}

func newServerHelloMessage(origMessage *CoAPMessage, publicKey []byte) *CoAPMessage {
	message := NewCoAPMessageId(ACK, CoapCodeContent, origMessage.MessageID)
	message.AddOption(OptionHandshakeType, CoapHandshakeTypePeerHello)
	message.Payload = NewBytesPayload(publicKey)
	message.Token = origMessage.Token
	return message
}

func incomingHandshake(tr *transport, publicKey []byte, origMessage *CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	if _, err := tr.SendTo(message, origMessage.Sender); err != nil {
		return err
	}

	return nil
}
