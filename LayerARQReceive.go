package coalago

import (
	"github.com/coalalib/coalago/blockwise"
	m "github.com/coalalib/coalago/message"
)

func (l *layerARQ) ARQReceiveHandler(message *m.CoAPMessage) bool {
	block1 := message.GetBlock1()
	block2 := message.GetBlock2()

	windowSizeOption := message.GetOption(m.OptionSelectiveRepeatWindowSize)
	if windowSizeOption == nil {
		if block1 == nil && block2 == nil {
			return true
		}
		return false
	}

	windowSize := windowSizeOption.IntValue()

	if block1 == nil && block2 == nil {
		if message.Code == m.CoapCodeEmpty && message.Type == m.ACK {
			return l.emptyACKHandler(message)
		}

		return true
	}

	ackMessage := ackTo(message, m.CoapCodeEmpty)
	if block1 != nil {
		return l.process(message, m.OptionBlock1, block1, windowSize, ackMessage)
	}

	if block2 != nil {
		return l.process(message, m.OptionBlock2, block2, windowSize, ackMessage)
	}

	return true
}

func toACKContinueMessage(message *m.CoAPMessage) *m.CoAPMessage {
	msg := m.NewCoAPMessage(m.ACK, m.CoapCodeContinue)
	msg.CloneOptions(message, m.OptionBlock1, m.OptionBlock2, m.OptionSelectiveRepeatWindowSize, m.OptionURIScheme)
	msg.MessageID = message.MessageID
	msg.Token = message.Token
	msg.Recipient = message.Sender
	return msg
}

func ackTo(origMessage *m.CoAPMessage, code m.CoapCode) *m.CoAPMessage {
	result := m.NewCoAPMessage(m.ACK, code)
	result.MessageID = origMessage.MessageID
	result.Token = origMessage.Token
	result.CloneOptions(origMessage, m.OptionURIScheme)
	result.Recipient = origMessage.Sender
	if proxxyuri := origMessage.GetOptionProxyURIasString(); len(proxxyuri) > 0 {
		result.SetProxyURI(proxxyuri)
	}
	return result
}

func (l layerARQ) process(incomingMessage *m.CoAPMessage, blockOptionCode m.OptionCode, block *blockwise.Block, windowSize int, ackMessage *m.CoAPMessage) bool {
	switch incomingMessage.Type {
	case m.ACK:
		sendState := l.txStates.Get(incomingMessage.GetTokenString())

		if sendState != nil {
			if block.BlockNumber >= sendState.window.GetOffset()+windowSize && block.BlockNumber < sendState.window.GetOffset() {
				return false
			}
			// check in true to txState
			sendState.didTransmit(block.BlockNumber)
			l.sendMoreData(incomingMessage.GetTokenString(), windowSize)
			if incomingMessage.Code != m.CoapCodeContinue {
				origMessage := sendState.GetOriginalMessage()
				l.txStates.Delete(incomingMessage.GetTokenString())
				incomingMessage.MessageID = origMessage.MessageID

				if incomingMessage.Code == m.CoapCodeEmpty {
					return l.emptyACKHandler(incomingMessage)
				}

				return true
			}

			return false
		}

	case m.CON:
		receiveState := l.rxStates.Get(incomingMessage.GetTokenString())
		if receiveState == nil {
			receiveState = NewReceiveState(windowSize, incomingMessage)

			if block.BlockNumber >= receiveState.window.GetOffset()+windowSize && block.BlockNumber < receiveState.window.GetOffset() {
				return false
			}

			l.rxStates.Set(incomingMessage.GetTokenString(), receiveState)
		}

		receiveState.DidReceiveBlock(block.BlockNumber, block.MoreBlocks, incomingMessage.Payload.Bytes(), windowSize, incomingMessage.Code)

		ackMessage.AddOption(blockOptionCode, block.ToInt())
		ackMessage.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)

		// log.Warning("ARQ IsTransferCompleted:", receiveState.IsTransferCompleted(), receiveState.initMessage.MessageID)

		if receiveState.IsTransferCompleted() {
			if blockOptionCode == m.OptionBlock1 {
				l.rxStates.Delete(incomingMessage.GetTokenString())

				incomingMessage.Payload = m.NewBytesPayload(receiveState.GetData())
				incomingMessage.Code = receiveState.GetInitiatingMessage().Code
				if id, ok := l.emptyAcks.Load(incomingMessage.GetTokenString()); ok {
					incomingMessage.MessageID = id.(uint16)
				}
				return true

			}
			_, ok := l.emptyAcks.Load(incomingMessage.GetTokenString())
			if blockOptionCode == m.OptionBlock2 && ok {
				l.rxStates.Delete(incomingMessage.GetTokenString())

				incomingMessage.Payload = m.NewBytesPayload(receiveState.GetData())
				incomingMessage.Code = receiveState.GetInitiatingMessage().Code

				if id, ok := l.emptyAcks.Load(incomingMessage.GetTokenString()); ok {
					incomingMessage.MessageID = id.(uint16)
					l.emptyAcks.Delete(incomingMessage.GetTokenString())
				}

				l.sendARQmessage(ackMessage, ackMessage.Recipient, nil)
				return true
			}
			return false
		}

		ackMessage.Code = m.CoapCodeContinue
		l.sendARQmessage(ackMessage, ackMessage.Recipient, nil)
	}

	return false
}

func (l layerARQ) emptyACKHandler(incomingMessage *m.CoAPMessage) bool {
	l.emptyAcks.Store(incomingMessage.GetTokenString(), incomingMessage.MessageID)

	receiveState := l.rxStates.Get(incomingMessage.GetTokenString())
	if receiveState != nil {
		if receiveState.IsTransferCompleted() {
			l.rxStates.Delete(incomingMessage.GetTokenString())

			incomingMessage.Payload = m.NewBytesPayload(receiveState.GetData())
			incomingMessage.Code = receiveState.GetInitiatingMessage().Code

			if id, ok := l.emptyAcks.Load(incomingMessage.GetTokenString()); ok {
				incomingMessage.MessageID = id.(uint16)
				l.emptyAcks.Delete(incomingMessage.GetTokenString())
			}

			ackMessage := ackTo(incomingMessage, m.CoapCodeEmpty)

			l.sendARQmessage(ackMessage, ackMessage.Recipient, nil)

			incomingMessage.Code = receiveState.initMessage.Code
			return true
		}
	}
	return false
}
