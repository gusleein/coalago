package message

type CoAPMessageHandler func(message *CoAPMessage, err error)
