package observer

import (
	"time"

	m "github.com/coalalib/coalago/message"
)

//Key for Observer sender+|+Token

type CoAPObserverCallback struct {
	Key               string
	RegisteredMessage *m.CoAPMessage
	Condition         *CoAPObserverCondition
	MaxAge            int
	LastUpdate        int64
	ordering          uint32
	inProcess         bool
}

type CoAPObserverCondition func(callback *CoAPObserverCallback) bool

func NewObserverCallback(msg *m.CoAPMessage, callback *CoAPObserverCondition) *CoAPObserverCallback {
	return &CoAPObserverCallback{
		Condition:         callback,
		MaxAge:            maxAgeFromMessage(msg),
		RegisteredMessage: msg,
		Key:               msg.Sender.String() + "|" + msg.GetTokenString(),
		ordering:          2,
		inProcess:         false,
		LastUpdate:        time.Now().Unix(),
	}
}

func maxAgeFromMessage(msg *m.CoAPMessage) int {
	maxAgeOption := msg.GetOption(m.OptionMaxAge)
	var maxAge int
	if maxAgeOption == nil {
		maxAge = DEFAULT_MAX_AGE
	} else {
		maxAge = maxAgeOption.IntValue()
		if maxAge == 0 {
			maxAge = DEFAULT_MAX_AGE
		}
	}
	return maxAge
}

func (callback *CoAPObserverCallback) getNextOrdering() uint32 {
	if callback.ordering == 0 {
		callback.ordering = 2
	} else if callback.ordering >= 16777215 {
		callback.ordering = 1
	} else {
		callback.ordering++
	}

	return callback.ordering
}
