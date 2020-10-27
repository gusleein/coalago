package coalago

import (
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

var StorageLocalStates = cache.New(SESSIONS_POOL_EXPIRATION, time.Second)

type LocalStateFn func(*CoAPMessage)

func MakeLocalStateFn(r Resourcer, tr *transport, respHandler func(*CoAPMessage, error)) LocalStateFn {
	var mx sync.Mutex
	storageSession := newLocalStateSessionStorageImpl()

	return func(message *CoAPMessage) {
		mx.Lock()
		defer mx.Unlock()

		_, err := localStateSecurityInputLayer(storageSession, tr, message, "")
		if err != nil {
			return
		}

		respHandler = func(message *CoAPMessage, err error) {
			if err != nil {
				return
			}
			requestOnReceive(r.getResourceForPathAndMethod(message.GetURIPath(), message.GetMethod()), storageSession, tr, message)
		}

		tr.messageHandlerSelector(storageSession, message, respHandler)
	}
}

func localStateSecurityInputLayer(storageSessions sessionStorage, tr *transport, message *CoAPMessage, proxyAddr string) (isContinue bool, err error) {
	if len(proxyAddr) > 0 {
		proxyID, ok := getProxyIDIfNeed(proxyAddr)
		if ok {
			proxyAddr = fmt.Sprintf("%v%v", proxyAddr, proxyID)
		}
	}

	if ok, err := receiveHandshake(storageSessions, tr, tr.privateKey, message, proxyAddr); !ok {
		return false, err
	}

	// Check if the message has coaps:// scheme and requires a new Session
	if message.GetScheme() == COAPS_SCHEME {
		var addressSession string

		addressSession = message.Sender.String()

		currentSession, ok := getSessionForAddress(storageSessions, tr, tr.conn.LocalAddr().String(), addressSession, proxyAddr)

		if !ok || currentSession.AEAD == nil {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionNotFound, 1)
			responseMessage.Token = message.Token
			tr.SendTo(storageSessions, responseMessage, message.Sender)
			return false, ErrorSessionNotFound
		}

		// Decrypt message payload
		err := decrypt(message, currentSession.AEAD)
		if err != nil {
			responseMessage := NewCoAPMessageId(ACK, CoapCodeUnauthorized, message.MessageID)
			responseMessage.AddOption(OptionSessionExpired, 1)
			responseMessage.Token = message.Token
			tr.SendTo(storageSessions, responseMessage, message.Sender)
			return false, ErrorSessionExpired
		}

		message.PeerPublicKey = currentSession.PeerPublicKey
	}

	/* Receive Errors */
	sessionNotFound := message.GetOption(OptionSessionNotFound)
	sessionExpired := message.GetOption(OptionSessionExpired)
	if message.Code == CoapCodeUnauthorized {
		if sessionNotFound != nil {
			deleteSessionForAddress(storageSessions, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
			return false, ErrorSessionNotFound
		}
		if sessionExpired != nil {
			deleteSessionForAddress(storageSessions, tr.conn.LocalAddr().String(), message.Sender.String(), proxyAddr)
			return false, ErrorSessionExpired
		}
	}

	return true, nil
}
