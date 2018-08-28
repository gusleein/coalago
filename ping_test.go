package coalago

import (
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	srv := NewServer()
	go func() {
		err := srv.Listen(":12312")
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Second)
	ok, err := Ping("127.0.0.1:12312")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ping is false")
	}
}
