package coalago

// import (
// 	"net"
// 	"net/url"
// 	"strings"

//
// 	"github.com/labstack/gommon/log"
// 	cache "github.com/patrickmn/go-cache"
// )

// // type ProxyLayer struct{}

// type Proxy struct {
// 	conn               *net.UDPConn
// 	forwardConnections *cache.Cache
// 	reverseConnections *cache.Cache
// }

// func (p *Proxy) Listen(addr string) error {
// 	udpAddr, err := net.ResolveUDPAddr("udp", addr)
// 	if err != nil {
// 		return err
// 	}

// 	conn, err := net.ListenUDP("udp", udpAddr)
// 	if err != nil {
// 		return err
// 	}

// 	var (
// 		buffer = make([]byte, 1500)
// 		n      int
// 		sender net.Addr
// 	)

// 	for {
// 		n, sender, err = conn.ReadFrom(buffer)
// 		if err != nil {
// 			return err
// 		}
// 		if n == 0 {
// 			continue
// 		}

// 		message, err := Deserialize(buffer)
// 		if err != nil {
// 			continue
// 		}
// 		message.Sender = sender

// 		if message.IsProxied() {
// 			if !isValideProxyMode(coala, message) {
// 				return false
// 			}

// 			proxyMessage, address, err := makeMessageFromProxyToRecepient(message)

// 			if err != nil {
// 				sendResponseFromProxyToSenderAckMessage(coala, message, CoapCodeBadOption, "")
// 				return false
// 			}

// 			coala.GetAllPools().ProxyPool.Set(string(proxyMessage.Token)+address.String(), message.Sender)
// 			coala.GetAllPools().ProxyPool.Set(string(proxyMessage.Token)+message.Sender.String(), address)

// 			coala.Metrics.ProxiedMessages.Inc()
// 			sendToSocket(coala, proxyMessage, address)

// 			return false
// 		}
// 	}

// }

// func sendToSocket(conn net.UDPConn, message   *CoAPMessage)

// func proxyReceive(tr *transport, message   *CoAPMessage) bool {
// 	if !coala.IsProxyMode() {
// 		return true
// 	}

// 	if message.IsProxied() {
// 		if !isValideProxyMode(coala, message) {
// 			return false
// 		}

// 		proxyMessage, address, err := makeMessageFromProxyToRecepient(message)

// 		if err != nil {
// 			sendResponseFromProxyToSenderAckMessage(coala, message, CoapCodeBadOption, "")
// 			return false
// 		}

// 		coala.GetAllPools().ProxyPool.Set(string(proxyMessage.Token)+address.String(), message.Sender)
// 		coala.GetAllPools().ProxyPool.Set(string(proxyMessage.Token)+message.Sender.String(), address)

// 		coala.Metrics.ProxiedMessages.Inc()
// 		sendToSocket(coala, proxyMessage, address)

// 		return false
// 	}

// 	addrSender := coala.GetAllPools().ProxyPool.Get(string(message.Token) + message.Sender.String())
// 	if addrSender == nil {
// 		return true
// 	}

// 	message.IsProxies = true
// 	sendToSocket(coala, message, addrSender)

// 	return false
// }

// func proxySend(tr *transport, message   *CoAPMessage, address net.Addr) (bool, error) {
// 	if !coala.IsProxyMode() {
// 		return true, nil
// 	}

// 	if message.IsProxies {
// 		return true, nil
// 	}

// 	addr := coala.GetAllPools().ProxyPool.Get(message.GetProxyKeySender(address))
// 	if addr != nil {
// 		return false, nil
// 	}

// 	return true, nil
// }

// // Sends ACK message to sender from proxy
// func sendResponseFromProxyToSenderAckMessage(coala *Coala, message   *CoAPMessage, code CoapCode, payload string) error {
// 	responseMessage := makeMessageFromProxyToSender(message, code)
// 	responseMessage.SetStringPayload(payload)
// 	sendToSocket(coala, responseMessage, message.Sender)
// 	return nil
// }

// func isValideProxyMode(coala *Coala, message   *CoAPMessage) bool {
// 	proxyURI := message.GetOptionProxyURIasString()
// 	proxyScheme := message.GetOptionProxyScheme()
// 	if !coala.IsProxyMode() {
// 		sendResponseFromProxyToSenderAckMessage(coala, message, CoapCodeProxyingNotSupported, "")
// 		return false
// 	}

// 	if proxyScheme != COAP_SCHEME && proxyScheme != COAPS_SCHEME &&
// 		!strings.HasPrefix(proxyURI, "coap") && !strings.HasPrefix(proxyURI, "coaps") {

// 		log.Error("Proxy Scheme is invalid", proxyScheme, proxyURI)
// 		sendResponseFromProxyToSenderAckMessage(coala, message, CoapCodeBadRequest, "Proxy Scheme is invalid")
// 		return false
// 	}
// 	return true
// }

// // Prepares a message to send to the final recipient
// func makeMessageFromProxyToRecepient(message   *CoAPMessage) (proxyMessage   *CoAPMessage, address net.Addr, err error) {
// 	proxyURI := message.GetOptionProxyURIasString()

// 	parsedURL, err := url.Parse(proxyURI)
// 	if err != nil {
// 		log.Error("Error of parsing the ProxyURI:", err)
// 	}
// 	address, err = net.ResolveUDPAddr("udp4", parsedURL.Host)
// 	if err != nil {
// 		log.Error("Error of parsing the ProxyURI:", err)
// 		return
// 	}

// 	proxyMessage = message.Clone(true)

// 	deleteProxyOptions(proxyMessage)
// 	proxyMessage.IsProxies = true

// 	if observeOpt := message.GetOption( OptionObserve); observeOpt != nil {
// 		message.AddOptions([]  *CoAPMessageOption{observeOpt})
// 	}

// 	return
// }

// func makeMessageFromProxyToSender(message   *CoAPMessage, code CoapCode) (responseMessage   *CoAPMessage) {
// 	responseMessage = message.Clone(false)
// 	deleteProxyOptions(responseMessage)

// 	responseMessage.Type = ACK
// 	responseMessage.Code = code

// 	return
// }

// func deleteProxyOptions(message   *CoAPMessage) {
// 	message.RemoveOptions( OptionProxyScheme)
// 	message.RemoveOptions( OptionProxyURI)
// }
