package Coala

import (
	"net"
	"testing"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
)

func TestSecuritySimple(t *testing.T) {
	expectedPayload := "hello world"
	expectedResponse := "abc"

	coalaSender := NewCoala()
	coalaSender.StaticPrivateKeyEnable([]byte("Hello world"))

	defer coalaSender.connection.Close()

	coalaReceiver := NewCoala()
	coalaReceiver.Listen(11111)
	defer coalaReceiver.connection.Close()

	coalaReceiver.AddPOSTResource("/test", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		return resource.NewResponse(m.NewStringPayload(expectedResponse), m.CoapCodeContent)
	})

	message := m.NewCoAPMessage(m.CON, m.POST)
	message.SetSchemeCOAPS()
	message.SetStringPayload(expectedPayload)
	message.SetURIPath("/test")
	message.Token = []byte("OM")

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:11111")

	respMsg, err := coalaSender.Send(message, addr)
	if err != nil {
		t.Error(err)
	}

	if respMsg.Payload.String() != expectedResponse {
		t.Errorf("Expected result: %s \nActual:%v", expectedPayload, respMsg.Payload.String())
	}
}
