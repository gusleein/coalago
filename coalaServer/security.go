package coalaServer

import (
	"net"

	"github.com/gusleein/coalago/encription"
	cerr "github.com/gusleein/coalago/errors"
	m "github.com/gusleein/coalago/message"
	"github.com/gusleein/coalago/session"
	"github.com/gusleein/coalago/util"
)

func (s *Server) securityOutputLayer(pc net.PacketConn, message *m.CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != m.COAPS_SCHEME {
		return nil
	}
	session, ok := s.secSessions.Get(addr.String())
	if !ok {
		return cerr.ClientSessionNotFound
	}

	if err := encription.Encrypt(message, addr, session.AEAD); err != nil {
		return err
	}

	return nil
}

func (s *Server) securityInputLayer(pc net.PacketConn, privateKey []byte, message *m.CoAPMessage) (isContinue bool, err error) {
	option := message.GetOption(m.OptionHandshakeType)
	if option != nil {
		go s.receiveHandshake(pc, privateKey, option, message)
		return false, nil
	}

	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == m.COAPS_SCHEME {
		currentSession, ok := s.secSessions.Get(message.Sender.String())
		if !ok {
			go func() {
				responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
				responseMessage.AddOption(m.OptionSessionNotFound, 1)
				responseMessage.Token = message.Token
				if b, err := m.Serialize(responseMessage); err == nil {
					util.MetricSentMessages.Inc()
					pc.WriteTo(b, message.Sender)
				}
			}()
			return false, cerr.ClientSessionNotFound
		}

		// Decrypt message payload
		err := encription.Decrypt(message, currentSession.AEAD)
		if err != nil {
			s.secSessions.Delete(message.Sender.String())

			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionExpired, 1)
			responseMessage.Token = message.Token

			if b, err := m.Serialize(responseMessage); err == nil {
				util.MetricSentMessages.Inc()
				pc.WriteTo(b, message.Sender)
			}

			return false, cerr.ClientSessionExpired
		}

		s.secSessions.Update(message.Sender.String(), currentSession)

		// s.secSessions.Set(message.Sender.String(), currentSession)
		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(m.OptionSessionNotFound)
	sessionExpired := message.GetOption(m.OptionSessionExpired)
	if message.Code == m.CoapCodeUnauthorized {
		if sessionNotFound != nil {
			s.secSessions.Delete(message.Sender.String())
			return false, cerr.SessionNotFound
		}
		if sessionExpired != nil {
			s.secSessions.Delete(message.Sender.String())
			return false, cerr.SessionExpired
		}
	}

	return true, nil
}

func (s *Server) receiveHandshake(pc net.PacketConn, privatekey []byte, option *m.CoAPMessageOption, message *m.CoAPMessage) (isContinue bool, err error) {
	value := option.IntValue()
	if value != m.CoapHandshakeTypeClientSignature && value != m.CoapHandshakeTypeClientHello {
		return false, nil
	}
	peerSession, ok := s.secSessions.Get(message.Sender.String())

	if !ok {
		if peerSession, err = session.NewSecuredSession(privatekey); err != nil {
			return false, cerr.Handshake
		}
	}
	if value == m.CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		if err := incomingHandshake(pc, peerSession.Curve.GetPublicKey(), message); err != nil {
			return false, cerr.Handshake
		}

		if signature, err := peerSession.GetSignature(); err == nil {
			if err = peerSession.PeerVerify(signature); err != nil {
				return false, cerr.Handshake
			}
		}

		s.secSessions.Set(message.Sender.String(), peerSession)
		util.MetricSessionsRate.Inc()
		util.MetricSessionsCount.Set(int64(s.secSessions.seccache.ItemCount()))
		util.MetricSuccessfulHandhshakes.Inc()
		return false, nil
	}

	return false, cerr.Handshake
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

func incomingHandshake(pc net.PacketConn, publicKey []byte, origMessage *m.CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	b, err := m.Serialize(message)
	if err != nil {
		return err
	}
	_, err = pc.WriteTo(b, origMessage.Sender)
	return err
}
