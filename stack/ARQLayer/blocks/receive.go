package blocks

import (
	"net"
	"sync"

	"github.com/coalalib/coalago/common"
	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/pools"
	"github.com/coalalib/coalago/stack/ARQLayer/blockwise"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
)

var (
	lock sync.Mutex
)

var count int

func ReceiveRequestHandler(coala common.SenderIface, message *m.CoAPMessage, pools *pools.AllPools, block *blockwise.Block, windowSize int) bool {
	lock.Lock()
	if pools.ARQBuffers.Get(getBufferKeyForReceive(message)) == nil {
		pools.ARQBuffers.Set(getBufferKeyForReceive(message), byteBuffer.NewBuffer())
		if message.GetBlock2() != nil {
			if arqRespChan := pools.ARQRespMessages.Get(getBufferKeyForReceive(message)); arqRespChan == nil {
				arqRespChan = make(chan *byteBuffer.ARQResponse)
				pools.ARQRespMessages.Set(getBufferKeyForReceive(message), arqRespChan)
			}
		}
	}
	lock.Unlock()

	buf := pools.ARQBuffers.Get(getBufferKeyForReceive(message))
	if buf == nil {
		return false
	}

	buffer := buf.(*byteBuffer.Buffer)

	if block.MoreBlocks == false {
		buffer.SetLastBlock(block.BlockNumber)
	}

	isFull, err := buffer.WriteToWindow(windowSize, block.BlockNumber, message.Payload.Bytes())
	if err != nil {
		return false
	}

	if isFull {
		pools.ARQBuffers.Delete(getBufferKeyForReceive(message))
		message.Payload = m.NewBytesPayload(buffer.Byte())

		if message.GetBlock2() != nil {
			sendACK(coala, message, block, m.CoapCodeEmpty, windowSize, message.Sender)
			chanResponse := pools.ARQRespMessages.Get(getBufferKeyForReceive(message))
			resp := &byteBuffer.ARQResponse{
				Message: message,
				IsNext:  false,
			}
			count++

			chanResponse <- resp

			pools.ARQRespMessages.Delete(getBufferKeyForReceive(message))
			return true
		}
		return true
	}

	sendACK(coala, message, block, m.CoapCodeContinue, windowSize, message.Sender)

	return false
}

func ReceiveACKHandler(coala common.SenderIface, message *m.CoAPMessage, pools *pools.AllPools, block *blockwise.Block) bool {
	buf := pools.ARQBuffers.Get(getBufferKeyForReceive(message))
	if buf != nil {
		buffer := buf.(*byteBuffer.Buffer)
		buffer.AddACK(block.BlockNumber)
	}

	return false
}

func sendACK(coala common.SenderIface, message *m.CoAPMessage, block *blockwise.Block, code m.CoapCode, windowSize int, address *net.UDPAddr) error {
	nextBlockMessage := m.NewCoAPMessage(m.ACK, code)
	if message.GetScheme() == m.COAPS_SCHEME {
		nextBlockMessage.SetSchemeCOAPS()
	}

	nextBlockMessage.MessageID = message.MessageID
	nextBlockMessage.Token = message.Token

	var optionBlock m.OptionCode
	if message.GetBlock1() != nil {
		optionBlock = m.OptionBlock1
	} else {
		optionBlock = m.OptionBlock2
	}

	nextBlockMessage.AddOption(optionBlock, block.ToInt())
	nextBlockMessage.AddOption(m.OptionSelectiveRepeatWindowSize, windowSize)

	if obsrv := message.GetOption(m.OptionObserve); obsrv != nil {
		nextBlockMessage.AddOption(m.OptionObserve, obsrv.Value)
		maxAge := message.GetOption(m.OptionMaxAge)
		if maxAge != nil {
			nextBlockMessage.AddOption(m.OptionMaxAge, maxAge.Value)
		}
	}

	origMsg := coala.GetAllPools().SendMessages.Get(message.GetTokenString() + address.String())
	if origMsg != nil {
		nextBlockMessage.CloneOptions(origMsg, m.OptionProxyURI)
	}

	_, err := coala.Send(nextBlockMessage, address)
	return err
}
