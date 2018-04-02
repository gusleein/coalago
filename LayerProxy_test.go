package coalago

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
)

var (
	portForTestProxy  = 2222
	portForTestServer = portForTest

	pathTestProxy = "/testproxy"
)

func TestProxyWithSimpleMessage(t *testing.T) {
	expectedResponse := []byte("Hello from Coala!:)")

	c := newClient()
	s := newServer()
	newProxy()

	addResourceForProxy(s, expectedResponse)

	message := newProxiedMessageForTest(expectedResponse)
	address, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", portForTestProxy))
	resp, err := c.Send(message, address)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(resp.Payload.Bytes(), expectedResponse) {
		panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Payload.Bytes()))
	}
}

func TestProxyWithBigMessage(t *testing.T) {
	var stringPayload string
	for i := 0; i < 4; i++ {
		stringPayload += strings.Repeat(fmt.Sprintf("%d", i), MAX_PAYLOAD_SIZE)
	}

	expectedResponse := []byte(stringPayload)
	c := newClient()
	s := newServer()
	newProxy()

	addResourceForProxy(s, expectedResponse)

	message := newProxiedMessageForTest(expectedResponse)
	address, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", portForTestProxy))
	resp, err := c.Send(message, address)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(resp.Payload.Bytes(), expectedResponse) {
		panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Payload.Bytes()))
	}
}

func newProxy() *Coala {
	p := NewListen(portForTestProxy)
	p.EnableProxy()
	return p
}

func newProxiedMessageForTest(expectedPayload []byte) *m.CoAPMessage {
	message := m.NewCoAPMessage(m.CON, m.POST)
	message.SetURIPath(pathTestProxy)
	message.Payload = m.NewBytesPayload(expectedPayload)
	message.SetProxyURI(fmt.Sprintf("coap://127.0.0.1:%d", portForTestServer))

	return message
}

func addResourceForProxy(coala *Coala, expectedPayload []byte) *Coala {
	coala.AddPOSTResource(pathTestProxy, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		if !bytes.Equal(message.Payload.Bytes(), expectedPayload) {
			panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedPayload, message.Payload.Bytes()))
		}

		resp := m.NewBytesPayload(expectedPayload)
		return resource.NewResponse(resp, m.CoapCodeChanged)
	})

	return coala
}
