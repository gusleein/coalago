package session

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

var keyLen int = 16

func DeriveKeysFromSharedSecret(sharedSecret, salt, info []byte) ([]byte, []byte, []byte, []byte, error) {
	r := hkdf.New(sha256.New, sharedSecret, salt, info)

	s := make([]byte, 2*keyLen+2*4)
	if _, err := io.ReadFull(r, s); err != nil {
		return nil, nil, nil, nil, err
	}

	peerKey := s[:keyLen]
	myKey := s[keyLen : 2*keyLen]
	peerIV := s[2*keyLen : 2*keyLen+4]
	myIV := s[2*keyLen+4:]

	return peerKey, myKey, peerIV, myIV, nil
}
