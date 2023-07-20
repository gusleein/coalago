package coalago

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/gusleein/coalago/encription"
	cerr "github.com/gusleein/coalago/errors"
	m "github.com/gusleein/coalago/message"
	"github.com/gusleein/coalago/session"
	"github.com/gusleein/coalago/util"
)

func securityOutputLayer(tr *transport, message *m.CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != m.COAPS_SCHEME {
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
		return cerr.ClientSessionNotFound
	}

	if err := encription.Encrypt(message, addr, currentSession.AEAD); err != nil {
		return err
	}
	return nil
}

func setProxyIDIfNeed(message *m.CoAPMessage, senderAddr string) uint32 {
	if message.GetOption(m.OptionProxyURI) != nil {
		v, ok := proxyIDSessions.Get(message.ProxyAddr + senderAddr)
		if !ok {
			v = rand.Uint32()
			proxyIDSessions.Set(message.ProxyAddr+senderAddr, v)
		}
		message.AddOption(m.OptionProxySecurityID, v)
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

func securityInputLayer(tr *transport, message *m.CoAPMessage, proxyAddr string) (isContinue bool, err error) {
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
	if message.GetScheme() == m.COAPS_SCHEME {
		var addressSession string

		addressSession = message.Sender.String()

		currentSession, ok := getSessionForAddress(tr, tr.conn.LocalAddr().String(), addressSession, proxyAddr)

		if !ok {
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false, cerr.ClientSessionNotFound
		}

		// Decrypt message payload
		err := encription.Decrypt(message, currentSession.AEAD)
		if err != nil {
			deleteSessionForAddress(tr.conn.LocalAddr().String(), addressSession, proxyAddr)
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			tr.SendTo(responseMessage, message.Sender)
			return false, cerr.ClientSessionExpired
		}

		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(m.OptionSessionNotFound)
	sessionExpired := message.GetOption(m.OptionSessionExpired)
	if message.Code == m.CoapCodeUnauthorized {
		if sessionNotFound != nil {
			deleteSessionForAddress(tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
			return false, cerr.SessionNotFound
		}
		if sessionExpired != nil {
			deleteSessionForAddress(tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
			return false, cerr.SessionExpired
		}
	}

	return true, nil
}

func receiveHandshake(tr *transport, privatekey []byte, message *m.CoAPMessage, proxyAddr string) (isContinue bool, err error) {
	if message.IsProxies {
		return true, nil
	}
	option := message.GetOption(m.OptionHandshakeType)
	if option == nil {
		return true, nil
	}

	value := option.IntValue()
	if value != m.CoapHandshakeTypeClientSignature && value != m.CoapHandshakeTypeClientHello {
		return false, nil
	}

	peerSession, ok := getSessionForAddress(tr, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
	if !ok {
		if peerSession, err = session.NewSecuredSession(tr.privateKey); err != nil {
			return false, cerr.Handshake
		}
	}
	if value == m.CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		if err := incomingHandshake(tr, peerSession.Curve.GetPublicKey(), message); err != nil {
			return false, cerr.Handshake
		}
		if signature, err := peerSession.GetSignature(); err == nil {
			if err = peerSession.PeerVerify(signature); err != nil {
				return false, cerr.Handshake
			}
		}

		util.MetricSuccessfulHandhshakes.Inc()

		peerSession.UpdatedAt = int(time.Now().Unix())
		setSessionForAddress(privatekey, peerSession, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
		return false, nil
	}

	return false, cerr.Handshake
}

func handshake(tr *transport, message *m.CoAPMessage, address net.Addr, proxyAddr string) (session.SecuredSession, error) {
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

	if origMessage.BreakConnectionOnPK != nil {
		if origMessage.BreakConnectionOnPK(peerPublicKey) {
			return nil, errors.New(cerr.ERR_KEYS_NOT_MATCH)
		}
	}

	return peerPublicKey, err
}

func newClientHelloMessage(origMessage *m.CoAPMessage, myPublicKey []byte) *m.CoAPMessage {
	message := m.NewCoAPMessage(m.CON, m.POST)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypeClientHello)
	message.Payload = m.NewBytesPayload(myPublicKey)
	message.Token = m.GenerateToken(6)
	message.CloneOptions(origMessage, m.OptionProxyURI, m.OptionProxySecurityID)
	message.ProxyAddr = origMessage.ProxyAddr
	return message
}

func newServerHelloMessage(origMessage *m.CoAPMessage, publicKey []byte) *m.CoAPMessage {
	message := m.NewCoAPMessageId(m.ACK, m.CoapCodeContent, origMessage.MessageID)
	message.AddOption(m.OptionHandshakeType, m.CoapHandshakeTypePeerHello)
	message.Payload = m.NewBytesPayload(publicKey)
	message.Token = origMessage.Token
	message.CloneOptions(origMessage, m.OptionProxySecurityID)
	message.ProxyAddr = origMessage.ProxyAddr
	return message
}

func incomingHandshake(tr *transport, publicKey []byte, origMessage *m.CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	if _, err := tr.SendTo(message, origMessage.Sender); err != nil {
		return err
	}

	return nil
}
