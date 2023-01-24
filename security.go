package coalago

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/coalalib/coalago/session"
	"github.com/coalalib/coalago/util"
)

func securityOutputLayer(tr *transport, message *CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != COAPS_SCHEME {
		return nil
	}

	setProxyIDIfNeed(message, tr.conn.LocalAddr().String())

	proxyAddr := message.ProxyAddr
	if len(proxyAddr) > 0 {
		proxyID, ok := getProxyIDIfNeed(proxyAddr, tr.conn.LocalAddr().String())
		if ok {
			proxyAddr = fmt.Sprintf("%v%v", proxyAddr, proxyID)
		}
	}

	currentSession, ok := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addr.String(), proxyAddr)
	if !ok {
		return ErrorClientSessionNotFound
	}

	if err := encrypt(message, addr, currentSession.AEAD); err != nil {
		return err
	}
	return nil
}

func setProxyIDIfNeed(message *CoAPMessage, senderAddr string) uint32 {
	if message.GetOption(OptionProxyURI) != nil {
		v, ok := proxyIDSessions.Get(message.ProxyAddr + senderAddr)
		if !ok {
			v = rand.Uint32()
			proxyIDSessions.Set(message.ProxyAddr+senderAddr, v)
		}
		message.AddOption(OptionProxySecurityID, v)
		return v.(uint32)
	}
	return 0
}

func getProxyIDIfNeed(proxyAddr string, senderAddr string) (uint32, bool) {
	v, ok := proxyIDSessions.Get(proxyAddr + senderAddr)
	if ok {
		return v.(uint32), ok
	}
	return 0, ok
}

func getSessionForAddress(tr *transport, senderAddr, receiverAddr, proxyAddr string) (session.SecuredSession, bool) {
	securedSession, ok := globalSessions.Get(senderAddr, receiverAddr, proxyAddr)
	if ok {
		globalSessions.Set(senderAddr, receiverAddr, proxyAddr, securedSession)
	}
	return securedSession, ok
}

func setSessionForAddress(privatekey []byte, securedSession session.SecuredSession, senderAddr, receiverAddr, proxyAddr string) {
	globalSessions.Set(senderAddr, receiverAddr, proxyAddr, securedSession)
	util.MetricSessionsRate.Inc()
	util.MetricSessionsCount.Set(int64(globalSessions.ItemCount()))
}

func deleteSessionForAddress(senderAddr, receiverAddr, proxyAddr string) {
	globalSessions.Delete(senderAddr, receiverAddr, proxyAddr)
}

func securityInputLayer(tr *transport, message *CoAPMessage, proxyAddr string) (isContinue bool, err error) {
	if len(proxyAddr) > 0 {
		proxyID, ok := getProxyIDIfNeed(proxyAddr, tr.conn.LocalAddr().String())
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

		currentSession, ok := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addressSession, proxyAddr)

		if !ok {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false, ErrorClientSessionNotFound
		}

		// Decrypt message payload
		err := decrypt(message, currentSession.AEAD)
		if err != nil {
			deleteSessionForAddress(tr.conn.LocalAddr().String(), addressSession, proxyAddr)
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false, ErrorClientSessionExpired
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
		return false, nil
	}

	peerSession, ok := getSessionForAddress(tr, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
	if !ok {
		if peerSession, err = session.NewSecuredSession(tr.privateKey); err != nil {
			return false, ErrorHandshake
		}
	}
	if value == CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		if err := incomingHandshake(tr, peerSession.Curve.GetPublicKey(), message); err != nil {
			return false, ErrorHandshake
		}
		if signature, err := peerSession.GetSignature(); err == nil {
			if err = peerSession.PeerVerify(signature); err != nil {
				return false, ErrorHandshake
			}
		}

		util.MetricSuccessfulHandhshakes.Inc()

		peerSession.UpdatedAt = int(time.Now().Unix())
		setSessionForAddress(privatekey, peerSession, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
		return false, nil
	}

	return false, ErrorHandshake
}

func handshake(tr *transport, message *CoAPMessage, address net.Addr, proxyAddr string) (session.SecuredSession, error) {
	ses, ok := getSessionForAddress(tr, tr.conn.LocalAddr().String(), address.String(), proxyAddr)
	if ok {
		return ses, nil

	}

	ses, err := session.NewSecuredSession(tr.privateKey)
	if err != nil {
		return session.SecuredSession{}, err
	}

	// Sending my Public Key.
	// Receiving Peer's Public Key as a Response!
	peerPublicKey, err := sendHelloFromClient(tr, message, ses.Curve.GetPublicKey(), address)
	if err != nil {
		return session.SecuredSession{}, err
	}

	// assign new value
	ses.PeerPublicKey = peerPublicKey

	signature, err := ses.GetSignature()
	if err != nil {
		return session.SecuredSession{}, err
	}

	err = ses.Verify(signature)
	if err != nil {
		return session.SecuredSession{}, err
	}

	globalSessions.Set(tr.conn.LocalAddr().String(), address.String(), proxyAddr, ses)
	util.MetricSuccessfulHandhshakes.Inc()

	return ses, nil
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
	message.ProxyAddr = origMessage.ProxyAddr
	return message
}

func newServerHelloMessage(origMessage *CoAPMessage, publicKey []byte) *CoAPMessage {
	message := NewCoAPMessageId(ACK, CoapCodeContent, origMessage.MessageID)
	message.AddOption(OptionHandshakeType, CoapHandshakeTypePeerHello)
	message.Payload = NewBytesPayload(publicKey)
	message.Token = origMessage.Token
	message.CloneOptions(origMessage, OptionProxySecurityID)
	message.ProxyAddr = origMessage.ProxyAddr
	return message
}

func incomingHandshake(tr *transport, publicKey []byte, origMessage *CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	if _, err := tr.SendTo(message, origMessage.Sender); err != nil {
		return err
	}

	return nil
}
