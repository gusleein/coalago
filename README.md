# Coala



Coala is a Go-Lang library for secure peer-to-peer communication based on Constrained Application Protocol (CoAP, see [RFC#7252](https://tools.ietf.org/html/rfc7252)).

COAP diff:

- curve cripto
- arq fast data transmission (30MBit./s)




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
)

func main() {
	server()
	client()
}

func server() {
	coalaServer := coalago.NewListen(5683)

	coalaServer.AddGETResource("/parrot", func(message   *CoAPMessage) *resource.CoAPResourceHandlerResult {
		word := message.GetURIQuery("word")
		handlerResult := resource.NewResponse( NewStringPayload(word), CoapCodeContent)
		return handlerResult
	})
}

func client() {
	coalaClient := coalago.NewCoala()
	requestMessage := NewCoAPMessage( CON, GET)
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




