# Coala



Coala is a Go-Lang library for secure peer-to-peer communication based on Constrained Application Protocol (CoAP, see [RFC#7252](https://tools.ietf.org/html/rfc7252)).




# Installation
```
go get -u github.com/coalalib/coalago
```



# Usage



## Basic

```go
coala := Coala.NewCoala()
```



## Simple server & Simple client

```go
package main

import (
	"fmt"
	"net"

	"github.com/coalalib/coalago"
	"github.com/coalalib/coalago/resource"

	m "github.com/coalalib/coalago/message"
)

func main() {
	server()
	client()
}

func server() {
	coalaServer := coalago.NewListen(5683)

	coalaServer.AddGETResource("/parrot", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		word := message.GetURIQuery("word")
		handlerResult := resource.NewResponse(m.NewStringPayload(word), m.CoapCodeContent)
		return handlerResult
	})

}

func client() {
	coalaClient := coalago.NewCoala()
	requestMessage := m.NewCoAPMessage(m.CON, m.GET)
	requestMessage.SetURIPath("/parrot")
	requestMessage.SetURIQuery("word", "hello world!")

	address, err := net.ResolveUDPAddr("udp", "127.0.0.1:5683")
	if err != nil {
		fmt.Println(err)
		return
	}

	responseMessage, err := coalaClient.Send(requestMessage, address)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("RESPONSE: ", responseMessage.Payload.String())
}
```

## Encrypted Messaging

Coala is designed to be strongly secured, simple and lightweight and the same time.

Switching between standard and secured connections is easy as is, just specify "coaps" scheme in your request:

```go
request := coalago.NewCoAPMessage(coalago.CON, coalago.GET)
requestMessage.SetSchemeCOAPS()
request.SetStringPayload("Put your innermost secrets here... And nobody will be able to read it...")
```




# Dependencies

1. github.com/lucas-clemente/aes12
2. golang.org/x/crypto/curve25519
3. golang.org/x/crypto/hkdf
4. github.com/op/go-logging
5. golang.org/x/net/ipv4




# Poetry

Однажды был случай в далёком Макао -

Макака коалу в какао макала.

Коала какао лениво лакала,

Макака макала, коала икала.
