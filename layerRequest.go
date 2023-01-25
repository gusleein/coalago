package coalago

import (
	m "github.com/coalalib/coalago/message"
	r "github.com/coalalib/coalago/resource"
)

func requestOnReceive(resource *r.CoAPResource, sr *transport, message *m.CoAPMessage) bool {
	if message.Code < 0 || message.Code > 4 {
		return true
	}

	if isPing(message) {
		return returnPing(sr, message)
	}

	if resource == nil {
		if message.Type == m.CON {
			return noResource(sr, message)
		}
		return false
	}

	if resource.Method != message.GetMethod() {
		return methodNotAllowed(sr, message)
	}

	if handlerResult := resource.Handler(message); handlerResult != nil {
		if message.Type == m.NON {
			return false
		}
		return returnResultFromResource(sr, message, handlerResult)
	}

	if message.Type == m.CON {
		return noResultResourceHandler(sr, message)
	}
	return false
}

func isPing(message *m.CoAPMessage) bool {
	return message.Type == m.CON && message.Code == m.CoapCodeEmpty
}

func returnPing(sr *transport, message *m.CoAPMessage) bool {
	resp := m.NewCoAPMessage(m.RST, m.CoapCodeEmpty)
	resp.MessageID = message.MessageID
	copy(resp.Token, message.Token)
	_, err := sr.SendTo(resp, message.Sender)

	return err != nil
}

func methodNotAllowed(sr *transport, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeMethodNotAllowed, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("Method is not allowed for requested resource")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2, m.OptionProxySecurityID)
	sr.SendTo(responseMessage, message.Sender)
	return false
}

func returnResultFromResource(sr *transport, message *m.CoAPMessage, handlerResult *r.CoAPResourceHandlerResult) bool {
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
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2, m.OptionSelectiveRepeatWindowSize, m.OptionProxySecurityID)

	_, err := sr.SendTo(responseMessage, message.Sender)
	return err != nil
}

func noResultResourceHandler(sr *transport, message *m.CoAPMessage) bool {
	// responseMessage := NewCoAPMessageId(ACK, CoapCodeInternalServerError, message.MessageID)
	// responseMessage.Payload = NewStringPayload("No Result was returned by Resource Handler")
	// if message.Token != nil && len(message.Token) > 0 {
	// 	responseMessage.Token = message.Token
	// }
	// responseMessage.CloneOptions(message, OptionBlock1, OptionBlock2)

	// _, err := server.sr.SendTo(responseMessage, message.Sender)
	// return err != nil
	return false
}

func noResource(sr *transport, message *m.CoAPMessage) bool {
	responseMessage := m.NewCoAPMessageId(m.ACK, m.CoapCodeNotFound, message.MessageID)
	responseMessage.Payload = m.NewStringPayload("Requested resource " + message.GetURIPath() + " does not exist")
	if message.Token != nil && len(message.Token) > 0 {
		responseMessage.Token = message.Token
	}
	responseMessage.CloneOptions(message, m.OptionBlock1, m.OptionBlock2, m.OptionProxySecurityID)
	responseMessage.Recipient = message.Sender

	_, err := sr.SendTo(responseMessage, message.Sender)
	return err != nil
}
