package coalago

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
)

func init() {
	go func() {
		log.Panic(http.ListenAndServe("localhost:6060", nil))
	}()
}

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

	c := NewClient()
	s := NewServer()
	s.AddPOSTResource(pathTestBlock1, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		if !bytes.Equal(message.Payload.Bytes(), expectedPayload) {
			panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedPayload, message.Payload.Bytes()))
		}

		resp := m.NewBytesPayload(expectedResponse)
		return resource.NewResponse(resp, m.CoapCodeChanged)
	})

	go s.Listen(":1111")
	time.Sleep(time.Second)
	var wg sync.WaitGroup

	for i := 0; i < 1; i++ {
		wg.Add(1)
		func() {
			resp, err := c.POST(expectedPayload, fmt.Sprintf("coaps://127.0.0.1:%d%s", portForTest, pathTestBlock1))
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(resp.Body, expectedResponse) {
				panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Body))
			}
			wg.Done()
		}()
	}
	wg.Wait()

}

func TestBlock1(t *testing.T) {
	var stringPayload string
	stringPayload += strings.Repeat("o9иabcy", 100*MAX_PAYLOAD_SIZE)

	expectedPayload := []byte(stringPayload)
	expectedResponse := []byte("Hello from Coala!:)")

	c := NewClient()
	s := NewServer()
	s.AddPOSTResource(pathTestBlock1, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		if !bytes.Equal(message.Payload.Bytes(), expectedPayload) {
			println(fmt.Sprintf("Expected len: %d\n\nActual len: %d\n", len(expectedPayload), message.Payload.Length()))
			// panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedPayload, message.Payload.Bytes()))
			panic(fmt.Sprintf("Actual response: %s\n", message.Payload.Bytes()))

		}

		resp := m.NewBytesPayload(expectedResponse)

		return resource.NewResponse(resp, m.CoapCodeChanged)
	})

	go s.Listen(":1111")
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			resp, err := c.POST(expectedPayload, fmt.Sprintf("coap://127.0.0.1:%d%s", portForTest, pathTestBlock1))
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(resp.Body, expectedResponse) {
				panic(fmt.Sprintf("Expected len: %d\n\nActual len: %d\n", len(expectedResponse), len(resp.Body)))

				// panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Body))
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBlock2(t *testing.T) {
	var stringPayload string
	stringPayload += strings.Repeat("o9иabcy", 3*MAX_PAYLOAD_SIZE)
	expectedResponse := []byte(stringPayload)
	c := NewClient()
	s := NewServer()

	s.AddGETResource(pathTestBlock2, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		resp := m.NewBytesPayload(expectedResponse)
		return resource.NewResponse(resp, m.CoapCodeContent)
	})

	go s.Listen(":1111")
	time.Sleep(time.Millisecond)

	fmt.Println("ЗАПУСКАем!")
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		func() {
			defer wg.Done()

			resp, err := c.GET("coap://127.0.0.1:1111" + pathTestBlock2)
			if err != nil {
				panic(err)
				// return
			}

			if !bytes.Equal(resp.Body, expectedResponse) {
				panic(fmt.Sprintf("Actual response: %s\n", resp.Body))

				// panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Body))
			}

		}()
	}

	wg.Wait()
}

func TestBlockMix(t *testing.T) {
	var stringPayload string
	stringPayload += strings.Repeat("o9иabcy", 10*MAX_PAYLOAD_SIZE)
	expectedResponse := []byte(stringPayload)
	c := NewClient()
	s := NewServer()

	s.AddPOSTResource(pathTestBlockMix, func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		if !bytes.Equal(message.Payload.Bytes(), expectedResponse) {
			println(fmt.Sprintf("Expected len: %d\n\nActual len: %d\n", len(expectedResponse), message.Payload.Length()))
			panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedResponse, message.Payload.Bytes()))
		}

		resp := m.NewBytesPayload(expectedResponse)
		// time.Sleep(time.Second * 15)

		return resource.NewResponse(resp, m.CoapCodeChanged)
	})

	go s.Listen(":1111")
	time.Sleep(time.Millisecond)

	fmt.Println("ЗАПУСКАем!")
	var wg sync.WaitGroup
	// var count int

	for i := 0; i < 1; i++ {
		wg.Add(1)
		func() {
			resp, err := c.POST(expectedResponse, "coaps://127.0.0.1:1111"+pathTestBlockMix)
			if err != nil {
				panic(err)
			}

			if !bytes.Equal(resp.Body, expectedResponse) {
				panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Body))
			}
			wg.Done()
		}()
	}

	wg.Wait()
}
