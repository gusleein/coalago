package blocks

import (
	"errors"
	"net"
	"sync"

	"github.com/coalalib/coalago/common"

	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
	logging "github.com/op/go-logging"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/pools"
)

var (
	log = logging.MustGetLogger("coala.ARQ")
)

func SenderHandler(coala common.SenderIface, message *m.CoAPMessage, windowSize int, optionBlock m.OptionCode, address *net.UDPAddr) (response *m.CoAPMessage, isContinue bool) {
	var (
		windowMessages []*m.CoAPMessage
	)

	pools := coala.GetAllPools()

	buffer := byteBuffer.NewBuffer()
	buffer.Write(message.Payload.Bytes())
	pools.ARQBuffers.Set(getBufferKeyForSend(message, address), buffer)

	for {
		windowMessages, _ := getWindow(buffer, windowMessages, message, windowSize, optionBlock)
		if len(windowMessages) == 0 {
			return
		}
		next, isFull, lastMessage, err := processSending(coala, pools, buffer, windowMessages, address)

		if err != nil {
			return nil, false
		}
		if lastMessage != nil {
			response = lastMessage
		}
		if isFull {
			break
		}
		if !next {
			break
		}
	}

	return response, false
}

func getWindow(buf *byteBuffer.Buffer, currentWindow []*m.CoAPMessage, origMessage *m.CoAPMessage, windowSize int, optionBlock m.OptionCode) ([]*m.CoAPMessage, bool) {
	if buf.IsFirstWindow() {
		currentWindow, isNext := GetFirstWindow(origMessage, buf, windowSize, optionBlock)
		return currentWindow, isNext
	}

	lastMsg, isNext := GetLastElementOfWindow(origMessage, buf, windowSize, optionBlock)
	if lastMsg != nil {
		currentWindow = ReplaceElemenetsIntoWindow(currentWindow, lastMsg, optionBlock)
	}

	return currentWindow, isNext
}

func sendWindow(coala common.SenderIface, pools *pools.AllPools, window []*m.CoAPMessage, address *net.UDPAddr) (state *windowState, err error) {
	var wg sync.WaitGroup

	state = new(windowState)
	state.Lenght = len(window)
	state.WaitingConfirmation = len(window)

	for i, msg := range window {
		if msg.IsACKed {
			state.WaitingConfirmation--
			if i == 0 {
				state.FirstMessageAck = true
			}

			continue
		}

		if msg.Attempts != 0 {
			if msg.Attempts < 3 {
				coala.GetMetrics().Retransmissions.Inc()
			}
			msg.Attempts--
		} else {
			err = errors.New("Max attempts")
			return
		}
		go func(msg *m.CoAPMessage) {
			wg.Add(1)
			send(coala, msg, pools, state, i, address)
			wg.Done()
		}(msg)

	}
	wg.Wait()
	return
}

type windowState struct {
	Lenght              int
	Timeout             bool
	FirstMessageAck     bool
	Full                bool
	WaitingConfirmation int

	LastMessage *m.CoAPMessage
}

func send(coala common.SenderIface, msg *m.CoAPMessage, pools *pools.AllPools, state *windowState, index int, address *net.UDPAddr) {
	respMessage, err := coala.Send(msg, address)
	if err != nil {
		state.Timeout = true
		return
	}
	if msg.Type == m.ACK {
		return
	}

	if respMessage == nil {
		log.Error("Response is nil msg:", msg.ToReadableString())
		return
	}

	msg.IsACKed = true

	state.WaitingConfirmation--
	if index == 0 {
		state.FirstMessageAck = true
	}

	if respMessage.Code != m.CoapCodeContinue {
		state.LastMessage = respMessage
	}
	if state.WaitingConfirmation == 0 && state.LastMessage != nil {
		state.Full = true
	}

	block := respMessage.GetBlock1()
	if block == nil {
		block = respMessage.GetBlock2()
	}
}

func processSending(coala common.SenderIface, pools *pools.AllPools, buffer *byteBuffer.Buffer, windowMessages []*m.CoAPMessage, address *net.UDPAddr) (next bool, isFull bool, lastMessage *m.CoAPMessage, err error) {

	for {
		state, err := sendWindow(coala, pools, windowMessages, address)
		if err != nil {
			return false, false, state.LastMessage, err
		}
		if state.Full {
			return false, true, state.LastMessage, err
		}

		if state.FirstMessageAck {
			return true, false, state.LastMessage, err
		}

		if state.Timeout {
			continue
		}
	}
}
