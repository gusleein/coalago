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
	receiveState := l.rxStates.Get(message.GetTokenString())

	if block1 == nil && block2 == nil {
		if message.Code == m.CoapCodeEmpty && message.Type == m.ACK {
			return l.emptyACKHandler(message, receiveState)
		}

		return true
	}

	ackMessage := ackTo(message, m.CoapCodeEmpty)
	sendState := l.txStates.Get(message.GetTokenString())

	if block1 != nil {
		return l.process(message, m.OptionBlock1, block1, windowSize, ackMessage, sendState, receiveState)
	}

	if block2 != nil {
		return l.process(message, m.OptionBlock2, block2, windowSize, ackMessage, sendState, receiveState)
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

func (l layerARQ) process(incomingMessage *m.CoAPMessage, blockOptionCode m.OptionCode, block *blockwise.Block, windowSize int, ackMessage *m.CoAPMessage, sendState *ARQState, receiveState *ARQState) bool {
	switch incomingMessage.Type {
	case m.ACK:
		return l.receiveACK(incomingMessage, blockOptionCode, block, windowSize, ackMessage, sendState, receiveState)
	case m.CON:
		return l.receiveCON(incomingMessage, blockOptionCode, block, windowSize, ackMessage, receiveState)
	}

	return false
}

func (l layerARQ) receiveACK(
	incomingMessage *m.CoAPMessage,
	blockOptionCode m.OptionCode,
	block *blockwise.Block,
	windowSize int,
	ackMessage *m.CoAPMessage,
	sendState *ARQState,
	receiveState *ARQState,
) bool {
	if sendState != nil {
		if !isWithinWindow(windowSize, block.BlockNumber, sendState.window.GetOffset()) {
			return false
		}

		// check in true to txState
		sendState.didTransmit(block.BlockNumber)
		l.sendMoreData(incomingMessage.GetTokenString(), windowSize, sendState)
		if incomingMessage.Code != m.CoapCodeContinue {
			origMessage := sendState.GetOriginalMessage()
			l.txStates.Delete(incomingMessage.GetTokenString())

			incomingMessage.MessageID = origMessage.MessageID

			if incomingMessage.Code == m.CoapCodeEmpty {
				return l.emptyACKHandler(incomingMessage, receiveState)
			}

			return true
		}
	}

	return false
}

func (l layerARQ) receiveCON(incomingMessage *m.CoAPMessage, blockOptionCode m.OptionCode, block *blockwise.Block, windowSize int, ackMessage *m.CoAPMessage, receiveState *ARQState) bool {
	if receiveState == nil {
		receiveState = NewReceiveState(windowSize, incomingMessage)

		if !isWithinWindow(windowSize, block.BlockNumber, receiveState.window.GetOffset()) {
			return false
		}

		l.rxStates.Set(incomingMessage.GetTokenString(), receiveState)
	}

	receiveState.DidReceiveBlock(block.BlockNumber, block.MoreBlocks, incomingMessage.Payload.Bytes(), windowSize, incomingMessage.Code)

	if receiveState.IsTransferCompleted() {
		switch blockOptionCode {
		case m.OptionBlock1:
			l.processBlock1ReceiveCompleted(incomingMessage, receiveState)
		case m.OptionBlock2:
			if _, ok := l.emptyAcks.Load(incomingMessage.GetTokenString()); ok {
				l.processBlock2ReceiveCompleted(incomingMessage, ackMessage, receiveState)
			} else {
				return false
			}
		}

		return true

	}

	ackMessage.AddOption(blockOptionCode, block.ToInt())
	ackMessage.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)

	ackMessage.Code = m.CoapCodeContinue
	l.sendARQmessage(ackMessage, ackMessage.Recipient, nil)
	return false
}

func isWithinWindow(windowSize int, blockNum int, windowOffset int) bool {
	return blockNum >= windowOffset && blockNum < windowOffset+windowSize
}

func (l layerARQ) processBlock1ReceiveCompleted(incomingMessage *m.CoAPMessage, receiveState *ARQState) {
	l.txStates.Delete(incomingMessage.GetTokenString())
	l.rxStates.Delete(incomingMessage.GetTokenString())

	incomingMessage.Payload = m.NewBytesPayload(receiveState.GetData())
	incomingMessage.Code = receiveState.GetInitiatingMessage().Code
	if id, ok := l.emptyAcks.Load(incomingMessage.GetTokenString()); ok {
		incomingMessage.MessageID = id.(uint16)
	}
}

func (l layerARQ) processBlock2ReceiveCompleted(incomingMessage *m.CoAPMessage, ackMessage *m.CoAPMessage, receiveState *ARQState) {
	l.rxStates.Delete(incomingMessage.GetTokenString())

	incomingMessage.Payload = m.NewBytesPayload(receiveState.GetData())
	incomingMessage.Code = receiveState.GetInitiatingMessage().Code

	if id, ok := l.emptyAcks.Load(incomingMessage.GetTokenString()); ok {
		incomingMessage.MessageID = id.(uint16)
		l.emptyAcks.Delete(incomingMessage.GetTokenString())
	}

	l.sendARQmessage(ackMessage, ackMessage.Recipient, nil)
}

func (l layerARQ) emptyACKHandler(incomingMessage *m.CoAPMessage, receiveState *ARQState) bool {
	l.emptyAcks.Store(incomingMessage.GetTokenString(), incomingMessage.MessageID)

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
