package targets

import (
	"fmt"
	"net"
)

var _ Target = (*IP)(nil)

type IP struct {
	addr string
	ip   net.IP
}

func (i *IP) Type() Type {
	return TypeIP
}

func (i *IP) String() string {
	return i.addr
}

func parseIP(s string) (*IP, error) {
	parsed := net.ParseIP(s)
	if parsed == nil {
		return nil, fmt.Errorf("invalid ip: %s", s)
	}
	return &IP{addr: parsed.String(), ip: parsed}, nil
}
