package newcoala

import (
	"net"
	"time"

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

func (s *Server) securityOutputLayer(pc net.PacketConn, message *CoAPMessage, addr net.Addr) error {
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

func (s *Server) securityInputLayer(pc net.PacketConn, privateKey []byte, message *CoAPMessage) (isContinue bool, err error) {
	option := message.GetOption(OptionHandshakeType)
	if option != nil {
		go s.receiveHandshake(pc, privateKey, option, message)
		return false, nil
	}

	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == COAPS_SCHEME {
		currentSession, ok := s.secSessions.Get(message.Sender.String())
		if !ok {
			go func() {
				responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
				responseMessage.AddOption(OptionSessionNotFound, 1)
				responseMessage.Token = message.Token
				if b, err := Serialize(responseMessage); err == nil {
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

			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionExpired, 1)
			responseMessage.Token = message.Token

			if b, err := Serialize(responseMessage); err == nil {
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
	sessionNotFound := message.GetOption(OptionSessionNotFound)
	sessionExpired := message.GetOption(OptionSessionExpired)
	if message.Code == CoapCodeUnauthorized {
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

func (s *Server) receiveHandshake(pc net.PacketConn, privatekey []byte, option *CoAPMessageOption, message *CoAPMessage) (isContinue bool, err error) {
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

func newServerHelloMessage(origMessage *CoAPMessage, publicKey []byte) *CoAPMessage {
	message := NewCoAPMessageId(ACK, CoapCodeContent, origMessage.MessageID)
	message.AddOption(OptionHandshakeType, CoapHandshakeTypePeerHello)
	message.Payload = NewBytesPayload(publicKey)
	message.Token = origMessage.Token
	message.CloneOptions(origMessage, OptionProxySecurityID)
	message.ProxyAddr = origMessage.ProxyAddr
	return message
}

func incomingHandshake(pc net.PacketConn, publicKey []byte, origMessage *CoAPMessage) error {
	message := newServerHelloMessage(origMessage, publicKey)
	b, err := Serialize(message)
	if err != nil {
		return err
	}
	_, err = pc.WriteTo(b, origMessage.Sender)
	return err
}
