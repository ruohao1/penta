package types

import (
	"net/netip"
	"strings"
	"time"
)

type HostState string

const (
	HostStateUnknown HostState = "unknown"
	HostStateUp      HostState = "up"
	HostStateDown    HostState = "down"
)

type Host struct {
	Addr      netip.Addr `json:"addr"`
	Hostnames []string   `json:"hostnames,omitempty"`
	State     HostState  `json:"state,omitempty"`

	MAC    string `json:"mac,omitempty"`
	Vendor string `json:"vendor,omitempty"`

	Ports []Port `json:"ports,omitempty"`

	// Network-level context
	NetworkCIDR string `json:"network_cidr,omitempty"` // "192.168.2.0/24"
	Scope       string `json:"scope,omitempty"`        // "in","out","unknown"

	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

func (h Host) Address() string {
	if len(h.Hostnames) > 0 {
		hn := strings.TrimSpace(h.Hostnames[0])
		if hn != "" {
			return hn
		}
	}
	if h.Addr.IsValid() {
		return h.Addr.String()
	}
	return ""
}
