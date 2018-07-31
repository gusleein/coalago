package coalago

import (
	"errors"
	"time"

	m "github.com/coalalib/coalago/message"
)

var (
	ErrMaxAttempts = errors.New("Max attempts")
)

func pendingMessagesReader(coala *Coala, senderPool chan *m.CoAPMessage, acknowledgePool *ackPool) {
	for {
		message := <-senderPool

		if message.Type != m.CON {
			sendToSocket(coala, message, message.Recipient)
			continue
		}

		if time.Since(message.LastSent).Seconds() < 3 {
			senderPool <- message
			continue
		}

		message.Attempts++

		if message.Attempts > 3 {
			coala.Metrics.ExpiredMessages.Inc()
			callback := acknowledgePool.GetAndDelete(newPoolID(message.MessageID, message.Token, message.Recipient))
			if callback != nil {
				callback(nil, ErrMaxAttempts)
			}
			continue
		}

		message.LastSent = time.Now()
		if message.Attempts > 1 {
			coala.Metrics.Retransmissions.Inc()
		}

		sendToSocket(coala, message, message.Recipient)
	}
}
