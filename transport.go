package coalago

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/coalalib/coalago/util"
	log "github.com/ndmsystems/golog"
	"github.com/patrickmn/go-cache"
)

var (
	globalSessions     = newSessionStorageImpl()
	handlersStateCache = cache.New(sumTimeAttempts, time.Second)
	proxyIDSessions    = newProxySessionStorage()
)

type transport struct {
	conn           dialer
	block2channels sync.Map
	block1channels sync.Map
	privateKey     []byte
}

func newtransport(conn dialer) *transport {
	sr := new(transport)
	sr.conn = conn

	return sr
}

func (tr *transport) SetPrivateKey(pk []byte) {
	tr.privateKey = pk
}

func (sr *transport) Send(message *CoAPMessage) (resp *CoAPMessage, err error) {
	switch message.Type {
	case CON:

		if message.GetScheme() == COAPS_SCHEME {
			proxyAddr := message.ProxyAddr
			if len(proxyAddr) > 0 {
				proxyID := setProxyIDIfNeed(message, sr.conn.LocalAddr().String())
				proxyAddr = fmt.Sprintf("%v%v", proxyAddr, proxyID)
			}

			_, err := handshake(sr, message, sr.conn.RemoteAddr(), proxyAddr)
			if err != nil {
				return nil, err
			}
		}

		resp, err := sr.sendCON(message)
		if err == ErrorSessionExpired || err == ErrorSessionNotFound ||
			err == ErrorClientSessionExpired || err == ErrorClientSessionNotFound {
			if message.GetScheme() == COAPS_SCHEME {
				proxyAddr := message.ProxyAddr
				if len(proxyAddr) > 0 {
					proxyID := setProxyIDIfNeed(message, sr.conn.LocalAddr().String())
					proxyAddr = fmt.Sprintf("%v%v", proxyAddr, proxyID)
				}
				if _, err := handshake(sr, message, sr.conn.RemoteAddr(), proxyAddr); err != nil {
					return nil, err
				}
			}

			resp, err = sr.sendCON(message)
		}
		return resp, err
	case RST, NON:
		return nil, sr.sendToSocket(message)
	default:
		return nil, ErrUnsupportedType
	}
}

func (sr *transport) SendTo(message *CoAPMessage, addr net.Addr) (resp *CoAPMessage, err error) {
	switch message.Type {
	case ACK, NON, RST:
		return nil, sr.sendACKTo(message, addr)
	default:
		return nil, ErrUnsupportedType
	}
}

func (sr *transport) sendCON(message *CoAPMessage) (resp *CoAPMessage, err error) {
	if isBigPayload(message) {
		resp, err = sr.sendARQBlock1CON(message)
		return
	}

	data, err := preparationSendingMessage(sr, message, sr.conn.RemoteAddr())
	if err != nil {
		return nil, err
	}

	attempts := 0

	for {
		if attempts > 0 {
			util.MetricRetransmitMessages.Inc()
		}
		attempts++
		util.MetricSentMessages.Inc()
		_, err = sr.conn.Write(data)
		if err != nil {
			util.MetricSentMessageErrors.Inc()
			return nil, err
		}

		resp, err = receiveMessage(sr, message)
		if err == ErrMaxAttempts {
			if attempts == maxSendAttempts {
				util.MetricExpiredMessages.Inc()
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}

		if isPingACK(resp) {
			return resp, nil
		}

		if resp.Type == ACK && resp.Code == CoapCodeEmpty {
			resp, err = sr.receiveARQBlock2(message, nil)
			return resp, err
		}

		if resp.GetBlock2() != nil {
			resp, err = sr.receiveARQBlock2(message, resp)
			return resp, err
		}

		break
	}

	return
}

func isPingACK(resp *CoAPMessage) bool {
	return resp.Type == RST && resp.Code == CoapCodeEmpty
}

func (sr *transport) sendACKTo(message *CoAPMessage, addr net.Addr) (err error) {
	if message.Type == ACK {
		if isBigPayload(message) {
			ch := make(chan *CoAPMessage, 102400)
			id := addr.String() + message.GetTokenString()
			sr.block2channels.Store(id, ch)
			err = sr.sendARQBlock2ACK(ch, message, addr)
			sr.block2channels.Delete(id)
			return err
		}
	}

	return sr.sendToSocketByAddress(message, addr)
}

func (sr *transport) sendToSocket(message *CoAPMessage) error {
	buf, err := preparationSendingMessage(sr, message, sr.conn.RemoteAddr())
	if err != nil {
		return err
	}
	util.MetricSentMessages.Inc()
	_, err = sr.conn.Write(buf)
	if err != nil {
		util.MetricSentMessageErrors.Inc()
	}
	buf = nil
	return err
}

func (sr *transport) sendToSocketByAddress(message *CoAPMessage, addr net.Addr) error {

	buf, err := preparationSendingMessage(sr, message, addr)
	if err != nil {
		return err
	}
	util.MetricSentMessages.Inc()
	_, err = sr.conn.WriteTo(buf, addr.String())
	if err != nil {
		util.MetricSentMessageErrors.Inc()
	}
	buf = nil
	return err
}

func (sr *transport) sendPackets(packets []*packet, windowsize *int, shift int, relative_shift int, localMetricsRetransmitMessages *int, overflowIndicator *int) error {
	stop := *windowsize
	if *overflowIndicator > 0 {
		stop += shift
	} else {
		stop += relative_shift
	}
	if stop >= len(packets) {
		stop = len(packets)
	}
	var acked int

	for i := shift; i < stop; i++ {
		if !packets[i].acked {
			if time.Since(packets[i].lastSend) >= timeWait {
				if packets[i].attempts > 0 && *windowsize >= MIN_WiNDOW_SIZE {
					util.MetricRetransmitMessages.Inc()
					*localMetricsRetransmitMessages++
				}

				if packets[i].attempts == maxSendAttempts {
					util.MetricExpiredMessages.Inc()
					return ErrMaxAttempts
				}
				if packets[i].attempts == 3 {
					*overflowIndicator++
				}
				packets[i].attempts++
				packets[i].lastSend = time.Now()

				if err := sr.sendToSocket(packets[i].message); err != nil {
					return err
				}

			}
		} else {
			acked++
		}
	}

	return nil
}

func (sr *transport) sendPacketsByWindowOffset(packets []*packet, windowsize, shift, blockNumber, offset int) error {
	stop := shift + windowsize
	if stop >= blockNumber {
		stop = blockNumber
	}

	start := blockNumber - offset
	if start < 0 {
		start = 0
	} else if start > blockNumber {
		return nil
	}

	var acked int
	for i := start; i < stop; i++ {
		if !packets[i].acked {
			if time.Since(packets[i].lastSend) >= timeWait {
				if packets[i].attempts > 0 {
					util.MetricRetransmitMessages.Inc()
				}
				if packets[i].attempts == maxSendAttempts {
					util.MetricExpiredMessages.Inc()
					return ErrMaxAttempts
				}
				packets[i].attempts++
				packets[i].lastSend = time.Now()
				if err := sr.sendToSocket(packets[i].message); err != nil {
					return err
				}
			}
		} else {
			acked++
		}
	}

	if len(packets) == stop {
		if time.Since(packets[len(packets)-1].lastSend) >= timeWait {
			util.MetricExpiredMessages.Inc()
			return ErrMaxAttempts
		}
	}

	return nil
}

func (sr *transport) sendPacketsByWindowOffsetToAddr(packets []*packet, windowsize, shift, blockNumber, offset int, addr net.Addr) error {
	stop := shift + windowsize
	if stop >= blockNumber {
		stop = blockNumber
	}

	start := blockNumber - offset
	if start < 0 {
		start = 0
	} else if start > blockNumber {
		return nil
	}

	var acked int
	for i := start; i < stop; i++ {
		if !packets[i].acked {
			if time.Since(packets[i].lastSend) >= timeWait {
				if packets[i].attempts > 0 {
					util.MetricRetransmitMessages.Inc()
				}
				if packets[i].attempts == maxSendAttempts {
					util.MetricExpiredMessages.Inc()
					return ErrMaxAttempts
				}
				packets[i].attempts++
				packets[i].lastSend = time.Now()
				if err := sr.sendToSocketByAddress(packets[i].message, addr); err != nil {
					return err
				}
			}
		} else {
			acked++
		}
	}

	if len(packets) == stop {
		if time.Since(packets[len(packets)-1].lastSend) >= timeWait {
			util.MetricExpiredMessages.Inc()
			return ErrMaxAttempts
		}
	}

	return nil
}

func (sr *transport) sendPacketsToAddr(packets []*packet, windowsize *int, shift int, relative_shift int, localMetricsRetransmitMessages *int, overflowIndicator *int, addr net.Addr) error {
	stop := *windowsize
	if *overflowIndicator > 0 {
		stop += shift
	} else {
		stop += relative_shift
	}
	if stop >= len(packets) {
		stop = len(packets)
	}
	// need a more elegant solution
	if shift == len(packets) {
		return ErrMaxAttempts
	}

	var acked int
	for i := shift; i < stop; i++ {
		if !packets[i].acked {
			if time.Since(packets[i].lastSend) >= timeWait {
				if packets[i].attempts == maxSendAttempts {
					util.MetricExpiredMessages.Inc()
					return ErrMaxAttempts
				}

				if packets[i].attempts == 3 {
					*overflowIndicator++
				}

				packets[i].attempts++

				if packets[i].attempts > 1 && *windowsize > MIN_WiNDOW_SIZE {
					util.MetricRetransmitMessages.Inc()
					*localMetricsRetransmitMessages++
				}
				packets[i].lastSend = time.Now()
				if err := sr.sendToSocketByAddress(packets[i].message, addr); err != nil {
					return err
				}
			}
		} else {
			acked++
		}
	}

	return nil
}

func (sr *transport) sendARQBlock1CON(message *CoAPMessage) (*CoAPMessage, error) {

	state := new(stateSend)
	state.payload = message.Payload.Bytes()
	state.lenght = len(state.payload)
	state.origMessage = message
	state.blockSize = MAX_PAYLOAD_SIZE
	numblocks := math.Ceil(float64(state.lenght) / float64(MAX_PAYLOAD_SIZE))
	if int(numblocks) < DEFAULT_WINDOW_SIZE {
		state.windowsize = int(numblocks)
	} else {
		state.windowsize = DEFAULT_WINDOW_SIZE
	}

	packets := []*packet{}

	for {
		blockMessage, end := constructNextBlock(OptionBlock1, state)
		packets = append(packets, &packet{
			acked:   false,
			message: blockMessage,
		})

		if end {
			break
		}
	}

	var shift = 0
	var relative_shift = 0
	var localMetricsRetransmitMessages = 0
	var downloadStartTime = time.Now()
	var retransmitsTmp = 0
	var balancerCounter = 0
	var overflowIndicator = 0

	err := sr.sendPackets(packets, &state.windowsize, shift, relative_shift, &localMetricsRetransmitMessages, &overflowIndicator)

	if err != nil {
		return nil, err
	}

	for {
		resp, err := receiveMessage(sr, message)
		if err != nil {
			if err == ErrMaxAttempts {
				if err = sr.sendPackets(packets, &state.windowsize, shift, relative_shift, &localMetricsRetransmitMessages, &overflowIndicator); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}

		if resp.Type == ACK {
			if resp.Type == ACK && resp.Code == CoapCodeEmpty {
				return sr.receiveARQBlock2(message, nil)
			}

			if resp.GetBlock2() != nil {
				return sr.receiveARQBlock2(message, resp)
			}

			block := resp.GetBlock1()
			if block != nil {
				// wo := resp.GetOption(OptionWindowtOffset)
				// if wo != nil {
				// 	sr.sendPacketsByWindowOffset(packets, state.windowsize, shift, block.BlockNumber, int(wo.Value.(uint32)))
				// }
				if len(packets) >= block.BlockNumber {
					balancerCounter++
					if resp.Code != CoapCodeContinue {
						if len(packets) > DEFAULT_WINDOW_SIZE*2 {
							log.Debug(fmt.Sprintf("COALA U: %s, %s, Packets: %d Lost: %d, FinalWSize: %d",
								util.ByteCountBinary(int64(state.lenght)),
								util.ByteCountBinaryBits(int64(state.lenght)*time.Second.Milliseconds()/time.Since(downloadStartTime).Milliseconds()),
								len(packets),
								localMetricsRetransmitMessages,
								state.windowsize))
						}
						return resp, nil
					}
					if !packets[block.BlockNumber].acked && packets[block.BlockNumber].attempts > 3 {
						overflowIndicator--
					}
					packets[block.BlockNumber].acked = true

					relative_shift++
					if block.BlockNumber == shift {
						shift++
						for _, p := range packets[shift:] {
							if p.acked {
								shift++
							} else {
								break
							}
						}
					}

					if balancerCounter%25 == 0 {
						dt := int(float64(2-localMetricsRetransmitMessages+retransmitsTmp) * 0.7)
						state.windowsize += dt
						retransmitsTmp = localMetricsRetransmitMessages

						if state.windowsize < MIN_WiNDOW_SIZE {
							state.windowsize = MIN_WiNDOW_SIZE
						}
						if state.windowsize > MAX_WINDOW_SIZE {
							state.windowsize = MAX_WINDOW_SIZE
						}

					}

					if err = sr.sendPackets(packets, &state.windowsize, shift, relative_shift, &localMetricsRetransmitMessages, &overflowIndicator); err != nil {
						return nil, err
					}

				}
			}
		}
	}
}

func (sr *transport) sendARQBlock2ACK(input chan *CoAPMessage, message *CoAPMessage, addr net.Addr) error {
	state := new(stateSend)
	state.payload = message.Payload.Bytes()
	state.lenght = len(state.payload)
	state.origMessage = message
	state.blockSize = MAX_PAYLOAD_SIZE
	numblocks := math.Ceil(float64(state.lenght) / float64(MAX_PAYLOAD_SIZE))
	if int(numblocks) < DEFAULT_WINDOW_SIZE {
		state.windowsize = int(numblocks)
	} else {
		state.windowsize = DEFAULT_WINDOW_SIZE
	}

	packets := []*packet{}

	emptyAckMessage := newACKEmptyMessage(message, state.windowsize)
	err := sr.sendToSocketByAddress(emptyAckMessage, addr)
	if err != nil {
		return err
	}
	emptyAckMessage = nil

	for {
		blockMessage, end := constructNextBlock(OptionBlock2, state)
		packets = append(packets, &packet{
			acked:   false,
			message: blockMessage,
		})

		if end {
			break
		}
	}

	var shift = 0
	var relative_shift = 0
	var localMetricsRetransmitMessages = 0
	downloadStartTime := time.Now()
	var retransmitsTmp = 0
	var balancerCounter = 0
	var overflowIndicator = 0

	if err := sr.sendPacketsToAddr(packets, &state.windowsize, shift, relative_shift, &localMetricsRetransmitMessages, &overflowIndicator, addr); err != nil {
		return err
	}
	for {
		select {
		case resp := <-input:
			if !bytes.Equal(resp.Token, message.Token) {
				continue
			}
			if resp.Type == ACK {
				block := resp.GetBlock2()
				if block != nil {
					if len(packets) >= block.BlockNumber {
						balancerCounter++
						if resp.Code != CoapCodeContinue {
							if len(packets) > DEFAULT_WINDOW_SIZE*2 {
								log.Debug(fmt.Sprintf("COALA U: %s, %s, Packets: %d Lost: %d, FinalWSize: %d",
									util.ByteCountBinary(int64(state.lenght)),
									util.ByteCountBinaryBits(int64(state.lenght)*time.Second.Milliseconds()/time.Since(downloadStartTime).Milliseconds()),
									len(packets),
									localMetricsRetransmitMessages,
									state.windowsize))
							}
							return nil
						}
						if block.BlockNumber < len(packets) {
							// wo := resp.GetOption(OptionWindowtOffset)
							// if wo != nil {
							// 	wov := wo.Uint16Value()
							// 	sr.sendPacketsByWindowOffset(packets, state.windowsize, shift, block.BlockNumber, int(wov))

							// }
							if !packets[block.BlockNumber].acked && packets[block.BlockNumber].attempts > 3 {
								overflowIndicator--
							}
							packets[block.BlockNumber].acked = true
							relative_shift++
							if block.BlockNumber == shift {
								shift++
								for _, p := range packets[shift:] {
									if p.acked {
										shift++
									} else {
										break
									}
								}
							}

							if balancerCounter%25 == 0 {
								dt := int(float64(2-localMetricsRetransmitMessages+retransmitsTmp) * 0.7)
								state.windowsize += dt
								retransmitsTmp = localMetricsRetransmitMessages

								if state.windowsize < MIN_WiNDOW_SIZE {
									state.windowsize = MIN_WiNDOW_SIZE
								}
								if state.windowsize > MAX_WINDOW_SIZE {
									state.windowsize = MAX_WINDOW_SIZE
								}
							}

							if err := sr.sendPacketsToAddr(packets, &state.windowsize, shift, relative_shift, &localMetricsRetransmitMessages, &overflowIndicator, addr); err != nil {
								return err
							}
						}
					}
				}
			}
		case <-time.After(timeWait):
			if err := sr.sendPacketsToAddr(packets, &state.windowsize, shift, relative_shift, &localMetricsRetransmitMessages, &overflowIndicator, addr); err != nil {
				return err
			}
		}
	}
}

func (sr *transport) receiveARQBlock1(input chan *CoAPMessage) (*CoAPMessage, error) {
	buf := make(map[int][]byte)
	totalBlocks := -1

	for {
		select {
		case inputMessage := <-input:
			block := inputMessage.GetBlock1()
			if block == nil || inputMessage.Type != CON {
				continue
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

				return inputMessage, nil
			}

			var ack *CoAPMessage
			w := inputMessage.GetOption(OptionSelectiveRepeatWindowSize)
			if w != nil {
				ack = ackToWithWindowOffset(nil, inputMessage, CoapCodeContinue, w.IntValue(), block.BlockNumber, buf)
			} else {
				ack = ackTo(nil, inputMessage, CoapCodeContinue)
			}

			if err := sr.sendToSocketByAddress(ack, inputMessage.Sender); err != nil {
				return nil, err
			}

		case <-time.After(timeWait):
			util.MetricExpiredMessages.Inc()
			return nil, ErrMaxAttempts
		}
	}
}

func (sr *transport) receiveARQBlock2(origMessage *CoAPMessage, inputMessage *CoAPMessage) (rsp *CoAPMessage, err error) {
	buf := make(map[int][]byte)
	totalBlocks := -1
	var attempts int

	if inputMessage != nil {
		block := inputMessage.GetBlock2()

		if block != nil && inputMessage.Type == CON {
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

				ack := ackTo(origMessage, inputMessage, CoapCodeEmpty)
				sr.sendToSocket(ack)
				return inputMessage, nil
			}

			var ack *CoAPMessage
			w := inputMessage.GetOption(OptionSelectiveRepeatWindowSize)
			if w != nil {
				ack = ackToWithWindowOffset(origMessage, inputMessage, CoapCodeContinue, w.IntValue(), block.BlockNumber, buf)
			} else {
				ack = ackTo(origMessage, inputMessage, CoapCodeContinue)
			}
			sr.sendToSocket(ack)
		}
	}

	downloadStartTime := time.Now()

	for {
		inputMessage, err = receiveMessage(sr, origMessage)
		if err == ErrMaxAttempts {
			if attempts == maxSendAttempts {
				util.MetricExpiredMessages.Inc()
				return nil, err
			}
			attempts++
			continue
		}
		if err != nil {
			return nil, err
		}

		if attempts > 0 {
			util.MetricRetransmitMessages.Inc()
		}
		block := inputMessage.GetBlock2()
		if block == nil || inputMessage.Type != CON {
			continue
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
			ack := ackTo(origMessage, inputMessage, CoapCodeEmpty)
			if err = sr.sendToSocket(ack); err != nil {
				return nil, err
			}
			if len(buf) > DEFAULT_WINDOW_SIZE*2 {
				log.Debug(fmt.Sprintf("COALA D: %s, %s",
					util.ByteCountBinary(int64(len(b))),
					util.ByteCountBinaryBits(int64(len(b))*time.Second.Milliseconds()/time.Since(downloadStartTime).Milliseconds())))
			}
			return inputMessage, nil
		}

		var ack *CoAPMessage
		w := inputMessage.GetOption(OptionSelectiveRepeatWindowSize)
		if w != nil {
			ack = ackToWithWindowOffset(origMessage, inputMessage, CoapCodeContinue, w.IntValue(), block.BlockNumber, buf)
		} else {
			ack = ackTo(origMessage, inputMessage, CoapCodeContinue)
		}

		if err = sr.sendToSocket(ack); err != nil {
			return nil, err
		}
	}
}

func preparationSendingMessage(tr *transport, message *CoAPMessage, addr net.Addr) ([]byte, error) {
	secMessage := message.Clone(true)

	if err := securityOutputLayer(tr, secMessage, addr); err != nil {
		return nil, err
	}

	buf, err := Serialize(secMessage)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func preparationReceivingBufferForStorageLocalStates(data []byte, senderAddr net.Addr) (*CoAPMessage, error) {
	message, err := Deserialize(data)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrNilMessage
	}

	util.MetricReceivedMessages.Inc()

	message.Sender = senderAddr

	return message, nil
}

func preparationReceivingBuffer(tr *transport, data []byte, senderAddr net.Addr, proxyAddr string) (*CoAPMessage, error) {
	message, err := Deserialize(data)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrNilMessage
	}

	util.MetricReceivedMessages.Inc()

	message.Sender = senderAddr

	_, err = securityInputLayer(tr, message, proxyAddr)

	if err != nil {
		return nil, err
	}
	return message, nil
}
