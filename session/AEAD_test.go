package session

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"testing"
)

func TestSealOpen(t *testing.T) {
	var (
		alice, bob                       *AEAD
		keyAlice, keyBob, ivAlice, ivBob []byte
		err                              error
	)

	// Prepare

	keyAlice = make([]byte, 16)
	keyBob = make([]byte, 16)
	ivAlice = make([]byte, 4)
	ivBob = make([]byte, 4)
	rand.Reader.Read(keyAlice)
	rand.Reader.Read(keyBob)
	rand.Reader.Read(ivAlice)
	rand.Reader.Read(ivBob)

	// Create

	alice, err = NewAEAD(keyBob, keyAlice, ivBob, ivAlice)
	if err != nil {
		t.Fail()
	}

	bob, err = NewAEAD(keyAlice, keyBob, ivAlice, ivBob)
	if err != nil {
		t.Fail()
	}

	// Test Forward

	b := alice.Seal([]byte("foobar"), 42, []byte("aad"))
	text, err := bob.Open(b, 42, []byte("aad"))
	if err != nil {
		t.Fail()
	}

	fmt.Println(text, string(text[:]))

	if !bytes.Equal(text, []byte("foobar")) {
		t.Fail()
	}

	// Test Reverse

	b = bob.Seal([]byte("foobar"), 42, []byte("aad"))
	text, err = alice.Open(b, 42, []byte("aad"))
	if err != nil {
		t.Fail()
	}

	fmt.Println(text, string(text[:]))

	if !bytes.Equal(text, []byte("foobar")) {
		t.Fail()
	}
}
