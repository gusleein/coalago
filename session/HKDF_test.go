package session

import (
	//"crypto/rand"
	"bytes"
	"fmt"

	"testing"
)

func TestKeyDerivation(t *testing.T) {
	myCurve, err := NewCurve25519()
	if err != nil {
		t.Fail()
	}

	peerCurve, err := NewCurve25519()
	if err != nil {
		t.Fail()
	}

	mySecret, err := myCurve.GenerateSharedSecret(peerCurve.GetPublicKey())
	peerSecret, err := peerCurve.GenerateSharedSecret(myCurve.GetPublicKey())

	if !bytes.Equal(mySecret, peerSecret) {
		fmt.Println("Secrets are not Equal!")
		t.Fail()
	}

	peerKey, myKey, peerIV, myIV, err := DeriveKeysFromSharedSecret(mySecret, nil, nil)
	if err != nil {
		t.Fail()
	}

	peerKey2, myKey2, peerIV2, myIV2, err := DeriveKeysFromSharedSecret(peerSecret, nil, nil)
	if err != nil {
		t.Fail()
	}

	fmt.Println(peerKey, myKey, peerIV, myIV)
	fmt.Println(peerKey2, myKey2, peerIV2, myIV2)

	myAEAD, err := NewAEAD(peerKey, myKey, peerIV, myIV)
	if err != nil {
		fmt.Println("Cannot create myAEAD")
		t.Fail()
	}

	peerAEAD, err := NewAEAD(myKey2, peerKey2, myIV2, peerIV2)
	if err != nil {
		fmt.Println("Cannot create peerAEAD")
		t.Fail()
	}

	// Test Forward

	b := myAEAD.Seal([]byte("foobar"), 33000, []byte("aad"))
	text, err := peerAEAD.Open(b, 33000, []byte("aad"))
	if err != nil {
		fmt.Println("Cannot peerAEAD.Open", err)
		t.Fail()
	}

	fmt.Println(text, string(text[:]))

	if !bytes.Equal(text, []byte("foobar")) {
		fmt.Println("Seal & Open are not Equal")
		fmt.Println(string(text[:]))
		t.Fail()
	}
}

func BenchmarkKeyDerivation(b *testing.B) {
	for i := 0; i < b.N; i++ {

		myCurve, err := NewCurve25519()
		if err != nil {
			panic(err)
		}

		peerCurve, err := NewCurve25519()
		if err != nil {
			panic(err)
		}

		mySecret, err := myCurve.GenerateSharedSecret(peerCurve.GetPublicKey())
		peerSecret, err := peerCurve.GenerateSharedSecret(myCurve.GetPublicKey())

		peerKey, myKey, peerIV, myIV, err := DeriveKeysFromSharedSecret(mySecret, nil, nil)
		if err != nil {
			panic(err)
		}

		peerKey2, myKey2, peerIV2, myIV2, err := DeriveKeysFromSharedSecret(peerSecret, nil, nil)
		if err != nil {
			panic(err)
		}

		myAEAD, err := NewAEAD(peerKey, myKey, peerIV, myIV)
		if err != nil {
			fmt.Println("Cannot create myAEAD")
			panic(err)
		}

		peerAEAD, err := NewAEAD(myKey2, peerKey2, myIV2, peerIV2)
		if err != nil {
			fmt.Println("Cannot create peerAEAD")
			panic(err)
		}

		// Test Forward

		b := myAEAD.Seal([]byte("foobar"), 33000, []byte("aad"))
		_, err = peerAEAD.Open(b, 33000, []byte("aad"))
		if err != nil {
			fmt.Println("Cannot peerAEAD.Open", err)
			panic(err)
		}

	}
}
