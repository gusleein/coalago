package coalago

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/coalalib/coalago/session"
)

var (
	SESSIONS_POOL_EXPIRATION = time.Minute * 10
)

func securityOutputLayer(tr *transport, message *CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != COAPS_SCHEME {
		return nil
	}

	setProxyIDIfNeed(message)

	proxyAddr := message.ProxyAddr
	if len(proxyAddr) > 0 {
		proxyID, ok := getProxyIDIfNeed(proxyAddr)
		if ok {
			proxyAddr = fmt.Sprintf("%v%v", proxyAddr, proxyID)
		}
	}

	if currentSession := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addr.String(), proxyAddr); currentSession != nil && currentSession.AEAD != nil {
		// Encrypt message payload
		if err := encrypt(message, addr, currentSession.AEAD); err != nil {
			return err
		}
	} else {
		return ErrorSessionNotFound
	}

	return nil
}

func setProxyIDIfNeed(message *CoAPMessage) {
	if message.GetOption(OptionProxyURI) != nil {
		v, ok := proxyIDSessions.Load(message.ProxyAddr)
		if !ok {
			v = rand.Uint32()
			proxyIDSessions.Store(message.ProxyAddr, v)
		}
		message.AddOption(OptionProxySecurityID, v)
	}
}

func getProxyIDIfNeed(proxyAddr string) (uint32, bool) {
	v, ok := proxyIDSessions.Load(proxyAddr)
	if ok {
		return v.(uint32), ok
	}
	return 0, ok
}

func getSessionForAddress(tr *transport, senderAddr, receiverAddr, proxyAddr string) *session.SecuredSession {
	securedSession := globalSessions.Get(senderAddr, receiverAddr, proxyAddr)
	var (
		err error
	)

	if securedSession == nil || securedSession.Curve == nil {
		securedSession, err = session.NewSecuredSession(tr.privateKey)
		if err != nil {
			return nil
		}
		setSessionForAddress(tr.privateKey, securedSession, senderAddr, receiverAddr, proxyAddr)
	}

	globalSessions.Set(senderAddr, receiverAddr, proxyAddr, securedSession)
	return securedSession
}

func setSessionForAddress(privatekey []byte, securedSession *session.SecuredSession, senderAddr, receiverAddr, proxyAddr string) {
	if securedSession == nil {
		securedSession, _ = session.NewSecuredSession(privatekey)
	}
	globalSessions.Set(senderAddr, receiverAddr, proxyAddr, securedSession)
	MetricSessionsRate.Inc()
	MetricSessionsCount.Set(int64(globalSessions.ItemCount()))
}

func deleteSessionForAddress(senderAddr, receiverAddr, proxyAddr string) {
	globalSessions.Delete(senderAddr, receiverAddr, proxyAddr)
}

var (
	ErrorSessionNotFound error = errors.New("session not found")
	ErrorSessionExpired  error = errors.New("session expired")
	ErrorHandshake       error = errors.New("error handshake")
)

func securityInputLayer(tr *transport, message *CoAPMessage, proxyAddr string) (isContinue bool, err error) {
	if len(proxyAddr) > 0 {
		proxyID, ok := getProxyIDIfNeed(proxyAddr)
		if ok {
			proxyAddr = fmt.Sprintf("%v%v", proxyAddr, proxyID)
		}
	}

	if ok, err := receiveHandshake(tr, tr.privateKey, message, proxyAddr); !ok {
		return false, err
	}

	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == COAPS_SCHEME {
		var addressSession string

		addressSession = message.Sender.String()

		currentSession := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addressSession, proxyAddr)

		if currentSession == nil || currentSession.AEAD == nil {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false, ErrorSessionNotFound
		}

		// Decrypt message payload
		err := decrypt(message, currentSession.AEAD)
		if err != nil {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false, ErrorSessionExpired
		}

		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(OptionSessionNotFound)
	sessionExpired := message.GetOption(OptionSessionExpired)
	if message.Code == CoapCodeUnauthorized {
		if sessionNotFound != nil {
			deleteSessionForAddress(tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
			return false, ErrorSessionNotFound
		}
		if sessionExpired != nil {
			deleteSessionForAddress(tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
			return false, ErrorSessionExpired
		}
	}

	return true, nil
}

func receiveHandshake(tr *transport, privatekey []byte, message *CoAPMessage, proxyAddr string) (isContinue bool, err error) {
	if message.IsProxies {
		return true, nil
	}
	option := message.GetOption(OptionHandshakeType)
	if option == nil {
		return true, nil
	}

	value := option.IntValue()
	if value != CoapHandshakeTypeClientSignature && value != CoapHandshakeTypeClientHello {
		return true, nil
	}

	peerSession := getSessionForAddress(tr, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)

	if value == CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		err := incomingHandshake(tr, peerSession.Curve.GetPublicKey(), message)
		if err != nil {
			return false, ErrorHandshake
		}
		if signature, err := peerSession.GetSignature(); err == nil {
			peerSession.PeerVerify(signature)
		}

	}
	MetricSuccessfulHandhshakes.Inc()

	peerSession.UpdatedAt = int(time.Now().Unix())
	setSessionForAddress(privatekey, peerSession, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)

	return false, ErrorHandshake
}

const (
	ERR_KEYS_NOT_MATCH = "Expected and current public keys do not match"
)

func handshake(tr *transport, message *CoAPMessage, address net.Addr, proxyAddr string) (*session.SecuredSession, error) {
	session := getSessionForAddress(tr, tr.conn.LocalAddr().String(), address.String(), proxyAddr)
	if session == nil {
		err := errors.New("Cannot encrypt: no session, message: %v  from: %v")
		return nil, err
	}

	// We skip handshake if session already exists
	if session.AEAD != nil {
		return session, nil
	}

	// Sending my Public Key.
	// Receiving Peer's Public Key as a Response!
	peerPublicKey, err := sendHelloFromClient(tr, message, session.Curve.GetPublicKey(), address)
	if err != nil {
		return nil, err
	}

	// assign new value
	session.PeerPublicKey = peerPublicKey

	signature, err := session.GetSignature()
	if err != nil {
		return nil, err
	}

	err = session.Verify(signature)
	if err != nil {
		return nil, err
	}

	MetricSuccessfulHandhshakes.Inc()

	return session, nil
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

	if origMessage.BreakConnectionOnPK != nil {
		if origMessage.BreakConnectionOnPK(peerPublicKey) {
			return nil, errors.New(ERR_KEYS_NOT_MATCH)
		}
	}

	return peerPublicKey, err
}

func newClientHelloMessage(origMessage *CoAPMessage, myPublicKey []byte) *CoAPMessage {
	message := NewCoAPMessage(CON, POST)
	message.AddOption(OptionHandshakeType, CoapHandshakeTypeClientHello)
	message.Payload = NewBytesPayload(myPublicKey)
	message.Token = generateToken(6)
	message.CloneOptions(origMessage, OptionProxyURI, OptionProxySecurityID)
	return message
}

func newServerHelloMessage(origMessage *CoAPMessage, publicKey []byte) *CoAPMessage {
	message := NewCoAPMessageId(ACK, CoapCodeContent, origMessage.MessageID)
	message.AddOption(OptionHandshakeType, CoapHandshakeTypePeerHello)
	message.Payload = NewBytesPayload(publicKey)
	message.Token = origMessage.Token
	message.CloneOptions(origMessage, OptionProxySecurityID)
	return message
}

func incomingHandshake(tr *transport, publicKey []byte, origMessage *CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	if _, err := tr.SendTo(message, origMessage.Sender); err != nil {
		return err
	}

	return nil
}
