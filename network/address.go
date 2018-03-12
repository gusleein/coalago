package network

type Address struct {
	addr string
}

func NewAddress(addr string) Address {
	return Address{
		addr: addr,
	}
}

func (a Address) Network() string {
	return "udp"
}

func (a Address) String() string {
	return a.addr
}
