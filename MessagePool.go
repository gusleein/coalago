package coalago

import (
	"errors"
	"sync"
	"time"

	m "github.com/coalalib/coalago/message"
)

var (
	ErrMaxAttempts = errors.New("Max attempts")
)

func pendingMessagesReader(coala *Coala, senderPool *Queue, receiverPool *sync.Map) {
	for {
		im := senderPool.Pop()

		if im == nil {
			continue
		}

		message := im.Value.(*m.CoAPMessage)

		if message.Type != m.CON {
			im := senderPool.Remove(im)
			if im != nil {
				message = im.Value.(*m.CoAPMessage)
				sendToSocket(coala, message, message.Recipient)
			}
			continue
		}

		if time.Since(message.LastSent).Seconds() < 3 {
			continue
		}

		message.Attempts++

		if message.Attempts > 3 {
			coala.Metrics.ExpiredMessages.Inc()
			ci, _ := receiverPool.Load(message.GetMessageIDString() + message.Recipient.String())
			if ci != nil {
				receiverPool.Delete(message.GetMessageIDString() + message.Recipient.String())
				callback := ci.(CoalaCallback)
				go callback(nil, errors.New("timed out"))
			}
			senderPool.Remove(im)
			continue
		}

		message.LastSent = time.Now()
		if message.Attempts > 1 {
			coala.Metrics.Retransmissions.Inc()
		}
		sendToSocket(coala, message, message.Recipient)
	}
}
