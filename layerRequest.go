package coalago

import (
	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
)

func requestOnReceive(server *Server, message *m.CoAPMessage) bool {
	if message.Code <= 0 || message.Code > 4 {
		return true
	}
	resource := server.getResourceForPathAndMethod(message.GetURIPath(), message.GetMethod())
	if resource == nil {
		if message.Type == m.CON {
			return noResource(server, message)
		}
		return false
	}

	if resource.Method != message.GetMethod() {
		return methodNotAllowed(server, message)
	}

	if handlerResult := resource.Handler(message); handlerResult != nil {
		if message.Type == m.NON {
			return false
		}
		return returnResultFromResource(server, message, handlerResult)
	}

	if message.Type == m.CON {
		return noResultResourceHandler(server, message)
	}

	return false
}

func methodNotAllowed(server *Server, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeMethodNotAllowed, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("Method is not allowed for requested resource")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2)
	server.sr.SendTo(responseMessage, message.Sender)
	return false
}

func returnResultFromResource(server *Server, message *m.CoAPMessage, handlerResult *resource.CoAPResourceHandlerResult) bool {
	// @TODO: Validate Response code! handlerResult.Code

	// Create ACK response with the same ID and given reponse Code
	responseMessage := m.NewCoAPMessageId(m.ACK, handlerResult.Code, message.MessageID)
	responseMessage.Payload = handlerResult.Payload

	// Replicate Token of the original message if any
	responseMessage.Token = message.Token

	// Setup additional Content Format description if necessary
	if handlerResult.MediaType >= 0 {
		responseMessage.AddOption(m.OptionContentFormat, handlerResult.MediaType)
	}

	// validate Observe option (add Option in Response upon registration!)
	if option := message.GetOption(m.OptionObserve); option != nil && option.IntValue() == 0 {
		responseMessage.AddOption(m.OptionObserve, 1)
	}

	// Validate message scheme
	if message.GetScheme() == m.COAPS_SCHEME {
		responseMessage.SetSchemeCOAPS()
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2, m.OptionSelectiveRepeatWindowSize)

	_, err := server.sr.SendTo(responseMessage, message.Sender)
	return err != nil
}

func noResultResourceHandler(server *Server, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeInternalServerError, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("No Result was returned by Resource Handler")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2)

	_, err := server.sr.SendTo(responseMessage, message.Sender)
	return err != nil
}

func noResource(server *Server, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeNotFound, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("Requested resource " + message.GetURIPath() + " does not exist")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2)
	responseMessage.Recipient = message.Sender

	_, err := server.sr.SendTo(responseMessage, message.Sender)
	return err != nil
}
