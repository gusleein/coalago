package coalago

import (
	"errors"
	"net"

	m "github.com/coalalib/coalago/message"
)

type SecurityLayer struct{}

func (layer *SecurityLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == m.COAPS_SCHEME {
		currentSession := coala.GetSessionForAddress(message.Sender)

		if currentSession == nil || currentSession.AEAD == nil {
			log.Error("Cannot decrypt Message: NO session", message.Sender, message.ToReadableString())
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionNotFound, 1)
			// if _, err := coala.Send(responseMessage, message.Sender); err != nil {
			// 	log.Error(err, message.ToReadableString())
			// }
			sendToSocket(coala, message, message.Sender)
			return false
		}

		// Decrypt message payload
		err := Decrypt(message, currentSession.AEAD)
		if err != nil {
			log.Error("Cannot decrypt Message, error occured: ", err, message.ToReadableString())
			responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(m.OptionSessionExpired, 1)
			// if _, err := coala.Send(responseMessage, message.Sender); err != nil {
			// 	log.Error("SecurityLayer", "OnReceive", err, message)
			// }
			sendToSocket(coala, responseMessage, message.Sender)

			return false
		}
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(m.OptionSessionNotFound)
	sessionExpired := message.GetOption(m.OptionSessionExpired)
	if message.Code == m.CoapCodeUnauthorized && (sessionNotFound != nil || sessionExpired != nil) {
		coala.GetAllPools().Sessions.Delete(message.Sender.String())
		return false
	}

	return true
}

func (layer *SecurityLayer) OnSend(coala *Coala, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	if message.IsProxies {
		return true, nil
	}
	// Check if the message has coaps:// scheme and requires Encryption

	if message.GetScheme() != m.COAPS_SCHEME {
		return true, nil
	}

	key := message.GetMessageIDString() + message.GetTokenString() + address.String()

	if !coala.GetAllPools().ExpectedHandshakePool.IsEmpty(key) {
		if err := coala.GetAllPools().ExpectedHandshakePool.Set(key, message); err != nil {
			log.Error("Error adding message to the handshake pool. Error: ", err, message.ToReadableString())
			return false, err
		}

	}

	currentSession := coala.GetSessionForAddress(address)

	if currentSession == nil {
		err := errors.New("Cannot encrypt: no session, message: %v  from: %v")
		log.Errorf(err.Error(), message.ToReadableString(), message.Sender.String())
		return false, err
	}

	// Perform the Handshake (if necessary)
	err := handshake(coala, message, currentSession, address)

	if err != nil {
		log.Error(err, message.ToReadableString(), message.Sender.String(), currentSession)
		coala.GetAllPools().ExpectedHandshakePool.Delete(key)
		return false, err
	}

	// Encrypt message payload
	err = Encrypt(message, address, currentSession.AEAD)
	if err != nil {
		log.Error(err, message.ToReadableString(), message.Sender.String(), currentSession)
		coala.GetAllPools().ExpectedHandshakePool.Delete(key)
		return false, err
	}

	for {
		msg := coala.GetAllPools().ExpectedHandshakePool.Pull(key)
		if msg == nil {
			coala.GetAllPools().ExpectedHandshakePool.Delete(key)
			break
		}
	}

	return true, nil
}
