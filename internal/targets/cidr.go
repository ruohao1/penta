package targets

import (
	"fmt"
	"net"
)

var _ Target = (*CIDR)(nil)

type CIDR struct {
	value   string
	network *net.IPNet
}

func (c *CIDR) Type() Type {
	return TypeCIDR
}

func (c *CIDR) String() string {
	return c.value
}

func parseCIDR(s string) (*CIDR, error) {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cidr: %s", s)
	}
	return &CIDR{value: network.String(), network: network}, nil
}

func (c *CIDR) Contains(ip *IP) bool {
	if c == nil || ip == nil || c.network == nil || ip.ip == nil {
		return false
	}
	return c.network.Contains(ip.ip)
}
