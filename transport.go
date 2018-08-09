package coalago

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

var (
	ErrUnsupportedType = errors.New("unsupported type of message")
)

type transport struct {
	conn           dialer
	sessions       *cache.Cache
	block2channels sync.Map
	block1channels sync.Map
}

func newtransport(conn dialer) *transport {
	sr := new(transport)
	sr.conn = conn
	sr.sessions = cache.New(SESSIONS_POOL_EXPIRATION, time.Second*10)

	return sr
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
	case ACK, NON:
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

		_, err = sr.conn.Write(data)
		if err != nil {
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

		if resp.Type == ACK && resp.Code == CoapCodeEmpty {
			return sr.receiveARQBlock2(nil)
		}

		// if block := resp.GetBlock2(); block != nil {
		// 	return receiveARQBlock2(conn, resp)
		// }

		break
	}

	return
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
	_, err = sr.conn.Write(buf)
	buf = nil
	return err
}

func (sr *transport) sendToSocketByAddress(message *CoAPMessage, addr net.Addr) error {

	buf, err := preparationSendingMessage(sr, message, addr)
	if err != nil {
		return err
	}
	_, err = sr.conn.WriteTo(buf, addr.String())
	buf = nil
	return err
}

func (sr *transport) sendPackets(packets []*packet, shift int) error {
	stop := shift + DEFAULT_WINDOW_SIZE
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

	return nil
}

func (sr *transport) sendPacketsToAddr(packets []*packet, shift int, addr net.Addr) error {
	stop := shift + DEFAULT_WINDOW_SIZE
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

	err := sr.sendPackets(packets, shift)
	if err != nil {
		return nil, err
	}

	for {
		resp, err := receiveMessage(sr)
		if err != nil {
			if err == ErrMaxAttempts {
				if err = sr.sendPackets(packets, shift); err != nil {
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
				return sr.receiveARQBlock2(nil)
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

						if err = sr.sendPackets(packets, shift); err != nil {
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

	if err := sr.sendPacketsToAddr(packets, shift, addr); err != nil {
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

							if err := sr.sendPacketsToAddr(packets, shift, addr); err != nil {
								return err
							}
						}
					}
				}
			}
		case <-time.After(time.Second * 10):
			if err := sr.sendPacketsToAddr(packets, shift, addr); err != nil {
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
			if block == nil {
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

			ack := ackTo(inputMessage, CoapCodeContinue)
			ack.AddOption(OptionBlock1, block.ToInt())
			if err := sr.sendToSocketByAddress(ack, inputMessage.Sender); err != nil {
				return nil, err
			}

		case <-time.After(time.Second * 18):
			return nil, ErrMaxAttempts
		}
	}
}

func (sr *transport) receiveARQBlock2(inputMessage *CoAPMessage) (rsp *CoAPMessage, err error) {
	buf := make(map[int][]byte)
	totalBlocks := -1

	var attempts int

	if inputMessage != nil {
		block := inputMessage.GetBlock2()
		if block != nil {
			if !block.MoreBlocks {
				totalBlocks = block.BlockNumber + 1
			}
			buf[block.BlockNumber] = inputMessage.Payload.Bytes()
			if totalBlocks == len(buf) {
				b := []byte{}
				for i := 0; i < totalBlocks; i++ {
					b = append(b, buf[0]...)
				}
				inputMessage.Payload = NewBytesPayload(b)
				ack := ackTo(inputMessage, CoapCodeEmpty)
				ack.AddOption(OptionBlock2, block.ToInt())
				sr.sendToSocket(inputMessage)
				return inputMessage, nil
			}
			ack := ackTo(inputMessage, CoapCodeContinue)
			ack.AddOption(OptionBlock2, block.ToInt())
			sr.sendToSocket(inputMessage)
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
		if block == nil {
			continue
		}

		if !block.MoreBlocks {
			totalBlocks = block.BlockNumber + 1
		}

		buf[block.BlockNumber] = inputMessage.Payload.Bytes()
		if totalBlocks == len(buf) {
			b := []byte{}
			for i := 0; i < totalBlocks; i++ {
				b = append(b, buf[0]...)
			}
			inputMessage.Payload = NewBytesPayload(b)
			ack := ackTo(inputMessage, CoapCodeEmpty)
			ack.AddOption(OptionBlock1, block.ToInt())
			if err = sr.sendToSocket(ack); err != nil {
				return nil, err
			}
			return inputMessage, nil
		}

		ack := ackTo(inputMessage, CoapCodeContinue)
		ack.AddOption(OptionBlock2, block.ToInt())
		if err = sr.sendToSocket(ack); err != nil {
			return nil, err
		}
	}
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

	message, err := preparationReceivingMessage(sr, readBuf[:n], senderAddr)
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
					go respHandler(resp, err)
				}()
				sr.block1channels.Store(id, ch)
				ch <- message
				return
			}
		} else {
			c.(chan *CoAPMessage) <- message
			return
		}
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
	if err := securityClientSend(tr, tr.sessions, nil, message, addr); err != nil {
		return nil, err
	}

	buf, err := Serialize(message)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func preparationReceivingMessage(tr *transport, data []byte, senderAddr net.Addr) (*CoAPMessage, error) {
	message, err := Deserialize(data)
	if err != nil {
		return nil, err
	}
	message.Sender = senderAddr

	if securityReceive(tr, tr.sessions, nil, message) {
		return message, nil
	}

	return nil, errors.New("Session error")
}
