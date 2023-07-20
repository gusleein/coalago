package coalago

import (
	"bytes"
	"net"
	"time"

	cerr "github.com/gusleein/coalago/errors"
	m "github.com/gusleein/coalago/message"
)

var globalPoolConnections = newConnpool()

type dialer interface {
	Close() error
	Listen([]byte) (int, net.Addr, error)
	Read(buff []byte) (int, error)
	Write(buf []byte) (int, error)
	WriteTo(buf []byte, addr string) (int, error)
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	SetReadDeadline()
	SetUDPRecvBuf(size int) int
	SetReadDeadlineSec(timeout time.Duration)
}

type connection struct {
	end  chan struct{}
	conn *net.UDPConn
}

type connpool struct {
	balance chan struct{}
}

type packet struct {
	acked    bool
	attempts int
	lastSend time.Time
	message  *m.CoAPMessage
	response *m.CoAPMessage
}

func (c *connection) SetUDPRecvBuf(size int) int {
	for {
		if err := c.conn.SetReadBuffer(size); err == nil {
			break
		}
		size = size / 2
	}
	return size
}

func (c *connection) Close() error {
	err := c.conn.Close()
	<-c.end
	return err
}

func (c *connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
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

func newDialerV6(end chan struct{}, addr string) (dialer, error) {
	a, err := net.ResolveUDPAddr("udp6", addr)
	if err != nil {
		return nil, err
	}
	end <- struct{}{}
	conn, err := net.DialUDP("udp6", nil, a)
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

func newListenerV6(addr string) (dialer, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp6", a)
	if err != nil {
		return nil, err
	}
	c := new(connection)
	c.conn = conn
	return c, nil
}

func newConnpool() *connpool {
	c := new(connpool)
	c.balance = make(chan struct{}, NumberConnections)
	return c
}

func (c *connpool) Dial(addr string) (dialer, error) {
	return newDialer(c.balance, addr)
}

func (c *connection) SetReadDeadline() {
	c.conn.SetReadDeadline(time.Now().Add(timeWait))
}

func (c *connection) SetReadDeadlineSec(timeout time.Duration) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
}

func receiveMessage(tr *transport, origMessage *m.CoAPMessage) (*m.CoAPMessage, error) {
	for {
		tr.conn.SetReadDeadlineSec(origMessage.Timeout)

		buff := make([]byte, MTU+1)
		n, err := tr.conn.Read(buff)
		origMessage.Timeout = timeWait
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				return nil, cerr.MaxAttempts
			}
			return nil, err
		}
		if n > MTU {
			continue
		}

		message, err := preparationReceivingBuffer(tr, buff[:n], tr.conn.RemoteAddr(), origMessage.ProxyAddr)
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(message.Token, origMessage.Token) {
			continue
		}

		return message, nil
	}
}
