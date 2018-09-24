package coalago

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestDeserializeNonStandartPackage(t *testing.T) {
	m := NewCoAPMessage(CON, GET)

	payload := bytes.Repeat([]byte("a"), 1024)
	m.Payload = NewBytesPayload(payload)

	m.SetURIQuery("q", strings.Repeat("q", 1500))
	b, err := Serialize(m)
	if err != nil {
		panic(err)
	}

	fmt.Println("Lenght of raw packet:", len(b))

	newMessage, err := Deserialize(b[:1500])
	if err != nil {
		panic(err)
	}

	fmt.Println("Payload:", newMessage.Payload.String())
}
