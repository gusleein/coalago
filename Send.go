package coalago

import (
	"errors"
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/ARQLayer"
	"github.com/coalalib/coalago/stack/ProxyLayer"
)

const TIMEOUT_RTT = 3

func (coala *Coala) Send(message *m.CoAPMessage, address *net.UDPAddr) (response *m.CoAPMessage, err error) {
	nonACK := message.Type == m.ACK
	response, err = coala.send(message, address, nonACK)

	return
}

func (coala *Coala) send(message *m.CoAPMessage, address *net.UDPAddr, nonACK bool) (response *m.CoAPMessage, err error) {
	proxyLayer := ProxyLayer.ProxyLayer{}
	isContinue, err := proxyLayer.OnSend(coala, message, address)
	if err != nil {
		return nil, err
	}
	if isContinue {
		response, isContinue := ARQLayer.ARQSendHandler(coala, message, address)
		if !isContinue {
			return response, nil
		}
		_, err := coala.sendLayerStack.OnSend(message, address)
		if err != nil {
			return nil, err
		}
	}

	data, err := m.Serialize(message)
	if err != nil {
		return nil, err
	}

	respChannel := make(chan *m.CoAPMessage)
	defer close(respChannel)

	coala.incomingMessages.Store(message.MessageID, respChannel)
	log.Debug("Send message: ", message.ToReadableString(), "To:", address.String())
	_, err = coala.connection.WriteTo(data, address)
	coala.Metrics.SentMessages.Inc()
	if err != nil {
		coala.Metrics.SentMessageError.Inc()
		return nil, err
	}
	if nonACK {
		return nil, nil
	}

	var ok bool
	select {
	case <-time.After(time.Second * TIMEOUT_RTT):
		coala.Metrics.ExpiredMessages.Inc()
		return nil, errors.New("Timeout")
	case response, ok = <-respChannel:
		if !ok {
			return nil, errors.New("Timeout")
		}
		break
	}

	if arqRespChan := coala.Pools.ARQRespMessages.Get(getBufferKeyForSend(message, address)); arqRespChan != nil {

	LabelNext:
		select {
		case <-time.After(time.Second * TIMEOUT_RTT):
			coala.Metrics.ExpiredMessages.Inc()
			return nil, errors.New("Timeout ARQ")

		case arqResp, ok := <-arqRespChan:
			if !ok {
				err = errors.New("Timeout. Channel is nil")
			}
			response = arqResp.Message
			if arqResp.IsNext {
				break LabelNext
			}
		}
	}

	coala.Pools.ARQRespMessages.Delete(getBufferKeyForSend(message, address))
	coala.Pools.ARQBuffers.Delete(getBufferKeyForSend(message, address))

	return
}

func getBufferKeyForReceive(msg *m.CoAPMessage) string {
	return msg.Sender.String() + msg.GetTokenString()
}

func getBufferKeyForSend(msg *m.CoAPMessage, address *net.UDPAddr) string {
	return address.String() + msg.GetTokenString()
}
