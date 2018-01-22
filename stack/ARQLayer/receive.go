package ARQLayer

import (
	"github.com/coalalib/coalago/common"
	"github.com/coalalib/coalago/stack/ARQLayer/blocks"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"

	m "github.com/coalalib/coalago/message"
	logging "github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("coala.ARQ")
)

const (
	MAX_PAYLOAD_SIZE    = byteBuffer.FRAME_SIZE
	DEFAULT_WINDOW_SIZE = 70
)

func OnReceive(coala common.SenderIface, message *m.CoAPMessage) bool {
	windowSize := DEFAULT_WINDOW_SIZE

	block1 := message.GetBlock1()
	block2 := message.GetBlock2()

	windowSizeOption := message.GetOption(m.OptionSelectiveRepeatWindowSize)
	if windowSizeOption != nil {
		windowSize = windowSizeOption.IntValue()
	}

	pools := coala.GetAllPools()

	switch message.Type {
	case m.CON:
		if block2 != nil {
			checkAndSetARQChan(pools, message)
			return blocks.ReceiveRequestHandler(coala, message, pools, block2, windowSize)
		}

		if block1 != nil {
			return blocks.ReceiveRequestHandler(coala, message, pools, block1, windowSize)
		}

	case m.ACK:
		if block2 != nil {
			return blocks.ReceiveACKHandler(coala, message, pools, block2)
		}

		if block1 != nil {
			if message.Code == m.CoapCodeEmpty {
				if message.GetOption(m.OptionObserve) != nil {
					return true
				}
				checkAndSetARQChan(pools, message)
				return true
			}

			return blocks.ReceiveACKHandler(coala, message, pools, block1)
		}

		if message.Code == m.CoapCodeEmpty && message.GetOption(m.OptionSelectiveRepeatWindowSize) != nil {
			checkAndSetARQChan(pools, message)
		}

	}

	return true
}
