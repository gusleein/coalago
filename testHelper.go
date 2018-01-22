package Coala

import (
	"net"
	"time"
)

type mockUDPConnector struct {
	ResponseClose   error
	ResponseWriteTo struct {
		Buffer  []byte
		Address net.Addr
	}
	ResponseRead struct {
		Result   []byte
		N        int
		FromAddr *net.UDPAddr
		Error    error
	}
	ResponseIsClosed bool
}

func (m *mockUDPConnector) WriteTo(b []byte, addr net.Addr) (int, error) {
	return 0, nil
}

func (m *mockUDPConnector) Close() error {
	return m.ResponseClose
}

func (m *mockUDPConnector) Read() (result []byte, n int, fromAddr *net.UDPAddr, err error) {
	return m.ResponseRead.Result, m.ResponseRead.N, m.ResponseRead.FromAddr, m.ResponseRead.Error
}

func (m *mockUDPConnector) IsClosed() bool {
	return m.ResponseIsClosed
}

func (m *mockUDPConnector) SetReadDeadline(t time.Time) error {
	return nil
}

func newCoalaMocked() (coala *Coala) {
	coala = NewCoala()
	coala.connection = new(mockUDPConnector)
	return coala
}
