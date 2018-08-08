package coalago

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
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

func (sr *transport) Send(message *m.CoAPMessage) (resp *m.CoAPMessage, err error) {
	switch message.Type {
	case m.CON:
		return sr.sendCON(message)
	case m.RST:
		return nil, sr.sendToSocket(message)
	default:
		return nil, ErrUnsupportedType
	}
}

func (sr *transport) SendTo(message *m.CoAPMessage, addr net.Addr) (resp *m.CoAPMessage, err error) {
	switch message.Type {
	case m.ACK, m.NON:
		return nil, sr.sendACKTo(message, addr)
	default:
		return nil, ErrUnsupportedType
	}
}

func (sr *transport) sendCON(message *m.CoAPMessage) (resp *m.CoAPMessage, err error) {
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

		if resp.Type == m.ACK && resp.Code == m.CoapCodeEmpty {
			return sr.receiveARQBlock2(nil)
		}

		// if block := resp.GetBlock2(); block != nil {
		// 	return receiveARQBlock2(conn, resp)
		// }

		break
	}

	return
}

func (sr *transport) sendACKTo(message *m.CoAPMessage, addr net.Addr) (err error) {
	if message.Type == m.ACK {
		if isBigPayload(message) {
			ch := make(chan *m.CoAPMessage, 102400)
			id := addr.String() + message.GetTokenString()
			sr.block2channels.Store(id, ch)

			err = sr.sendARQBlock2ACK(ch, message, addr)
			sr.block2channels.Delete(id)
		}
	}

	return sr.sendToSocketByAddress(message, addr)
}

func (sr *transport) sendToSocket(message *m.CoAPMessage) error {
	buf, err := preparationSendingMessage(sr, message, sr.conn.RemoteAddr())
	if err != nil {
		return err
	}
	_, err = sr.conn.Write(buf)
	buf = nil
	return err
}

func (sr *transport) sendToSocketByAddress(message *m.CoAPMessage, addr net.Addr) error {

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

func (sr *transport) sendARQBlock1CON(message *m.CoAPMessage) (*m.CoAPMessage, error) {
	state := new(stateSend)
	state.payload = message.Payload.Bytes()
	state.lenght = len(state.payload)
	state.origMessage = message

	packets := []*packet{}

	for {
		blockMessage, end := constructNextBlock(m.OptionBlock1, state)
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

		if resp.Type == m.ACK {
			if resp.Type == m.ACK && resp.Code == m.CoapCodeEmpty {
				return sr.receiveARQBlock2(nil)
			}
			block := resp.GetBlock1()
			if block != nil {
				if len(packets) >= block.BlockNumber {
					if resp.Code != m.CoapCodeContinue {
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

func (sr *transport) sendARQBlock2ACK(input chan *m.CoAPMessage, message *m.CoAPMessage, addr net.Addr) error {
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
		blockMessage, end := constructNextBlock(m.OptionBlock2, state)
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
			if resp.Type == m.ACK {
				block := resp.GetBlock2()
				if block != nil {
					if len(packets) >= block.BlockNumber {
						if resp.Code != m.CoapCodeContinue {
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

func (sr *transport) receiveARQBlock1(input chan *m.CoAPMessage) (*m.CoAPMessage, error) {
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
				inputMessage.Payload = m.NewBytesPayload(b)

				return inputMessage, nil
			}

			ack := ackTo(inputMessage, m.CoapCodeContinue)
			ack.AddOption(m.OptionBlock1, block.ToInt())
			if err := sr.sendToSocketByAddress(ack, inputMessage.Sender); err != nil {
				return nil, err
			}

		case <-time.After(time.Second * 18):
			return nil, ErrMaxAttempts
		}
	}
}

func (sr *transport) receiveARQBlock2(inputMessage *m.CoAPMessage) (rsp *m.CoAPMessage, err error) {
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
				inputMessage.Payload = m.NewBytesPayload(b)
				ack := ackTo(inputMessage, m.CoapCodeEmpty)
				ack.AddOption(m.OptionBlock2, block.ToInt())
				sr.sendToSocket(inputMessage)
				return inputMessage, nil
			}
			ack := ackTo(inputMessage, m.CoapCodeContinue)
			ack.AddOption(m.OptionBlock2, block.ToInt())
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
			inputMessage.Payload = m.NewBytesPayload(b)
			ack := ackTo(inputMessage, m.CoapCodeEmpty)
			ack.AddOption(m.OptionBlock1, block.ToInt())
			if err = sr.sendToSocket(ack); err != nil {
				return nil, err
			}
			return inputMessage, nil
		}

		ack := ackTo(inputMessage, m.CoapCodeContinue)
		ack.AddOption(m.OptionBlock2, block.ToInt())
		if err = sr.sendToSocket(ack); err != nil {
			return nil, err
		}
	}
}

func (sr *transport) ReceiveOnce(respHandler func(*m.CoAPMessage, error)) {
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

func (sr *transport) messageHandlerSelector(message *m.CoAPMessage, respHandler func(*m.CoAPMessage, error)) {
	block1 := message.GetBlock1()
	block2 := message.GetBlock2()

	id := message.Sender.String() + string(message.Token)
	var (
		c  interface{}
		ch chan *m.CoAPMessage
		ok bool
	)

	if block1 != nil {
		c, ok = sr.block1channels.Load(id)
		if !ok {
			if message.Type == m.CON {
				ch = make(chan *m.CoAPMessage, 102400)
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
			c.(chan *m.CoAPMessage) <- message
			return
		}
	}

	if block2 != nil {
		if message.Type == m.ACK {
			c, ok = sr.block2channels.Load(id)
			if ok {
				c.(chan *m.CoAPMessage) <- message
			}
		}
		return
	}
	go respHandler(message, nil)
}

func preparationSendingMessage(tr *transport, message *m.CoAPMessage, addr net.Addr) ([]byte, error) {
	if err := securityClientSend(tr, tr.sessions, nil, message, addr); err != nil {
		return nil, err
	}

	buf, err := m.Serialize(message)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func preparationReceivingMessage(tr *transport, data []byte, senderAddr net.Addr) (*m.CoAPMessage, error) {
	message, err := m.Deserialize(data)
	if err != nil {
		return nil, err
	}
	message.Sender = senderAddr

	if securityReceive(tr, tr.sessions, nil, message) {
		return message, nil
	}

	return nil, errors.New("Session error")
}
