package coalago

import (
	"net"
	"net/url"
	"strings"

	m "github.com/coalalib/coalago/message"
)

type ProxyLayer struct{}

func (layer *ProxyLayer) OnReceive(coala *Coala, message *m.CoAPMessage) bool {
	if !coala.IsProxyMode() {
		return true
	}

	if message.IsProxied() {
		if !isValideProxyMode(coala, message) {
			return false
		}

		proxyMessage, address, err := makeMessageFromProxyToRecepient(message)

		if err != nil {
			sendResponseFromProxyToSenderAckMessage(coala, message, m.CoapCodeBadOption, "")
			return false
		}

		coala.GetAllPools().ProxySessions.Store(string(proxyMessage.Token)+address.String(), message.Sender)
		coala.GetAllPools().ProxySessions.Store(string(proxyMessage.Token)+message.Sender.String(), address)

		sendToSocket(coala, proxyMessage, address)

		return false
	}

	addrSender, ok := coala.GetAllPools().ProxySessions.Load(string(message.Token) + message.Sender.String())
	if !ok {
		return true
	}

	message.IsProxies = true
	sendToSocket(coala, message, addrSender.(*net.UDPAddr))

	return false
}

func (layer *ProxyLayer) OnSend(coala *Coala, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	if proxyURI := message.GetOptionProxyURIasString(); proxyURI != "" {
		message.RemoveOptions(m.OptionURIPath)
		message.RemoveOptions(m.OptionURIQuery)

		coala.GetAllPools().SendMessages.Set(message.GetTokenString()+address.String(), message)
	}

	if !coala.IsProxyMode() {
		return true, nil
	}

	if message.IsProxies {
		return true, nil
	}

	_, ok := coala.GetAllPools().ProxySessions.Load(message.GetProxyKeySender(address))
	if ok {
		return false, nil
	}

	return true, nil
}

// Sends ACK message to sender from proxy
func sendResponseFromProxyToSenderAckMessage(coala *Coala, message *m.CoAPMessage, code m.CoapCode, payload string) error {
	responseMessage := makeMessageFromProxyToSender(message, code)
	responseMessage.SetStringPayload(payload)
	sendToSocket(coala, responseMessage, message.Sender)
	return nil
}

func isValideProxyMode(coala *Coala, message *m.CoAPMessage) bool {
	proxyURI := message.GetOptionProxyURIasString()
	proxyScheme := message.GetOptionProxyScheme()
	if !coala.IsProxyMode() {
		sendResponseFromProxyToSenderAckMessage(coala, message, m.CoapCodeProxyingNotSupported, "")
		return false
	}

	if proxyScheme != m.COAP_SCHEME && proxyScheme != m.COAPS_SCHEME &&
		!strings.HasPrefix(proxyURI, "coap") && !strings.HasPrefix(proxyURI, "coaps") {

		log.Error("Proxy Scheme is invalid", proxyScheme, proxyURI)
		sendResponseFromProxyToSenderAckMessage(coala, message, m.CoapCodeBadRequest, "Proxy Scheme is invalid")
		return false
	}
	return true
}

// Prepares a message to send to the final recipient
func makeMessageFromProxyToRecepient(message *m.CoAPMessage) (proxyMessage *m.CoAPMessage, address *net.UDPAddr, err error) {
	message.RemoveOptions(m.OptionURIPath)
	proxyURI := message.GetOptionProxyURIasString()

	parsedURL, err := url.Parse(proxyURI)
	if err != nil {
		log.Error("Error of parsing the ProxyURI:", err)
		return
	}

	proxyMessage = message.Clone(true)
	proxyMessage.SetURIPath(parsedURL.Path)
	queries := m.ParseQuery(parsedURL.RawQuery)

	for k, v := range queries {
		proxyMessage.SetURIQuery(k, v[0])
	}

	deleteProxyOptions(proxyMessage)
	proxyMessage.IsProxies = true

	if observeOpt := message.GetOption(m.OptionObserve); observeOpt != nil {
		message.AddOptions([]*m.CoAPMessageOption{observeOpt})
	}

	address, err = net.ResolveUDPAddr("udp", parsedURL.Host)
	return
}

func makeMessageFromProxyToSender(message *m.CoAPMessage, code m.CoapCode) (responseMessage *m.CoAPMessage) {
	responseMessage = message.Clone(false)
	deleteProxyOptions(responseMessage)

	responseMessage.Type = m.ACK
	responseMessage.Code = code

	return
}

func deleteProxyOptions(message *m.CoAPMessage) {
	message.RemoveOptions(m.OptionProxyScheme)
	message.RemoveOptions(m.OptionProxyURI)
}
