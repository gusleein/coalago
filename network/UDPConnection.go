package network

import (
	"net"
	"runtime"
	"strconv"
	"time"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("network")

const MaxPacketSize = 1500

type UDPConnection interface {
	IsClosed() bool
	Close() error
	WriteTo(b []byte, addr net.Addr) (int, error)
	Read(buffer []byte) (n int, fromAddr net.Addr, err error)
	SetReadDeadline(time.Time) error
}

type udpConnection struct {
	conn     *net.UDPConn
	isClosed bool
}

func NewUDPConnection(port int) (UDPConnection, error) {
	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	p, err := net.ListenUDP("udp4", addr)

	if err != nil {
		return nil, err
	}

	return &udpConnection{
		conn:     p,
		isClosed: false,
	}, nil
}

func (c *udpConnection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *udpConnection) Read(buffer []byte) (int, net.Addr, error) {
	n, peer, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return 0, nil, err
	}
	return n, peer, err
}

func (c *udpConnection) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	addr, err = net.ResolveUDPAddr(addr.Network(), addr.String())
	if err != nil {
		return
	}
	return c.conn.WriteTo(b, addr)
}

func (c *udpConnection) Close() error {
	c.isClosed = true
	err := c.conn.Close()
	return err
}

func (c *udpConnection) IsClosed() bool {
	return c.isClosed
}

func getInterfaceForMulticastIPGroup(interfaces []net.Interface) *net.Interface {
	for _, i := range interfaces {
		if runtime.GOOS == "darwin" { /* MAC OS X */
			// the interface name should start with `enX`
			if i.Name[0:2] == "en" {
				if addrs, err := i.Addrs(); err == nil && len(addrs) > 0 {
					return &i
				}
			}
		} else if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" { /* Ubuntu, Debian, etc. */
			if i.Name == "wlan0" {
				return &i
			}
		} else if runtime.GOOS == "linux" && runtime.GOARCH == "arm" { /* IoT (GOARM=6) */
			if i.Name == "eth0" {
				return &i
			}
		}
	}

	return nil
}
