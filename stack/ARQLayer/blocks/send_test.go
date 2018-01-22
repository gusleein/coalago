package blocks

import (
	"testing"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/stack/ARQLayer/byteBuffer"
)

var strTestPayload = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffgggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggghhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiijjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"

type coalaMock struct{}

func (c *coalaMock) Send(m *m.CoAPMessage) (*m.CoAPMessage, error) {
	return nil, nil
}

func (c *coalaMock) SendToSocket(m *m.CoAPMessage) error {
	return nil
}

func TestGetWindow(t *testing.T) {
	var (
		windowMessages []*m.CoAPMessage
		isNext         bool
		outputStr      string
	)

	buffer := byteBuffer.NewBuffer()
	buffer.Write([]byte(strTestPayload))

	message := m.NewCoAPMessage(m.CON, m.POST)
	windowSize := 3

	isFirst := true

	for {
		windowMessages, isNext = getWindow(buffer, windowMessages, message, windowSize, m.OptionBlock1)
		if isFirst {
			for _, msg := range windowMessages {
				outputStr += msg.Payload.String()
			}
			isFirst = false
		} else {
			outputStr += windowMessages[len(windowMessages)-1].Payload.String()
		}

		if !isNext {
			break
		}

	}

	if outputStr != strTestPayload {
		t.Error("Invalid output")
	}
}
