package Coala

import (
	"fmt"
	"net"
	"strings"
	"testing"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
)

func TestARQBlock1(t *testing.T) {
	expectedPayload := string(m.GenerateToken(byteBuffer.FRAME_SIZE * 2 * 1024))
	expectedResponse := "abc"

	coalaSender := NewCoala()
	defer coalaSender.connection.Close()

	coalaReceiver := NewCoala()
	coalaReceiver.Listen(11111)
	defer coalaReceiver.connection.Close()

	coalaReceiver.AddPOSTResource("/arqtest", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		a := message.GetURIQuery("a")
		if a != "hello" {
			t.Error(a)
		}
		return resource.NewResponse(m.NewStringPayload(expectedResponse), m.CoapCodeContent)
	})

	message := m.NewCoAPMessage(m.CON, m.POST)

	message.SetStringPayload(expectedPayload)
	message.SetURIPath("/arqtest")
	message.SetSchemeCOAPS()
	message.Token = []byte("OM")
	message.SetURIQuery("a", "hello")

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:11111")

	respMsg, err := coalaSender.Send(message, addr)
	if err != nil {
		t.Error(err)
	}

	if respMsg.Payload.String() != expectedResponse {
		t.Errorf("Expected result: %s \nActual:%v", expectedPayload, respMsg.Payload.String())
	}
}

func TestARQBlock2(t *testing.T) {
	expectedResponse := string(m.GenerateToken(byteBuffer.FRAME_SIZE * 2 * 1024))

	coalaSender := NewCoala()
	defer coalaSender.connection.Close()

	coalaReceiver := NewCoala()
	coalaReceiver.Listen(11111)
	defer coalaReceiver.connection.Close()

	coalaReceiver.AddGETResource("/arqtest", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		return resource.NewResponse(m.NewStringPayload(expectedResponse), m.CoapCodeContent)
	})

	message := m.NewCoAPMessage(m.CON, m.GET)
	message.SetSchemeCOAPS()
	message.SetURIPath("/arqtest")
	message.Token = []byte("OM")

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:11111")

	respMsg, err := coalaSender.Send(message, addr)
	if err != nil {
		t.Error(err)
	}

	if respMsg.Payload.String() != expectedResponse {
		t.Errorf("Expected result: %s \nActual:%v \nLen1: %v, %v", expectedResponse, respMsg.Payload.String(), len(expectedResponse), len(respMsg.Payload.String()))
	}
}

func TestARQBlockMix(t *testing.T) {
	expectedPayload := strings.Repeat("a", byteBuffer.FRAME_SIZE*2*1024)

	coalaSender := NewCoala()
	defer coalaSender.connection.Close()

	coalaReceiver := NewCoala()
	coalaReceiver.Listen(11111)
	defer coalaReceiver.connection.Close()

	coalaReceiver.AddPOSTResource("/arqtest", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		fmt.Println("RESULT OF HANDLER: ", message.Payload.String())
		if message.Payload.String() != expectedPayload {
			t.Errorf("Expected result: %s \nActual:%v", expectedPayload, message.Payload.String())
		}
		return resource.NewResponse(m.NewStringPayload(message.Payload.String()), m.CoapCodeContent)
	})

	message := m.NewCoAPMessage(m.CON, m.POST)

	message.SetStringPayload(expectedPayload)
	message.SetURIPath("/arqtest")
	message.Token = []byte("OM")
	message.SetSchemeCOAPS()

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:11111")

	respMsg, err := coalaSender.Send(message, addr)
	if err != nil {
		t.Error(err)
	}

	if respMsg.Payload.String() != expectedPayload {
		t.Errorf("Expected result: %s \nActual:%v", expectedPayload, respMsg.Payload.String())
	}
}

func TestARQBlock1NotFound(t *testing.T) {
	expectedPayload := string(m.GenerateToken(byteBuffer.FRAME_SIZE * 2 * 1024))

	coalaSender := NewCoala()
	defer coalaSender.connection.Close()

	coalaReceiver := NewCoala()
	coalaReceiver.Listen(11111)
	defer coalaReceiver.connection.Close()

	message := m.NewCoAPMessage(m.CON, m.POST)

	message.SetStringPayload(expectedPayload)
	message.SetURIPath("/arqtest")
	message.Token = []byte("OM")

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:11111")

	respMsg, err := coalaSender.Send(message, addr)
	if err != nil {
		t.Error(err)
	}

	if respMsg.Code != m.CoapCodeNotFound {
		t.Errorf("Expected result: %v \nActual:%v", m.CoapCodeNotFound, respMsg.Code)
	}

	if respMsg.GetBlock1() == nil {
		t.Errorf("Block1 option not found.")
	}
}
