package coalago

import (
	"net"
	"time"

	m "github.com/coalalib/coalago/message"
)

var globalPoolConnections = newConnpool()

type dialer interface {
	Close() error
	Listen([]byte) (int, net.Addr, error)
	Read(buff []byte) (int, error)
	Write(buf []byte) (int, error)
	WriteTo(buf []byte, addr string) (int, error)
	RemoteAddr() net.Addr
	SetReadDeadline()
}

type connection struct {
	end  chan struct{}
	conn *net.UDPConn
}

func (c *connection) Close() error {
	err := c.conn.Close()
	<-c.end
	return err
}

func (c *connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connection) Read(buff []byte) (int, error) {
	return c.conn.Read(buff)
}

func (c *connection) Listen(buff []byte) (int, net.Addr, error) {
	return c.conn.ReadFromUDP(buff)
}

func (c *connection) Write(buf []byte) (int, error) {
	return c.conn.Write(buf)
}

func (c *connection) WriteTo(buf []byte, addr string) (int, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return 0, err
	}
	return c.conn.WriteTo(buf, a)
}

func newDialer(end chan struct{}, addr string) (dialer, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	end <- struct{}{}
	conn, err := net.DialUDP("udp4", nil, a)
	if err != nil {
		return nil, err
	}

	c := new(connection)
	c.conn = conn
	c.end = end
	return c, nil
}

func newListener(addr string) (dialer, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", a)
	if err != nil {
		return nil, err
	}
	c := new(connection)
	c.conn = conn
	return c, nil
}

type connpool struct {
	balance chan struct{}
}

func newConnpool() *connpool {
	c := new(connpool)
	c.balance = make(chan struct{}, 100)
	return c
}

func (c *connpool) Dial(addr string) (dialer, error) {
	return newDialer(c.balance, addr)
}

func (c *connection) SetReadDeadline() {
	c.conn.SetReadDeadline(time.Now().Add(time.Second * 3))
}

type packet struct {
	acked    bool
	attempts int
	lastSend time.Time
	message  *m.CoAPMessage
	response *m.CoAPMessage
}

func receiveMessage(tr *transport) (*m.CoAPMessage, error) {
	tr.conn.SetReadDeadline()

	for {
		buff := make([]byte, 1500)
		n, err := tr.conn.Read(buff)
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				return nil, ErrMaxAttempts
			}
			return nil, err
		}

		message, err := preparationReceivingMessage(tr, buff[:n], tr.conn.RemoteAddr())
		if err != nil {
			continue
		}

		return message, nil
	}

}
