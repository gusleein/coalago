package ARQLayer

import (
	"net"

	"github.com/coalalib/coalago/common"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/ARQLayer/blocks"
)

func ARQSendHandler(coala common.SenderIface, message *m.CoAPMessage, address *net.UDPAddr) (response *m.CoAPMessage, isContinue bool) {
	if !isBigPayload(message) {
		return nil, true
	}

	windowSize := DEFAULT_WINDOW_SIZE

	windowSizeOption := message.GetOption(m.OptionSelectiveRepeatWindowSize)
	if windowSizeOption != nil {
		windowSize = windowSizeOption.IntValue()
	}

	switch message.Type {
	case m.ACK:
		ackMsg := emptyACKmessage(message, windowSize)
		coala.Send(ackMsg, address)
		message.MessageID = m.GenerateMessageID()
		go blocks.SenderHandler(coala, message, windowSize, m.OptionBlock2, address)
		return nil, false
	case m.CON:
		return blocks.SenderHandler(coala, message, windowSize, m.OptionBlock1, address)
	}

	return nil, false
}
