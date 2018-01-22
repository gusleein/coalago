package SecurityLayer

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/coalalib/coalago/crypto"
	m "github.com/coalalib/coalago/message"
)

func TestEncryptionOptions(t *testing.T) {
	myCurve, err := crypto.NewCurve25519()
	if err != nil {
		t.Fail()
	}

	peerCurve, err := crypto.NewCurve25519()
	if err != nil {
		t.Fail()
	}

	mySecret, err := myCurve.GenerateSharedSecret(peerCurve.GetPublicKey())
	peerSecret, err := peerCurve.GenerateSharedSecret(myCurve.GetPublicKey())

	if !bytes.Equal(mySecret, peerSecret) {
		fmt.Println("Secrets are not Equal!")
		t.Fail()
	}

	peerKey, myKey, peerIV, myIV, err := crypto.DeriveKeysFromSharedSecret(mySecret, nil, nil)
	if err != nil {
		t.Fail()
	}

	myAEAD, err := crypto.NewAEAD(peerKey, myKey, peerIV, myIV)
	if err != nil {
		fmt.Println("Cannot create myAEAD")
		t.Fail()
	}

	message := m.NewCoAPMessage(m.CON, m.GET)
	message.SetSchemeCOAPS()
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:5683")

	err = EncryptionOptions(message, addr, myAEAD)
	if err != nil {
		t.Error(err)
	}
}
