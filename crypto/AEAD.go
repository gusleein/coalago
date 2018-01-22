package crypto

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"

	"github.com/lucas-clemente/aes12"
)

type AEAD struct {
	PeerKey   []byte
	MyKey     []byte
	PeerIV    []byte
	MyIV      []byte
	encrypter cipher.AEAD
	decrypter cipher.AEAD
}

func NewAEAD(peerKey, myKey, peerIV, myIV []byte) (*AEAD, error) {
	if len(myKey) != 16 || len(peerKey) != 16 || len(myIV) != 4 || len(peerIV) != 4 {
		return nil, errors.New("AES-GCM: expected 16-byte keys and 4-byte IVs")
	}

	encrypterCipher, err := aes12.NewCipher(myKey)
	if err != nil {
		return nil, err
	}
	encrypter, err := aes12.NewGCM(encrypterCipher)
	if err != nil {
		return nil, err
	}
	decrypterCipher, err := aes12.NewCipher(peerKey)
	if err != nil {
		return nil, err
	}
	decrypter, err := aes12.NewGCM(decrypterCipher)
	if err != nil {
		return nil, err
	}

	return &AEAD{
		PeerKey:   peerKey,
		MyKey:     myKey,
		PeerIV:    peerIV,
		MyIV:      myIV,
		encrypter: encrypter,
		decrypter: decrypter,
	}, nil
}

func (aead *AEAD) Open(cipherText []byte, counter uint16, associatedData []byte) ([]byte, error) {
	plainText, err := aead.decrypter.Open(nil, makeNonce(aead.PeerIV, counter), cipherText, associatedData)
	return plainText, err
}

func (aead *AEAD) Seal(plainText []byte, counter uint16, associatedData []byte) []byte {
	cipherText := aead.encrypter.Seal(nil, makeNonce(aead.MyIV, counter), plainText, associatedData)
	return cipherText
}

func makeNonce(iv []byte, counter uint16) []byte {
	res := make([]byte, 12)
	copy(res[0:4], iv)
	binary.LittleEndian.PutUint16(res[4:12], counter)
	return res
}
