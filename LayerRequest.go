package coalago

import (
	"net"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
)

type RequestLayer struct{}

func (layer *RequestLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if message.Code <= 0 || message.Code > 4 {
		return true
	}
	resources := coala.GetResourcesForPathAndMethod(message.GetURIPath(), message.GetMethod())

	if len(resources) == 0 {
		resources = coala.GetResourcesForPath(message.GetURIPath())
	}

	if len(resources) == 0 && message.Type == m.CON {
		return noResource(coala, message)
	}

	for _, resource := range resources {
		// check the Method (405 error code handler)
		if resource.Method != message.GetMethod() {
			return methodNotAllowed(coala, message)
		}

		if handlerResult := resource.Handler(message); handlerResult != nil {
			return returnResultFromResource(coala, message, handlerResult)
		}

		if message.Type == m.CON {
			return noResultResourceHandler(coala, message)
		}
	}

	return false
}

func (layer *RequestLayer) OnSend(coala *Coala, message *m.CoAPMessage, address net.Addr) (bool, error) {
	return true, nil
}

func methodNotAllowed(coala *Coala, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeMethodNotAllowed, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("Method is not allowed for requested resource")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}

	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2)
	coala.Send(responseMessage, message.Sender)

	return false
}

func returnResultFromResource(coala *Coala, message *m.CoAPMessage, handlerResult *resource.CoAPResourceHandlerResult) bool {
	// @TODO: Validate Response code! handlerResult.Code

	// Create ACK response with the same ID and given reponse Code
	responseMessage := m.NewCoAPMessageId(m.ACK, handlerResult.Code, message.MessageID)
	responseMessage.Payload = handlerResult.Payload

	// Replicate Token of the original message if any
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}

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

	_, err := coala.Send(responseMessage, message.Sender)
	if err != nil {
		return true
	}
	return false
}

func noResultResourceHandler(coala *Coala, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeInternalServerError, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("No Result was returned by Resource Handler")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2)

	_, err := coala.Send(responseMessage, message.Sender)
	if err != nil {
		return true
	}
	return false
}

func noResource(coala *Coala, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeNotFound, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("Requested resource " + message.GetURIPath() + " does not exist")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2)

	_, err := coala.Send(responseMessage, message.Sender)

	if err != nil {
		return true
	}
	return false
}
