package coalago

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
)

var (
	portForTest      int = 1111
	pathTestBlock1       = "/testblock1"
	pathTestBlock2       = "/testblock2"
	pathTestBlockMix     = "/testblockmix"
)

func TestSimple(t *testing.T) {
	var stringPayload string

	stringPayload = "hello"

	expectedPayload := []byte(stringPayload)
	expectedResponse := []byte("Hello from Coala!:)")

	c := newClient()
	s := newServer()
	addResourceForBlock1(s, expectedPayload, expectedResponse)

	var wg sync.WaitGroup
	// var count int
	for i := 0; i < 4000; i++ {
		wg.Add(1)
		go func() {
			message := newMessageForTestBlock1(expectedPayload)
			address, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", portForTest))
			// count++
			resp, err := c.Send(message, address)
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(resp.Payload.Bytes(), expectedResponse) {
				panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Payload.Bytes()))
			}
			wg.Done()
		}()
	}

	wg.Wait()

	// time.Sleep(9 * time.Second)

}

func TestBlock1(t *testing.T) {
	var stringPayload string

	stringPayload += strings.Repeat("a", 10*MAX_PAYLOAD_SIZE)

	expectedPayload := []byte(stringPayload)
	expectedResponse := []byte("Hello from Coala!:)")

	c := newClient()
	s := newServer()
	addResourceForBlock1(s, expectedPayload, expectedResponse)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			message := newMessageForTestBlock1(expectedPayload)
			address, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", portForTest))
			resp, err := c.Send(message, address)
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(resp.Payload.Bytes(), expectedResponse) {
				panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Payload.Bytes()))
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestBlock2(t *testing.T) {
	var stringPayload string
	for i := 0; i < 100; i++ {
		stringPayload += strings.Repeat(fmt.Sprintf("%d", i), MAX_PAYLOAD_SIZE)
	}

	expectedResponse := []byte(stringPayload)
	c := newClient()
	s := newServer()
	addResourceForBlock2(s, expectedResponse)
	message := newMessageForTestBlock2()
	address, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", portForTest))
	resp, err := c.Send(message, address)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(resp.Payload.Bytes(), expectedResponse) {
		panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Payload.Bytes()))
	}
}

func TestBlockMix(t *testing.T) {
	var stringPayload string
	for i := 0; i < 1000; i++ {
		stringPayload += strings.Repeat(fmt.Sprintf("%d", i), MAX_PAYLOAD_SIZE)
	}

	expectedResponse := []byte(stringPayload)

	c := newClient()
	s := newServer()
	addResourceForBlockMix(s, expectedResponse)
	var wg sync.WaitGroup
	address, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", portForTest))

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			message := newMessageForTestMix(expectedResponse)
			resp, err := c.Send(message, address)
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(resp.Payload.Bytes(), expectedResponse) {
				panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Payload.Bytes()))
			}
		}()
	}

	wg.Wait()
}

func newClient() *Coala {
	return NewListen(0)
}

func newServer() *Coala {
	return NewListen(portForTest)
}

func addResourceForBlock1(coala *Coala, expectedPayload []byte, expectedResponse []byte) *Coala {
	coala.AddPOSTResource(pathTestBlock1, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		if !bytes.Equal(message.Payload.Bytes(), expectedPayload) {
			panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedPayload, message.Payload.Bytes()))
		}

		resp := m.NewBytesPayload(expectedResponse)
		return resource.NewResponse(resp, m.CoapCodeChanged)
	})
	return coala
}

func addResourceForBlock2(coala *Coala, expectedResponse []byte) *Coala {
	coala.AddGETResource(pathTestBlock2, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		resp := m.NewBytesPayload(expectedResponse)
		return resource.NewResponse(resp, m.CoapCodeContent)
	})
	return coala
}

func addResourceForBlockMix(coala *Coala, expectedPayload []byte) *Coala {
	coala.AddPOSTResource(pathTestBlockMix, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		if !bytes.Equal(message.Payload.Bytes(), expectedPayload) {
			panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedPayload, message.Payload.Bytes()))
		}

		resp := m.NewBytesPayload(expectedPayload)
		return resource.NewResponse(resp, m.CoapCodeContent)
	})
	return coala
}

func newMessageForTestBlock1(expectedPayload []byte) *m.CoAPMessage {
	message := m.NewCoAPMessage(m.CON, m.POST)
	message.Payload = m.NewBytesPayload(expectedPayload)
	message.SetURIPath(pathTestBlock1)
	return message
}

func newMessageForTestBlock2() *m.CoAPMessage {
	message := m.NewCoAPMessage(m.CON, m.GET)
	message.SetURIPath(pathTestBlock2)
	return message
}

func newMessageForTestMix(expectedPayload []byte) *m.CoAPMessage {
	message := m.NewCoAPMessage(m.CON, m.POST)
	message.Payload = m.NewBytesPayload(expectedPayload)
	message.SetURIPath(pathTestBlockMix)
	return message
}
