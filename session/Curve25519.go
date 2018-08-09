package session

import (
	"crypto/rand"
	"errors"
	"strconv"

	golangCurve25519 "golang.org/x/crypto/curve25519"
)

const KEY_SIZE int = 32

type Curve25519 struct {
	privateKey [KEY_SIZE]byte
	publicKey  [KEY_SIZE]byte
}

func NewCurve25519() (*Curve25519, error) {
	curve := &Curve25519{}

	if _, err := rand.Read(curve.privateKey[:]); err != nil {
		return nil, errors.New("Curve25519: could not create private key")
	}

	// See https://cr.yp.to/ecdh.html
	curve.privateKey[0] &= 248
	curve.privateKey[31] &= 127
	curve.privateKey[31] |= 64

	golangCurve25519.ScalarBaseMult(&curve.publicKey, &curve.privateKey)

	return curve, nil
}

func NewStaticCurve25519(privateKey [KEY_SIZE]byte) (*Curve25519, error) {
	curve := &Curve25519{}

	curve.privateKey = privateKey

	// See https://cr.yp.to/ecdh.html
	curve.privateKey[0] &= 248
	curve.privateKey[31] &= 127
	curve.privateKey[31] |= 64

	golangCurve25519.ScalarBaseMult(&curve.publicKey, &curve.privateKey)

	return curve, nil
}

func (curve *Curve25519) GetPublicKey() []byte {
	return curve.publicKey[:]
}

func (curve *Curve25519) GenerateSharedSecret(peerPublicKey []byte) ([]byte, error) {
	if len(peerPublicKey) != KEY_SIZE {
		return nil, errors.New("Curve25519: expected public key of " + strconv.Itoa(KEY_SIZE) + " byte")
	}

	var res [KEY_SIZE]byte
	var peerPublicArray [KEY_SIZE]byte
	copy(peerPublicArray[:], peerPublicKey)

	golangCurve25519.ScalarMult(&res, &curve.privateKey, &peerPublicArray)

	return res[:], nil
}
