package coalago

import (
	"errors"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/queue"
)

func messagePoolSender(coala *Coala, senderPool *queue.Queue, receiverPool *sync.Map) {
	for {
		im := senderPool.Pop()

		if im == nil {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		message := im.(*m.CoAPMessage)

		if message.Type == m.ACK {
			senderPool.Delete(message.GetMessageIDString() + message.Recipient.String())
			sendToSocket(coala, message, message.Recipient)
			continue
		}

		if time.Since(message.LastSent).Seconds() < 3 {
			continue
		}

		message.Attempts++

		if message.Attempts > 3 {
			ci, _ := receiverPool.Load(message.GetMessageIDString() + message.Recipient.String())
			if ci != nil {
				receiverPool.Delete(message.GetMessageIDString() + message.Recipient.String())
				callback := ci.(CoalaCallback)
				go callback(nil, errors.New("timed out"))
			}
			senderPool.Delete(message.GetMessageIDString() + message.Recipient.String())
			continue
		}

		message.LastSent = time.Now()
		sendToSocket(coala, message, message.Recipient)
	}
}
