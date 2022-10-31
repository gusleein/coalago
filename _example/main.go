package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/coalalib/coalago"
)

var (
	portForTest      int = 1111
	pathTestBlock1       = "/testblock1"
	pathTestBlock2       = "/testblock2"
	pathTestBlockMix     = "/testblockmix"

	MAX_PAYLOAD_SIZE = 1024
)

func main() {
	mode := os.Args[1]

	var stringPayload string
	stringPayload += strings.Repeat("a", 100*MAX_PAYLOAD_SIZE)

	expectedPayload := []byte(stringPayload)
	expectedResponse := []byte("Hello from Coala!:)")

	switch mode {
	case "server":
		s := coalago.NewServer()
		s.AddPOSTResource(pathTestBlock1, func(message *CoAPMessage) *CoAPResourceHandlerResult {
			if !bytes.Equal(message.Payload.Bytes(), expectedPayload) {
				panic(fmt.Sprintf("Expected payload: %s\n\nActual payload: %s\n", expectedPayload, message.Payload.Bytes()))
			}

			resp := NewBytesPayload(expectedResponse)
			// time.Sleep(time.Second * 15)
			return NewResponse(resp, CoapCodeChanged)
		})
		log.Panic(s.Listen(":1111"))
	case "client":
		c := coalago.NewClient()
		var wg sync.WaitGroup
		var count int32
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				resp, err := c.POST(expectedPayload, fmt.Sprintf("coap://127.0.0.1:%d%s", portForTest, pathTestBlock1))
				if err != nil {
					println(err)
					atomic.AddInt32(&count, 1)
					return
				}

				if !bytes.Equal(resp.Body, expectedResponse) {
					panic(fmt.Sprintf("Expected response: %s\n\nActual response: %s\n", expectedResponse, resp.Body))
				}
			}()
		}
		wg.Wait()
	}
}
