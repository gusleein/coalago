package coalago

import (
	"net"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/ProxyLayer"
)

func (coala *Coala) listenConnection() {
	for {
		readBuf, lenght, senderAddr, err := coala.connection.Read()
		if err != nil {
			log.Error(err)
			go coala.listenConnection()
			return
		}
		coala.Metrics.ReceivedMessages.Inc()
		go rawBufferHandler(coala, readBuf, lenght, senderAddr)
	}
}

func rawBufferHandler(coala *Coala, readBuf []byte, length int, senderAddr *net.UDPAddr) {
	message, err := m.Deserialize(readBuf[:length])
	if err != nil {
		log.Error("Error while making CoAPMessage Object", err)
		return
	}
	message.Sender = senderAddr

	log.Debugf("Receiving message: %s, from: %s", message.ToReadableString(), senderAddr.String())

	isNext := receive(coala, message, senderAddr)
	if !isNext {
		return
	}

	if message.IsRequest() {
		return
	}
	respChannel, ok := coala.incomingMessages.Load(message.MessageID)
	if !ok {
		return
	}

	defer func() {
		recover()
	}()

	respChannel.(chan *m.CoAPMessage) <- message
}

func receive(coala *Coala, message *m.CoAPMessage, senderAddr *net.UDPAddr) (isNext bool) {
	proxyLayer := ProxyLayer.ProxyLayer{}
	if !proxyLayer.OnReceive(coala, message) {
		return false
	}

	coala.receiveLayerStack.OnReceive(message)
	return true
}
