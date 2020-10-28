package coalago

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/patrickmn/go-cache"
)

var StorageLocalStates = cache.New(sumTimeAttempts, time.Second)

type LocalStateFn func(*CoAPMessage)

func MakeLocalStateFn(r Resourcer, tr *transport, respHandler func(*CoAPMessage, error), closeCallback func()) LocalStateFn {
	var mx sync.Mutex
	// storageSession := newLocalStateSessionStorageImpl()

	var bufBlock1 = make(map[int][]byte)
	var totalBlocks1 = -1
	var runnedHandler int32 = 0

	return func(message *CoAPMessage) {
		mx.Lock()
		defer mx.Unlock()

		if _, err := localStateSecurityInputLayer(globalSessions, tr, message, ""); err != nil {
			return
		}

		respHandler = func(message *CoAPMessage, err error) {
			if atomic.LoadInt32(&runnedHandler) == 1 {
				return
			}
			atomic.StoreInt32(&runnedHandler, 1)

			if err != nil {
				return
			}

			requestOnReceive(r.getResourceForPathAndMethod(message.GetURIPath(), message.GetMethod()), globalSessions, tr, message)
			closeCallback()
		}

		totalBlocks1, bufBlock1 = localStateMessageHandlerSelector(tr, totalBlocks1, bufBlock1, globalSessions, message, respHandler)
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

		if !ok {
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

func localStateMessageHandlerSelector(
	sr *transport,
	totalBlocks int,
	buffer map[int][]byte,
	storageSessions sessionStorage,
	message *CoAPMessage,
	respHandler func(*CoAPMessage, error),
) (
	int, map[int][]byte,
) {
	block1 := message.GetBlock1()
	block2 := message.GetBlock2()

	if block1 != nil {
		if message.Type == CON {
			var (
				ok  bool
				err error
			)
			ok, totalBlocks, buffer, message, err = localStateReceiveARQBlock1(sr, totalBlocks, buffer, storageSessions, message)
			if ok {
				go respHandler(message, err)
			}
		}
		return totalBlocks, buffer
	}

	if block2 != nil {
		if message.Type == ACK {
			id := message.Sender.String() + string(message.Token)

			c, ok := sr.block2channels.Load(id)
			if ok {
				c.(chan *CoAPMessage) <- message
			}
		}
		return totalBlocks, buffer
	}
	go respHandler(message, nil)
	return totalBlocks, buffer
}

func localStateReceiveARQBlock1(sr *transport, totalBlocks int, buf map[int][]byte, storageSessions sessionStorage, inputMessage *CoAPMessage) (bool, int, map[int][]byte, *CoAPMessage, error) {
	block := inputMessage.GetBlock1()
	if block == nil || inputMessage.Type != CON {
		return false, totalBlocks, buf, inputMessage, nil
	}
	if !block.MoreBlocks {
		totalBlocks = block.BlockNumber + 1
	}

	buf[block.BlockNumber] = inputMessage.Payload.Bytes()
	if totalBlocks == len(buf) {
		b := []byte{}
		for i := 0; i < totalBlocks; i++ {
			b = append(b, buf[i]...)
		}
		inputMessage.Payload = NewBytesPayload(b)

		return true, totalBlocks, buf, inputMessage, nil
	}

	var ack *CoAPMessage
	w := inputMessage.GetOption(OptionSelectiveRepeatWindowSize)
	if w != nil {
		ack = ackToWithWindowOffset(nil, inputMessage, CoapCodeContinue, w.IntValue(), block.BlockNumber, buf)
	} else {
		ack = ackTo(nil, inputMessage, CoapCodeContinue)
	}

	if err := sr.sendToSocketByAddress(storageSessions, ack, inputMessage.Sender); err != nil {
		return false, totalBlocks, buf, inputMessage, err
	}

	return false, totalBlocks, buf, inputMessage, nil
}
