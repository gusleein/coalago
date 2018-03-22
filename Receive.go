package coalago

import (
	"net"
	"sync"

	m "github.com/coalalib/coalago/message"
)

// buffer pool to reduce GC
var buffers = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1500)
	},
}

func (coala *Coala) listenConnection() {
	for {
		readBuf := [1500]byte{}

		n, senderAddr, err := coala.connection.Read(readBuf[:])
		if err != nil {
			log.Error(err)
			go coala.listenConnection()
			return
		}
		coala.Metrics.ReceivedMessages.Inc()

		message, err := m.Deserialize(readBuf[:n])
		if err != nil {
			log.Error("Error while making CoAPMessage Object", err)
			continue
		}

		go rawBufferHandler(coala, message, senderAddr)
	}
}

func rawBufferHandler(coala *Coala, message *m.CoAPMessage, senderAddr net.Addr) {
	message.Sender = senderAddr
	// fmt.Printf("\n|<----- %s\t%s\tPAYLOAD: %s\n\n", senderAddr, message.ToReadableString(), "message.Payload.String()")

	if coala.receiveLayerStack.OnReceive(message) {
		ic, _ := coala.reciverPool.Load(message.GetMessageIDString() + message.Sender.String())
		if ic != nil {
			coala.reciverPool.Delete(message.GetMessageIDString() + message.Sender.String())
			callback := ic.(CoalaCallback)
			callback(message, nil)
		}
	}
}
