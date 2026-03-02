package hosts

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"runtime"

	"github.com/Ruohao1/penta/internal/core/types"
	"github.com/vishvananda/netlink"
)

type arpProber struct{}

func (p *arpProber) Name() string { return "arp" }

func (p *arpProber) Probe(ctx context.Context, target types.Target, opts types.RunOptions) (types.Finding, error) {
	addr := target.IP
	if !addr.IsValid() {
		return types.Finding{
			Check:    "arp_probe",
			Proto:    types.ProtocolARP,
			Severity: "error",
			Status:   "invalid_target",
			Meta:     map[string]any{"raw": target.Raw},
		}, nil
	}

	finding := types.Finding{
		Check:    "arp_probe",
		Proto:    types.ProtocolARP,
		Severity: "info",
		Status:   "unknown",
		Endpoint: types.NewEndpointNet(addr.String(), 0),
		Meta:     map[string]any{"addr": addr.String()},
	}

	if !canUseARP(addr) {
		finding.Status = "unsupported"
		finding.Meta["reason"] = "arp_unsupported"
		return finding, nil
	}

	neigh, err := lookupARP(addr)
	if err != nil {
		finding.Status = "down"
		finding.Meta["err"] = err.Error()
		return finding, nil
	}

	switch neigh.State {
	case netlink.NUD_REACHABLE, netlink.NUD_STALE, netlink.NUD_DELAY, netlink.NUD_PROBE:
		finding.Status = "up"
		finding.Meta["mac"] = neigh.HardwareAddr.String()
		finding.Meta["reason"] = "arp_reachable"
		return finding, nil

	case netlink.NUD_INCOMPLETE, netlink.NUD_FAILED:
		finding.Status = "down"
		finding.Meta["reason"] = "arp_incomplete"
		return finding, nil
	}
	return finding, nil
}

func lookupARP(ip netip.Addr) (netlink.Neigh, error) {
	linkIndex, err := lookupLinkIndex(ip)
	if err != nil {
		return netlink.Neigh{}, err
	}

	neighbors, err := netlink.NeighList(linkIndex, netlink.FAMILY_V4)
	if err != nil {
		return netlink.Neigh{}, err
	}
	for _, n := range neighbors {
		if n.IP.Equal(net.ParseIP(ip.String())) {
			return n, nil
		}
	}
	return netlink.Neigh{}, nil
}

func lookupLinkIndex(ip netip.Addr) (int, error) {
	linkList, err := netlink.LinkList()
	if err != nil {
		return -1, err
	}
	for _, link := range linkList {
		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return -1, err
		}
		for _, a := range addrs {
			ipNet := a.IPNet
			if ipNet.IP.To4() != nil && ipNet.Contains(ip.AsSlice()) {
				return link.Attrs().Index, nil
			}
		}
	}
	return -1, fmt.Errorf("link not found for ip %s", ip.String())
}

func canUseARP(addr netip.Addr) bool {
	if !addr.Is4() || runtime.GOOS != "linux" {
		return false
	}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.To4() != nil && ipNet.Contains(addr.AsSlice()) {
				return true // same L2 subnet
			}
		}
	}
	return false
}
