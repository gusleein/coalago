package coalago

import (
	"bytes"
	"errors"
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network/session"
	"github.com/labstack/gommon/log"
	cache "github.com/patrickmn/go-cache"
)

var (
	SESSIONS_POOL_EXPIRATION = time.Minute * 5
)

func securityClientSend(tr *transport, sessions *cache.Cache, privatekey []byte, message *m.CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != m.COAPS_SCHEME {
		return nil
	}

	currentSession := getSessionForAddress(tr, sessions, privatekey, addr.String())

	if currentSession == nil {
		err := errors.New("Cannot encrypt: no session, message: %v  from: %v")
		log.Errorf(err.Error(), message.ToReadableString(), message.Sender.String())
		return err
	}

	// Perform the Handshake (if necessary)
	err := handshake(tr, message, currentSession, addr)
	if err != nil {
		log.Error(err, message.ToReadableString(), currentSession)
		return err
	}

	// Encrypt message payload
	err = Encrypt(message, addr, currentSession.AEAD)
	if err != nil {
		log.Error(err, message.ToReadableString(), message.Sender.String(), currentSession)
		return err
	}

	return nil
}

func getSessionForAddress(tr *transport, sessions *cache.Cache, privatekey []byte, udpAddr string) *session.SecuredSession {
	s, _ := sessions.Get(udpAddr)
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
		securedSession, err = session.NewSecuredSession(privatekey)
		if err != nil {
			log.Error(err)
			return nil
		}
		// coala.Metrics.SessionsRate.Inc()
		setSessionForAddress(sessions, privatekey, securedSession, udpAddr)
	}

	return securedSession
}

func setSessionForAddress(sessions *cache.Cache, privatekey []byte, securedSession *session.SecuredSession, udpAddr string) {
	if securedSession == nil {
		securedSession, _ = session.NewSecuredSession(privatekey)
	}
	sessions.SetDefault(udpAddr, securedSession)
	// coala.Metrics.Sessions.Set(int64(coala.Sessions.ItemCount()))
}

func securityReceive(tr *transport, sessions *cache.Cache, privatekey []byte, message *m.CoAPMessage) bool {
	if !receiveHandshake(tr, sessions, privatekey, message) {
		return false
	}
	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == m.COAPS_SCHEME {
		var addressSession string

		// a, _ := coala.ProxySessions.Get(message.Sender.String())

		// if a == nil {
		addressSession = message.Sender.String()
		// } else {
		// addressSession = a.(string)
		// }

		currentSession := getSessionForAddress(tr, sessions, privatekey, addressSession)

		if currentSession == nil || currentSession.AEAD == nil {
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false
		}

		// Decrypt message payload
		err := Decrypt(message, currentSession.AEAD)
		if err != nil {
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false
		}

		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(m.OptionSessionNotFound)
	sessionExpired := message.GetOption(m.OptionSessionExpired)
	if message.Code == m.CoapCodeUnauthorized && (sessionNotFound != nil || sessionExpired != nil) {
		sessions.Delete(message.Sender.String())
		return false
	}

	return true
}

func receiveHandshake(tr *transport, sessions *cache.Cache, privatekey []byte, message *m.CoAPMessage) bool {
	if message.IsProxies {
		return true
	}
	option := message.GetOption(m.OptionHandshakeType)
	if option == nil {
		return true
	}

	value := option.IntValue()
	if value != m.CoapHandshakeTypeClientSignature && value != m.CoapHandshakeTypeClientHello {
		// coala.GetMetrics().SuccessfulHandshakes.Inc()
		return true
	}

	peerSession := getSessionForAddress(tr, sessions, privatekey, message.Sender.String())

	if value == m.CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		err := incomingHandshake(tr, peerSession.Curve.GetPublicKey(), message)
		if err != nil {
			return false
		}
		if signature, err := peerSession.GetSignature(); err != nil {
			log.Error(err)
		} else {
			peerSession.PeerVerify(signature)
		}

	}
	// coala.GetMetrics().SuccessfulHandshakes.Inc()

	peerSession.UpdatedAt = int(time.Now().Unix())
	setSessionForAddress(sessions, privatekey, peerSession, message.Sender.String())

	return false
}

const (
	ERR_KEYS_NOT_MATCH = "Expected and current public keys do not match"
)

func handshake(tr *transport, message *m.CoAPMessage, session *session.SecuredSession, address net.Addr) error {
	// We skip handshake if session already exists
	if session.AEAD != nil {
		return nil
	}

	// Sending my Public Key.
	// Receiving Peer's Public Key as a Response!
	peerPublicKey, err := sendHelloFromClient(tr, message, session.Curve.GetPublicKey(), address)
	if err != nil {
		log.Error("sendHelloFromClient", err.Error())
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

func sendHelloFromClient(tr *transport, origMessage *m.CoAPMessage, myPublicKey []byte, address net.Addr) ([]byte, error) {
	var peerPublicKey []byte
	message := newClientHelloMessage(origMessage, myPublicKey)

	respMsg, err := tr.Send(message)
	if err != nil {
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

func newClientHelloMessage(origMessage *m.CoAPMessage, myPublicKey []byte) *m.CoAPMessage {
	message := m.NewCoAPMessage(m.CON, m.POST)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypeClientHello)
	message.Payload = m.NewBytesPayload(myPublicKey)
	message.Token = m.GenerateToken(6)
	message.CloneOptions(origMessage, m.OptionProxyURI)
	return message
}

func newServerHelloMessage(origMessage *m.CoAPMessage, publicKey []byte) *m.CoAPMessage {
	message := m.NewCoAPMessageId(m.ACK, m.CoapCodeContent, origMessage.MessageID)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypePeerHello)
	message.Payload = m.NewBytesPayload(publicKey)
	message.Token = origMessage.Token
	return message
}

func incomingHandshake(tr *transport, publicKey []byte, origMessage *m.CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	if _, err := tr.SendTo(message, origMessage.Sender); err != nil {
		log.Error("Can't send HELLO", err)
		return err
	}

	return nil
}
