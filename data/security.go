package newcoala

import (
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/session"
	"github.com/coalalib/coalago/util"
	"github.com/patrickmn/go-cache"
)

const (
	sessionLifetime = time.Minute*4 + time.Second*9
)

type sessionState struct {
	key string
	est time.Time
}

type securitySessionStorage struct {
	// rwmx sync.RWMutex
	// m       map[string]session.SecuredSession
	// indexes map[string]time.Time
	// est     []sessionState
	seccache *cache.Cache
}

func newSecuritySessionStorage() *securitySessionStorage {
	s := &securitySessionStorage{
		seccache: cache.New(sessionLifetime, time.Second),
	}

	return s
}

func (s *securitySessionStorage) Set(k string, v session.SecuredSession) {
	s.seccache.SetDefault(k, v)
}

func (s *securitySessionStorage) Delete(k string) {
	s.seccache.Delete(k)
}

func (s *securitySessionStorage) Update(k string, sess session.SecuredSession) {
	s.seccache.SetDefault(k, sess)
}

func (s *securitySessionStorage) Get(k string) (sess session.SecuredSession, ok bool) {
	v, ok := s.seccache.Get(k)
	if !ok {
		return sess, ok
	}
	sess = v.(session.SecuredSession)
	return sess, ok
}

func (s *Server) securityOutputLayer(pc net.PacketConn, message *m.CoAPMessage, addr net.Addr) error {
	if message.GetScheme() != COAPS_SCHEME {
		return nil
	}
	session, ok := s.secSessions.Get(addr.String())
	if !ok {
		return ErrorClientSessionNotFound
	}

	if err := encrypt(message, addr, session.AEAD); err != nil {
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
	if message.GetScheme() == COAPS_SCHEME {
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
			return false, ErrorClientSessionNotFound
		}

		// Decrypt message payload
		err := decrypt(message, currentSession.AEAD)
		if err != nil {
			s.secSessions.Delete(message.Sender.String())

			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionExpired, 1)
			responseMessage.Token = message.Token

			if b, err := m.Serialize(responseMessage); err == nil {
				util.MetricSentMessages.Inc()
				pc.WriteTo(b, message.Sender)
			}

			return false, ErrorClientSessionExpired
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
			return false, ErrorSessionNotFound
		}
		if sessionExpired != nil {
			s.secSessions.Delete(message.Sender.String())
			return false, ErrorSessionExpired
		}
	}

	return true, nil
}

func (s *Server) receiveHandshake(pc net.PacketConn, privatekey []byte, option *m.CoAPMessageOption, message *m.CoAPMessage) (isContinue bool, err error) {
	value := option.IntValue()
	if value != CoapHandshakeTypeClientSignature && value != CoapHandshakeTypeClientHello {
		return false, nil
	}
	peerSession, ok := s.secSessions.Get(message.Sender.String())

	if !ok {
		if peerSession, err = session.NewSecuredSession(privatekey); err != nil {
			return false, ErrorHandshake
		}
	}
	if value == CoapHandshakeTypeClientHello && message.Payload != nil {
		peerSession.PeerPublicKey = message.Payload.Bytes()

		if err := incomingHandshake(pc, peerSession.Curve.GetPublicKey(), message); err != nil {
			return false, ErrorHandshake
		}

		if signature, err := peerSession.GetSignature(); err == nil {
			if err = peerSession.PeerVerify(signature); err != nil {
				return false, ErrorHandshake
			}
		}

		s.secSessions.Set(message.Sender.String(), peerSession)
		util.MetricSessionsRate.Inc()
		// MetricSessionsCount.Set(int64(len(s.secSessions.m)))
		util.MetricSessionsCount.Set(int64(s.secSessions.seccache.ItemCount()))

		util.MetricSuccessfulHandhshakes.Inc()
		return false, nil
	}

	return false, ErrorHandshake
}

const (
	ERR_KEYS_NOT_MATCH = "Expected and current public keys do not match"
)

func newServerHelloMessage(origMessage *m.CoAPMessage, publicKey []byte) *m.CoAPMessage {
	message := m.NewCoAPMessageId(m.ACK, m.CoapCodeContent, origMessage.MessageID)
	message.AddOption(m.OptionHandshakeType, CoapHandshakeTypePeerHello)
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
