package coalago

import (
	"errors"
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/network/session"
)

var (
	SESSIONS_POOL_EXPIRATION = time.Second * 60 * 5
)

type SecurityLayer struct {
}

func newSecurityLayer() *SecurityLayer {
	return &SecurityLayer{}
}
func (layer *SecurityLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if !layer.receiveHandshake(coala, message) {
		return false
	}
	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == m.COAPS_SCHEME {
		var addressSession string

		a, _ := coala.ProxySessions.Get(message.Sender.String())

		if a == nil {
			addressSession = message.Sender.String()
		} else {
			addressSession = a.(string)
		}

		currentSession := layer.GetSessionForAddress(coala, addressSession)

		if currentSession == nil || currentSession.AEAD == nil {
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			sendToSocket(coala, responseMessage, message.Sender)
			return false
		}

		// Decrypt message payload
		err := Decrypt(message, currentSession.AEAD)
		if err != nil {
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			sendToSocket(coala, responseMessage, message.Sender)

			return false
		}

		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(m.OptionSessionNotFound)
	sessionExpired := message.GetOption(m.OptionSessionExpired)
	if message.Code == m.CoapCodeUnauthorized && (sessionNotFound != nil || sessionExpired != nil) {
		coala.Sessions.Delete(message.Sender.String())
		return false
	}

	return true
}

func (layer *SecurityLayer) receiveHandshake(coala *Coala, message *m.CoAPMessage) bool {
	if message.IsProxies {
		return true
	}
	option := message.GetOption(m.OptionHandshakeType)
	if option == nil {
		return true
	}

	value := option.IntValue()
	if value != m.CoapHandshakeTypeClientSignature && value != m.CoapHandshakeTypeClientHello {
		coala.GetMetrics().SuccessfulHandshakes.Inc()
		return true
	}

	peerSession := layer.GetSessionForAddress(coala, message.Sender.String())

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
	coala.GetMetrics().SuccessfulHandshakes.Inc()

	peerSession.UpdatedAt = int(time.Now().Unix())
	layer.SetSessionForAddress(coala, peerSession, message.Sender.String())

	return false
}

func (layer *SecurityLayer) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	if message.IsProxies {
		return true, nil
	}
	// Check if the message has coaps:// scheme and requires Encryption

	if message.GetScheme() != m.COAPS_SCHEME {
		return true, nil
	}

	var (
		addressSession string
		err            error
	)
	a := message.GetOptionProxyURIasString()
	if len(a) > 0 {
		addressSession = a
		coala.ProxySessions.SetDefault(address.String(), a)
	} else {
		addressSession = address.String()
	}

	currentSession := layer.GetSessionForAddress(coala, addressSession)

	if currentSession == nil {
		err := errors.New("Cannot encrypt: no session, message: %v  from: %v")
		log.Errorf(err.Error(), message.ToReadableString(), message.Sender.String())
		return false, err
	}

	// Perform the Handshake (if necessary)
	err = handshake(coala, message, currentSession, address)
	if err != nil {
		log.Error(err, message.ToReadableString(), currentSession)
		return false, err
	}

	// Encrypt message payload
	err = Encrypt(message, address, currentSession.AEAD)
	if err != nil {
		log.Error(err, message.ToReadableString(), message.Sender.String(), currentSession)
		return false, err
	}

	return true, nil
}

func (layer *SecurityLayer) GetSessionForAddress(coala *Coala, udpAddr string) *session.SecuredSession {
	s, _ := coala.Sessions.Get(udpAddr)
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
		securedSession, err = session.NewSecuredSession(coala.GetPrivateKey())
		if err != nil {
			log.Error(err)
			return nil
		}
		coala.Metrics.SessionsRate.Inc()
		layer.SetSessionForAddress(coala, securedSession, udpAddr)
	}

	return securedSession
}

func (layer *SecurityLayer) SetSessionForAddress(coala *Coala, securedSession *session.SecuredSession, udpAddr string) {
	if securedSession == nil {
		securedSession, _ = session.NewSecuredSession(coala.privatekey)
	}
	coala.Sessions.SetDefault(udpAddr, securedSession)
	coala.Metrics.Sessions.Set(int64(coala.Sessions.ItemCount()))
}
