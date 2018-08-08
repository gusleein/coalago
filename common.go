package coalago

import (
	"net"

	"github.com/coalalib/coalago/blockwise"
	m "github.com/coalalib/coalago/message"
)

const (
	DEFAULT_WINDOW_SIZE = 70
)

func constructNextBlock(blockType m.OptionCode, s *stateSend) (*m.CoAPMessage, bool) {
	s.stop = s.start + MAX_PAYLOAD_SIZE
	if s.stop > s.lenght {
		s.stop = s.lenght
	}

	blockbyte := s.payload[s.start:s.stop]
	isMore := s.stop < s.lenght

	blockMessage := newBlockingMessage(
		s.origMessage,
		s.origMessage.Recipient,
		blockbyte,
		blockType,
		s.nextNumBlock,
		DEFAULT_WINDOW_SIZE,
		isMore,
	)

	s.nextNumBlock++
	s.start = s.stop

	return blockMessage, !isMore
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

func newBlockingMessage(origMessage *m.CoAPMessage, recipient net.Addr, frame []byte, optionBlock m.OptionCode, blockNum, windowSize int, isMore bool) *m.CoAPMessage {
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

type stateSend struct {
	lenght       int
	offset       int
	start        int
	stop         int
	nextNumBlock int
	origMessage  *m.CoAPMessage
	payload      []byte
}

func newACKEmptyMessage(message *m.CoAPMessage) *m.CoAPMessage {
	emptyAckMessage := m.NewCoAPMessage(m.ACK, m.CoapCodeEmpty)
	emptyAckMessage.Token = message.Token
	emptyAckMessage.MessageID = message.MessageID
	emptyAckMessage.Code = m.CoapCodeEmpty
	emptyAckMessage.Recipient = message.Recipient
	emptyAckMessage.Payload = m.NewEmptyPayload()
	emptyAckMessage.AddOption(m.OptionSelectiveRepeatWindowSize, DEFAULT_WINDOW_SIZE)
	return emptyAckMessage
}
