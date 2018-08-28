package coalago

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

var (
	ErrUnsupportedType = errors.New("unsupported type of message")
	globalSessions     = cache.New(SESSIONS_POOL_EXPIRATION, time.Second*10)
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
		return sr.sendCON(message)
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
		attempts++
		MetricSentMessages.Inc()
		fmt.Println("WRITE CON Message")
		_, err = sr.conn.Write(data)
		if err != nil {
			MetricSentMessageErrors.Inc()
			return nil, err
		}

		resp, err = receiveMessage(sr)
		if err == ErrMaxAttempts {
			if attempts == 3 {
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
			return sr.receiveARQBlock2(message, nil)
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
		}
	}

	return sr.sendToSocketByAddress(message, addr)
}

func (sr *transport) sendToSocket(message *CoAPMessage) error {
	buf, err := preparationSendingMessage(sr, message, sr.conn.RemoteAddr())
	if err != nil {
		return err
	}
	MetricSentMessages.Inc()
	_, err = sr.conn.Write(buf)
	if err != nil {
		MetricSentMessageErrors.Inc()
	}
	buf = nil
	return err
}

func (sr *transport) sendToSocketByAddress(message *CoAPMessage, addr net.Addr) error {

	buf, err := preparationSendingMessage(sr, message, addr)
	if err != nil {
		return err
	}
	MetricSentMessages.Inc()
	_, err = sr.conn.WriteTo(buf, addr.String())
	if err != nil {
		MetricSentMessageErrors.Inc()
	}
	buf = nil
	return err
}

func (sr *transport) sendPackets(packets []*packet, windowsize int, shift int) error {
	stop := shift + windowsize
	if stop >= len(packets) {
		stop = len(packets)
	}

	_3s := time.Second * 3
	var acked int
	for i := 0; i < stop; i++ {
		if !packets[i].acked {
			if time.Since(packets[i].lastSend) >= _3s {
				if packets[i].attempts == 3 {
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
		if time.Since(packets[len(packets)-1].lastSend) >= _3s {
			MetricExpiredMessages.Inc()
			return ErrMaxAttempts
		}
	}

	return nil
}

func (sr *transport) sendPacketsToAddr(packets []*packet, windowsize int, shift int, addr net.Addr) error {
	stop := shift + windowsize
	if stop >= len(packets) {
		stop = len(packets)
	}

	// need a more elegant solution
	if shift == len(packets) {
		return ErrMaxAttempts
	}

	_3s := time.Second * 3
	var acked int
	for i := 0; i < stop; i++ {
		if !packets[i].acked {
			if time.Since(packets[i].lastSend) >= _3s {
				if packets[i].attempts == 3 {
					MetricExpiredMessages.Inc()
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

	return nil
}

func (sr *transport) sendARQBlock1CON(message *CoAPMessage) (*CoAPMessage, error) {
	state := new(stateSend)
	state.payload = message.Payload.Bytes()
	state.lenght = len(state.payload)
	state.origMessage = message
	state.blockSize = MAX_PAYLOAD_SIZE
	numblocks := math.Ceil(float64(state.lenght) / float64(MAX_PAYLOAD_SIZE))
	if numblocks < DEFAULT_WINDOW_SIZE {
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

	err := sr.sendPackets(packets, state.windowsize, shift)
	if err != nil {
		return nil, err
	}

	for {
		resp, err := receiveMessage(sr)
		if err != nil {
			if err == ErrMaxAttempts {
				if err = sr.sendPackets(packets, state.windowsize, shift); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}

		if !bytes.Equal(resp.Token, message.Token) {
			continue
		}

		if resp.Type == ACK {
			if resp.Type == ACK && resp.Code == CoapCodeEmpty {
				return sr.receiveARQBlock2(message, nil)
			}
			block := resp.GetBlock1()
			if block != nil {
				if len(packets) >= block.BlockNumber {
					if resp.Code != CoapCodeContinue {
						return resp, nil
					}
					packets[block.BlockNumber].acked = true
					if block.BlockNumber == shift {
						shift++
						for _, p := range packets[shift:] {
							if p.acked {
								shift++
							} else {
								break
							}
						}

						if err = sr.sendPackets(packets, state.windowsize, shift); err != nil {
							return nil, err
						}
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
	if numblocks < DEFAULT_WINDOW_SIZE {
		state.windowsize = int(numblocks)
	} else {
		state.windowsize = DEFAULT_WINDOW_SIZE
	}

	packets := []*packet{}

	emptyAckMessage := newACKEmptyMessage(message)
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

	if err := sr.sendPacketsToAddr(packets, state.windowsize, shift, addr); err != nil {
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
						if resp.Code != CoapCodeContinue {
							return nil
						}
						packets[block.BlockNumber].acked = true
						if block.BlockNumber == shift {
							shift++
							for _, p := range packets[shift:] {
								if p.acked {
									shift++
								} else {
									break
								}
							}

							if err := sr.sendPacketsToAddr(packets, state.windowsize, shift, addr); err != nil {
								return err
							}
						}
					}
				}
			}
		case <-time.After(time.Second * 10):
			if err := sr.sendPacketsToAddr(packets, state.windowsize, shift, addr); err != nil {
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

			ack := ackTo(nil, inputMessage, CoapCodeContinue)

			if err := sr.sendToSocketByAddress(ack, inputMessage.Sender); err != nil {
				return nil, err
			}

		case <-time.After(time.Second * 18):
			MetricExpiredMessages.Inc()
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
			ack := ackTo(origMessage, inputMessage, CoapCodeContinue)
			sr.sendToSocket(ack)
		}
	}

	for {
		inputMessage, err = receiveMessage(sr)
		if err == ErrMaxAttempts {
			if attempts == 3 {
				return nil, err
			}
			attempts++
			continue
		}
		if err != nil {
			return nil, err
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
			return inputMessage, nil
		}

		ack := ackTo(origMessage, inputMessage, CoapCodeContinue)
		if err = sr.sendToSocket(ack); err != nil {
			return nil, err
		}
	}
}

func (sr *transport) ReceiveMessage(message *CoAPMessage, respHandler func(*CoAPMessage, error)) {
	message, err := preparationReceivingMessage(sr, message)
	if err != nil {
		return
	}

	sr.messageHandlerSelector(message, respHandler)
}

func (sr *transport) ReceiveOnce(respHandler func(*CoAPMessage, error)) {
	readBuf := make([]byte, 1500)
start:
	n, senderAddr, err := sr.conn.Listen(readBuf)
	if err != nil {
		panic(err)
	}
	if n == 0 {
		goto start
	}

	message, err := preparationReceivingBuffer(sr, readBuf[:n], senderAddr)
	if err != nil {
		goto start
	}

	message.Sender = senderAddr

	sr.messageHandlerSelector(message, respHandler)
}

func (sr *transport) messageHandlerSelector(message *CoAPMessage, respHandler func(*CoAPMessage, error)) {
	block1 := message.GetBlock1()
	block2 := message.GetBlock2()

	id := message.Sender.String() + string(message.Token)
	var (
		c  interface{}
		ch chan *CoAPMessage
		ok bool
	)

	if block1 != nil {
		c, ok = sr.block1channels.Load(id)
		if !ok {
			if message.Type == CON {
				ch = make(chan *CoAPMessage, 102400)
				go func() {
					resp, err := sr.receiveARQBlock1(ch)
					sr.block1channels.Delete(id)
					respHandler(resp, err)
				}()
				sr.block1channels.Store(id, ch)
				ch <- message
				return
			}
		} else {
			c.(chan *CoAPMessage) <- message
			return
		}
		return
	}

	if block2 != nil {
		if message.Type == ACK {
			c, ok = sr.block2channels.Load(id)
			if ok {
				c.(chan *CoAPMessage) <- message
			}
		}
		return
	}
	go respHandler(message, nil)
}

func preparationSendingMessage(tr *transport, message *CoAPMessage, addr net.Addr) ([]byte, error) {
	if err := securityClientSend(tr, message, addr); err != nil {
		return nil, err
	}

	// fmt.Println(time.Now().Format("15:04:05.000000000"), "\t---> send\t", addr, message.ToReadableString())

	buf, err := Serialize(message)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func preparationReceivingBuffer(tr *transport, data []byte, senderAddr net.Addr) (*CoAPMessage, error) {
	message, err := Deserialize(data)
	if err != nil {
		return nil, err
	}

	MetricReceivedMessages.Inc()

	message.Sender = senderAddr
	// fmt.Println(time.Now().Format("15:04:05.000000000"), "\t<--- receive\t", senderAddr, message.ToReadableString())

	if securityReceive(tr, message) {
		return message, nil
	}

	return nil, errors.New("Session error")
}

func preparationReceivingMessage(tr *transport, message *CoAPMessage) (*CoAPMessage, error) {
	// fmt.Println(time.Now().Format("15:04:05.000000000"), "\t<--- receive\t", message.Sender, message.ToReadableString())
	MetricReceivedMessages.Inc()
	if securityReceive(tr, message) {
		return message, nil
	}

	return nil, errors.New("Session error")
}
