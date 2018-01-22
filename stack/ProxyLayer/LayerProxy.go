package ProxyLayer

import (
	"net"
	"net/url"
	"strings"

	"github.com/coalalib/coalago/common"
	logging "github.com/op/go-logging"

	m "github.com/coalalib/coalago/message"
)

var log = logging.MustGetLogger("ProxyLayer")

type ProxyLayer struct{}

func (layer *ProxyLayer) OnReceive(coala common.SenderIface, message *m.CoAPMessage) bool {
	if !coala.IsProxyMode() {
		return true
	}

	if message.IsProxied() {
		if !isValideProxyMode(coala, message) {
			return false
		}

		proxyMessage, address, err := makeMessageFromProxyToRecepient(message)
		if err != nil {
			sendResponseProxyAckMessage(coala, message, m.CoapCodeBadOption, "")
			return false
		}

		coala.GetAllPools().ProxySessions.Store(string(proxyMessage.Token)+address.String(), message.Sender)

		coala.Send(proxyMessage, address)

		return false
	}

	addrSender, ok := coala.GetAllPools().ProxySessions.Load(message.GetProxyKeyReceiver())
	if !ok {
		return true
	}

	message.IsProxies = true
	response, err := coala.Send(message, addrSender.(*net.UDPAddr))
	if err != nil && message.Type != m.ACK {
		log.Error(err, message.ToReadableString())
		sendResponseProxyAckMessage(coala, message, m.CoapCodeBadGateway, "Unable to send message to proxy sender")
	} else if response != nil {
		response.IsProxies = true
		_, err := coala.Send(response, message.Sender)
		if err != nil {
			log.Warning("An attempt to send a response from proxy. Error:", err, response.ToReadableString())
		}
	}

	return false
}

func (layer *ProxyLayer) OnSend(coala common.SenderIface, message *m.CoAPMessage, address *net.UDPAddr) (bool, error) {
	if proxyURI := message.GetOptionProxyURIasString(); proxyURI != "" {
		message.RemoveOptions(m.OptionURIPath)
		message.RemoveOptions(m.OptionURIQuery)

		coala.GetAllPools().SendMessages.Set(message.GetTokenString()+address.String(), message)
	}

	if !coala.IsProxyMode() {
		return true, nil
	}

	if message.IsProxies {
		return false, nil
	}

	_, ok := coala.GetAllPools().ProxySessions.Load(message.GetProxyKeySender(address))
	if ok {
		return false, nil
	}

	return true, nil
}

// Sends ACK message to sender from proxy
func sendResponseProxyAckMessage(coala common.SenderIface, message *m.CoAPMessage, code m.CoapCode, payload string) error {
	responseMessage := makeMessageFromProxyToSender(message, code)
	responseMessage.SetStringPayload(payload)
	_, err := coala.Send(responseMessage, message.Sender)
	return err
}

func isValideProxyMode(coala common.SenderIface, message *m.CoAPMessage) bool {
	proxyURI := message.GetOptionProxyURIasString()
	proxyScheme := message.GetOptionProxyScheme()
	if !coala.IsProxyMode() {
		sendResponseProxyAckMessage(coala, message, m.CoapCodeProxyingNotSupported, "")
		return false
	}

	if proxyScheme != m.COAP_SCHEME && proxyScheme != m.COAPS_SCHEME &&
		!strings.HasPrefix(proxyURI, "coap") && !strings.HasPrefix(proxyURI, "coaps") {

		log.Error("Proxy Scheme is invalid", proxyScheme, proxyURI)
		sendResponseProxyAckMessage(coala, message, m.CoapCodeBadRequest, "Proxy Scheme is invalid")
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
