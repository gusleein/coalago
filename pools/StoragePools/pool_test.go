package StoragePools

import (
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	storagePool = nil
	Start()
	if storagePool == nil {
		t.Error("poolDB == NIL")
	}
}

func TestCleaning(t *testing.T) {
	Start()
	poolName := "testing"
	id := "01"
	msg := "Hello"
	expirationTime := time.Second * 3

	AddPool(poolName, expirationTime, time.Second*1)
	Set(poolName, id, msg)
	msgOut := Get(poolName, id)
	if msgOut == nil {
		t.Error("Error set object to pool, object is nil.")
	}
	time.Sleep(expirationTime)
	msgOut = Get(poolName, id)
	if msgOut != nil {
		t.Error("Error cleaning pool, object is not nil.")
	}
}

func TestDeleting(t *testing.T) {
	Start()
	poolName := "testing"
	id := "01"
	msg := "Hello"
	expirationTime := time.Second * 3

	AddPool(poolName, expirationTime, time.Second*1)
	Set(poolName, id, msg)
	Delete(poolName, id)

	msgOut := Get(poolName, id)
	if msgOut != nil {
		t.Error("Error delete object from pool, object is not nil.")
	}
}
