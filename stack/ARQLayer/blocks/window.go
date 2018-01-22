package blocks

import (
	"github.com/coalalib/coalago/stack/ARQLayer/blockwise"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"

	m "github.com/coalalib/coalago/message"
)

const (
	MAX_PAYLOAD_SIZE = byteBuffer.FRAME_SIZE
)

func GetLastElementOfWindow(origMessage *m.CoAPMessage, buf *byteBuffer.Buffer, windowSize int, optionBlock m.OptionCode) (msg *m.CoAPMessage, isNext bool) {
	numFrame, frame, isNext := buf.NextFrame()
	if frame == nil {
		return msg, false
	}

	msg = m.NewCoAPMessage(m.CON, origMessage.Code)
	if origMessage.GetScheme() == m.COAPS_SCHEME {
		msg.SetSchemeCOAPS()
	}

	if proxyURI := origMessage.GetOptionProxyURIasString(); proxyURI != "" {
		msg.SetProxyURI(proxyURI)
	}

	msg.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)
	msg.SetStringPayload(string(frame.Body))
	msg.SetURIPath(origMessage.GetURIPath())
	msg.Token = origMessage.Token
	msg.Attempts = 3
	if obsrv := origMessage.GetOption(m.OptionObserve); obsrv != nil {
		msg.AddOption(m.OptionObserve, obsrv.Value)
		maxAge := origMessage.GetOption(m.OptionMaxAge)
		if maxAge != nil {
			msg.AddOption(m.OptionMaxAge, maxAge.Value)
		}

	}

	queries := origMessage.GetOptions(m.OptionURIQuery)
	msg.AddOptions(queries)

	block := blockwise.NewBlock(false, numFrame, MAX_PAYLOAD_SIZE)

	if isNext {
		block.MoreBlocks = true
	} else {
		block.MoreBlocks = false
	}

	msg.AddOption(optionBlock, block.ToInt())
	return msg, isNext
}

func GetFirstWindow(origMessage *m.CoAPMessage, buf *byteBuffer.Buffer, windowSize int, optionBlock m.OptionCode) (windowMesssages []*m.CoAPMessage, isNext bool) {
	for i := 0; i < windowSize; i++ {
		var msg *m.CoAPMessage
		msg, isNext = GetLastElementOfWindow(origMessage, buf, windowSize, optionBlock)
		if msg == nil {
			break
		}

		windowMesssages = append(windowMesssages, msg)

		if !isNext {
			break
		}
	}

	return windowMesssages, isNext
}

func ReplaceElemenetsIntoWindow(oldPool []*m.CoAPMessage, lastMsg *m.CoAPMessage, optionBlock m.OptionCode) []*m.CoAPMessage {
	if len(oldPool) != 0 {
		oldPool = append(oldPool[:0], oldPool[1:]...)
	}

	var blockOld, blockNew *blockwise.Block

	if len(oldPool) != 0 {
		blockOld, blockNew = getBlocks(oldPool[len(oldPool)-1], lastMsg, optionBlock)
	} else {
		oldPool = append(oldPool, lastMsg)
	}

	if blockOld != nil && blockNew != nil {
		if blockOld.BlockNumber != blockNew.BlockNumber {
			oldPool = append(oldPool, lastMsg)
		}
	}

	for _, msg := range oldPool {
		msg.Attempts = 3
	}

	return oldPool
}

func getBlocks(oldMessage, newMessage *m.CoAPMessage, optionBlock m.OptionCode) (oldBlock, newBlock *blockwise.Block) {
	if optionBlock == m.OptionBlock1 {
		newBlock = newMessage.GetBlock1()
		oldBlock = oldMessage.GetBlock1()
	} else if optionBlock == m.OptionBlock2 {
		oldBlock = oldMessage.GetBlock2()
		newBlock = newMessage.GetBlock2()
	}

	return
}
