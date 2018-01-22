package crypto

import (
	"testing"
)

func TestNewStaticCurve25519(t *testing.T) {
	private := [KEY_SIZE]byte{}
	copy(private[:], []byte("Hello"))

	session, err := NewStaticCurve25519(private)

	if err != nil {
		t.Error(err)
	}
}
