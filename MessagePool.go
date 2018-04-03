package coalago

import (
	"errors"
	"time"

	m "github.com/coalalib/coalago/message"
)

var (
	ErrMaxAttempts = errors.New("Max attempts")
)

func pendingMessagesReader(coala *Coala, senderPool *Queue, acknowledgePool *ackPool) {
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
			callback := acknowledgePool.GetAndDelete(newPoolID(message.MessageID, message.Token, message.Recipient))
			if callback != nil {
				go callback(nil, errors.New("mac attempts"))
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
