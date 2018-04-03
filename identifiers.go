package coalago

import (
	"fmt"
	"net"
	"strings"
)

type poolID string

func newPoolID(id uint16, token []byte, addr net.Addr) poolID {
	var b strings.Builder
	fmt.Fprintf(&b, "%d%s%s", id, token, addr)
	return poolID(b.String())
}
