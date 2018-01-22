package coalago

import (
	"fmt"
	"net"
	"strings"
	"testing"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/resource"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
)

func TestProxy(t *testing.T) {
	bigtxt := m.GenerateToken(byteBuffer.FRAME_SIZE * 2 * 1024)
	expectedPayload := strings.Repeat("a", byteBuffer.FRAME_SIZE*111)
	expectedString := "%100 %200 & - is amersant"

	coalaSender := NewCoala()
	defer coalaSender.connection.Close()

	coalaReceiver := NewCoala()
	coalaReceiver.Listen(44002)
	defer coalaReceiver.connection.Close()

	coalaProxy := NewCoala()
	coalaProxy.EnableProxy()
	coalaProxy.Listen(50001)
	defer coalaProxy.connection.Close()

	coalaReceiver.AddPOSTResource("/proxytest", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
		text := message.GetURIQuery("text")
		if text != expectedString {
			t.Error("Invalid transdered parameters:", text)
		}
		return resource.NewResponse(m.NewBytesPayload(bigtxt), m.CoapCodeContent)
	})

	message := m.NewCoAPMessage(m.CON, m.POST)

	message.SetStringPayload(expectedPayload)
	message.SetSchemeCOAPS()
	message.SetURIPath("/info")

	prx := fmt.Sprintf("coaps://127.0.0.1:44002/proxytest?text=%s", m.EscapeString(expectedString))

	message.SetProxyURI(prx)
	message.Token = []byte("OM")

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:50001")

	respMsg, err := coalaSender.Send(message, addr)
	if err != nil {
		t.Error(err)
	}

	if respMsg.Payload.String() != string(bigtxt) {
		t.Errorf("Expected result: %s \nActual:%v", bigtxt, respMsg.Payload.String())
	}
}
