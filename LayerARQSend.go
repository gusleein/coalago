package coalago

import (
	"net"

	"github.com/coalalib/coalago/blockwise"
	m "github.com/coalalib/coalago/message"
)

func (l *layerARQ) ARQSendHandler(message *m.CoAPMessage, address net.Addr) (isContinue bool) {
	if !isBigPayload(message) {
		return true
	}
	windowSize := getWindowSize(message)

	var (
		originalMessage *m.CoAPMessage
		blockOption     m.OptionCode
	)

	switch message.Type {
	case m.ACK:
		originalMessage = message.Clone(true)
		originalMessage.Recipient = message.Recipient
		originalMessage.Type = m.CON
		originalMessage.RemoveOptions(m.OptionBlock1)

		emptyAckMessage := message
		emptyAckMessage.Code = m.CoapCodeEmpty
		emptyAckMessage.Recipient = message.Recipient
		emptyAckMessage.Payload = m.NewEmptyPayload()
		emptyAckMessage.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)

		l.sendARQmessage(emptyAckMessage, message.Recipient, nil)
		blockOption = m.OptionBlock2

	case m.CON:
		originalMessage = message
		blockOption = m.OptionBlock1
	}

	sendState := NewSendState(originalMessage.Payload.Bytes(), windowSize, blockOption, MAX_PAYLOAD_SIZE, originalMessage)
	l.txStates.Set(originalMessage.GetTokenString(), sendState)
	l.sendMoreData(originalMessage.GetTokenString(), windowSize, sendState)

	return false
}

func getWindowSize(message *m.CoAPMessage) int {
	windowSizeOption := message.GetOption(m.OptionSelectiveRepeatWindowSize)
	if windowSizeOption != nil {
		return windowSizeOption.IntValue()
	}
	return DEFAULT_WINDOW_SIZE
}

func isBigPayload(message *m.CoAPMessage) bool {
	return message.Payload != nil && message.Payload.Length() > MAX_PAYLOAD_SIZE
}

func newBlockingMessage(origMessage *m.CoAPMessage, recipient net.Addr, frame []byte, optionBlock m.OptionCode, blockNum int, windowSize int, isMore bool) *m.CoAPMessage {
	msg := m.NewCoAPMessage(m.CON, origMessage.Code)
	if origMessage.GetScheme() == m.COAPS_SCHEME {
		msg.SetSchemeCOAPS()
	}

	msg.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)
	msg.Payload = m.NewBytesPayload(frame)
	msg.SetURIPath(origMessage.GetURIPath())
	msg.Token = origMessage.Token

	queries := origMessage.GetOptions(m.OptionURIQuery)
	msg.AddOptions(queries)

	block := blockwise.NewBlock(isMore, blockNum, MAX_PAYLOAD_SIZE)

	msg.AddOption(optionBlock, block.ToInt())
	msg.Recipient = recipient

	return msg
}
