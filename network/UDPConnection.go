package network

import (
	"net"
	"runtime"
	"strconv"
	"time"

	logging "github.com/op/go-logging"

	"golang.org/x/net/ipv4"
)

var log = logging.MustGetLogger("network")

const MaxPacketSize = 1500

var coapMulticastIPGroup net.IP = net.IPv4(224, 0, 0, 187)

type UDPConnection interface {
	IsClosed() bool
	Close() error
	WriteTo(b []byte, addr net.Addr) (int, error)
	Read() (result []byte, n int, fromAddr *net.UDPAddr, err error)
	SetReadDeadline(time.Time) error
}

type udpConnection struct {
	conn     *ipv4.PacketConn
	isClosed bool
}

func NewUDPConnection(port int) (UDPConnection, error) {
	c, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(port))

	if err != nil {
		return nil, err
	}
	p := ipv4.NewPacketConn(c)

	// Get the list of all available interface of this Platform
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Trying to find the interface to bind to
	// Not all interfaces are suitable for Multicast!
	i := getInterfaceForMulticastIPGroup(interfaces)
	if i != nil {
		if err := p.JoinGroup(i, &net.UDPAddr{IP: coapMulticastIPGroup}); err != nil {
			return nil, err
		}
	} else {
		log.Error("COULD NOT FIND an interface for MulticastIPGroup")
	}

	if err := p.SetControlMessage(ipv4.FlagDst, true); err != nil {
		return nil, err
	}

	return &udpConnection{
		conn:     p,
		isClosed: false,
	}, nil
}

func (c *udpConnection) Write(b []byte) (int, error) {
	return c.conn.WriteTo(b, nil, nil)
}

func (c *udpConnection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *udpConnection) Read() (result []byte, n int, fromAddr *net.UDPAddr, err error) {
	buffer := make([]byte, MaxPacketSize)
	n, cm, peer, err := c.conn.ReadFrom(buffer)

	// check if read failed
	if cm == nil {
		if c.IsClosed() {
			log.Debug("Read interrupted")
		} else {
			log.Error("Read interrupted")
		}
		return
	}

	//if !cm.Dst.IsMulticast() || !cm.Dst.Equal(net.IPv4(224, 0, 1, 187)) {
	if cm.Dst.IsMulticast() {
		log.Debug(cm)
	}

	result = buffer[:n]
	fromAddr = peer.(*net.UDPAddr)

	return
}

func (c *udpConnection) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return c.conn.WriteTo(b, nil, addr)
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
