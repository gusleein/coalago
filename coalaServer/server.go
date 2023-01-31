package coalaServer

import (
	"math"
	"net"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
	r "github.com/coalalib/coalago/resource"
	"github.com/coalalib/coalago/util"
)

type Server struct {
	privateKey []byte

	postResources   map[string]r.CoAPResourceHandler
	getResources    map[string]r.CoAPResourceHandler
	deleteResources map[string]r.CoAPResourceHandler

	block2sendsMX sync.RWMutex
	block2sends   map[string]chan *m.CoAPMessage

	block1receiveMX sync.RWMutex
	block1receive   map[string]chan *m.CoAPMessage

	inProcessMX sync.RWMutex
	inProcess   map[string]struct{}

	secSessions *securitySessionStorage
}

func NewServer(pk []byte) *Server {
	s := new(Server)
	s.postResources = make(map[string]r.CoAPResourceHandler)
	s.getResources = make(map[string]r.CoAPResourceHandler)
	s.deleteResources = make(map[string]r.CoAPResourceHandler)

	s.block2sends = make(map[string]chan *m.CoAPMessage)
	s.block1receive = make(map[string]chan *m.CoAPMessage)

	s.inProcess = make(map[string]struct{})
	s.secSessions = newSecuritySessionStorage()

	s.privateKey = pk
	return s
}

func (s *Server) Listen(addr string) (err error) {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer pc.Close()

	for {
		buf := make([]byte, 1500)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}
		msg, err := buildMsg(addr, buf[:n])
		if err != nil {
			continue
		}

		util.MetricReceivedMessages.Inc()
		s.serve(pc, msg)
	}
}

func (s *Server) GET(path string, handler r.CoAPResourceHandler) {
	s.getResources[path] = handler
}

func (s *Server) POST(path string, handler r.CoAPResourceHandler) {
	s.postResources[path] = handler

}

func (s *Server) DELETE(path string, handler r.CoAPResourceHandler) {
	s.deleteResources[path] = handler
}

func buildMsg(addr net.Addr, buf []byte) (*m.CoAPMessage, error) {
	message, err := m.Deserialize(buf)
	if err != nil {
		return nil, err
	}
	message.Sender = addr
	return message, nil
}

func (s *Server) serve(pc net.PacketConn, msg *m.CoAPMessage) {
	next, err := s.securityInputLayer(pc, s.privateKey, msg)
	if !next || err != nil {
		s.deleteInProcess(msg.GetTokenString())
		return
	}

	switch msg.Type {
	case m.NON, m.CON:
		if block := msg.GetBlock1(); block != nil {
			s.block1receiveMX.Lock()
			ch, ok := s.block1receive[msg.GetTokenString()]

			if !ok {
				ch = make(chan *m.CoAPMessage, 1)
				s.block1receive[msg.GetTokenString()] = ch
				go s.receiveARQBlock1(pc, msg, ch)
			}
			s.block1receiveMX.Unlock()

			ch <- msg
			return
		}

		s.serveCON(pc, msg)
	case m.ACK:
		s.serveACK(pc, msg)
	}
}

func (s *Server) serveCON(pc net.PacketConn, msg *m.CoAPMessage) {
	if res, ok := s.getResource(msg); ok {
		s.inProcessMX.Lock()

		_, ok := s.inProcess[msg.GetTokenString()]
		if ok {
			s.inProcessMX.Unlock()
			return
		}

		s.inProcess[msg.GetTokenString()] = struct{}{}
		s.inProcessMX.Unlock()

		go func() {
			s.resourceProcessor(pc, msg, res)
			s.deleteInProcess(msg.GetTokenString())
		}()
	}
}

func (s *Server) serveACK(pc net.PacketConn, msg *m.CoAPMessage) {
	if block := msg.GetBlock2(); block != nil {
		s.block2sendsMX.RLock()
		ch, ok := s.block2sends[msg.GetTokenString()]
		if ok {
			ch <- msg
		}
		s.block2sendsMX.RUnlock()
	}
}

func (s *Server) getResource(msg *m.CoAPMessage) (res r.CoAPResourceHandler, ok bool) {
	path := msg.GetURIPath()
	switch msg.Code {
	case m.POST:
		res, ok = s.postResources[path]
	case m.GET:
		res, ok = s.getResources[path]
	case m.DELETE:
		res, ok = s.deleteResources[path]
	}
	return
}

func (s *Server) resourceProcessor(pc net.PacketConn, msg *m.CoAPMessage, res r.CoAPResourceHandler) {
	result := res(msg)
	if msg.Type == m.NON {
		return
	}

	// Create ACK response with the same ID and given reponse Code
	responseMessage := m.NewCoAPMessageId(m.ACK, result.Code, msg.MessageID)
	if result.Payload != nil {
		responseMessage.Payload = result.Payload
	} else {
		responseMessage.Payload = m.NewEmptyPayload()
	}

	// Replicate Token of the original message if any
	responseMessage.Token = msg.Token

	// Setup additional Content Format description if necessary
	if result.MediaType >= 0 {
		responseMessage.AddOption(m.OptionContentFormat, result.MediaType)
	}

	// validate Observe option (add Option in Response upon registration!)
	if option := msg.GetOption(m.OptionObserve); option != nil && option.IntValue() == 0 {
		responseMessage.AddOption(m.OptionObserve, 1)
	}

	// Validate message scheme
	if msg.GetScheme() == m.COAPS_SCHEME {
		responseMessage.SetSchemeCOAPS()
	}

	responseMessage.CloneOptions(msg, m.OptionBlock1, m.OptionBlock2, m.OptionSelectiveRepeatWindowSize, m.OptionProxySecurityID)
	if responseMessage.Payload.Length() > MAX_PAYLOAD_SIZE {
		s.sendBlock2Response(pc, responseMessage, msg.Sender)
	} else {
		s.send(pc, responseMessage, msg.Sender)
	}
}

type packet struct {
	acked    bool
	attempts int
	lastSend time.Time
	message  *m.CoAPMessage
}

func (s *Server) deleteInProcess(token string) {
	s.inProcessMX.Lock()
	delete(s.inProcess, token)
	s.inProcessMX.Unlock()
}

func (s *Server) sendBlock2Response(pc net.PacketConn, sendsMessage *m.CoAPMessage, addr net.Addr) {
	ch := make(chan *m.CoAPMessage, 1024)

	s.block2sendsMX.Lock()
	s.block2sends[sendsMessage.GetTokenString()] = ch
	s.block2sendsMX.Unlock()

	defer func() {
		s.block2sendsMX.Lock()
		delete(s.block2sends, sendsMessage.GetTokenString())
		close(ch)
		s.block2sendsMX.Unlock()
	}()

	packets := []*packet{}
	state := makeState(sendsMessage)

	emptyAckMessage := m.NewACKEmptyMessage(sendsMessage, state.Windowsize)
	if err := s.send(pc, emptyAckMessage, addr); err != nil {
		return
	}

	for {
		blockMessage, end := m.ConstructNextBlock(m.OptionBlock2, state)
		packets = append(packets, &packet{
			acked:   false,
			message: blockMessage,
		})

		if end {
			break
		}
	}

	shift := 0

	if err := s.sendPacketsToAddr(pc, packets, state.Windowsize, shift, addr); err != nil {
		return
	}

	for {
		select {
		case <-time.After(sumTimeAttempts):
			return
		case resp := <-ch:
			block := resp.GetBlock2()
			if len(packets) < block.BlockNumber {
				continue
			}

			if resp.Code != m.CoapCodeContinue {
				return
			}

			if block.BlockNumber >= len(packets) {
				continue
			}

			packets[block.BlockNumber].acked = true
			if block.BlockNumber != shift {
				continue
			}

			shift++

			for _, p := range packets[shift:] {
				if p.acked {
					shift++
				} else {
					break
				}
			}

			if err := s.sendPacketsToAddr(pc, packets, state.Windowsize, shift, addr); err != nil {
				return
			}
		}
	}
}

func makeState(msg *m.CoAPMessage) *m.StateSend {
	state := new(m.StateSend)
	state.Payload = msg.Payload.Bytes()
	state.Lenght = len(state.Payload)
	state.OrigMessage = msg
	state.BlockSize = MAX_PAYLOAD_SIZE
	numblocks := math.Ceil(float64(state.Lenght) / float64(MAX_PAYLOAD_SIZE))
	if numblocks < DEFAULT_WINDOW_SIZE {
		state.Windowsize = int(numblocks)
	} else {
		state.Windowsize = DEFAULT_WINDOW_SIZE
	}

	return state
}

func (s *Server) send(pc net.PacketConn, msg *m.CoAPMessage, addr net.Addr) error {
	if err := s.securityOutputLayer(pc, msg, addr); err == nil {
		if b, err := m.Serialize(msg); err == nil {
			util.MetricSentMessages.Inc()
			pc.WriteTo(b, addr)
		}
	} else {
		return err
	}

	return nil
}

func (s *Server) deleteBlock1Receive(token string, c chan *m.CoAPMessage) {
	s.block1receiveMX.Lock()
	delete(s.block1receive, token)
	close(c)
	s.block1receiveMX.Unlock()
}

func (s *Server) receiveARQBlock1(pc net.PacketConn, msg *m.CoAPMessage, input chan *m.CoAPMessage) {
	var (
		fullmsg     *m.CoAPMessage
		buf         = make(map[int][]byte)
		totalBlocks = -1
	)

	for {
		select {
		case inputMessage := <-input:
			block := inputMessage.GetBlock1()
			if block == nil || inputMessage.Type != m.CON {
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
				fullmsg = inputMessage
				s.deleteBlock1Receive(msg.GetTokenString(), input)
				break
			}

			var ack *m.CoAPMessage
			w := inputMessage.GetOption(m.OptionSelectiveRepeatWindowSize)
			if w != nil {
				ack = m.AckToWithWindowOffset(nil, inputMessage, m.CoapCodeContinue, w.IntValue(), block.BlockNumber, buf)
			} else {
				ack = m.AckTo(nil, inputMessage, m.CoapCodeContinue)
			}

			if err := s.send(pc, ack, inputMessage.Sender); err != nil {
				s.deleteBlock1Receive(msg.GetTokenString(), input)
				return
			}

		case <-time.After(sumTimeAttempts):
			s.deleteBlock1Receive(msg.GetTokenString(), input)
			return
		}

		if fullmsg != nil {
			break
		}
	}
	s.serveCON(pc, fullmsg)
}
