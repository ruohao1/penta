package tasks

import (
	"context"
	"strconv"

	"github.com/Ruohao1/penta/internal/core/types"
)

type Type string

const (
	HostDiscovery Type = "host_discovery"
	PortScan      Type = "port_scan"
	WebProbe      Type = "web_content_discovery"
)

// Task is user intent: WHAT to do + WHAT to touch.
type Task struct {
	Type Type `json:"type"`

	Targets []types.Target `json:"targets"` // parsed expressions: CIDR, range, IP, hostname
	Ports   []int          `json:"ports,omitempty"`

	Wait func(ctx context.Context) error // optional external wait hook, can be nil
}

func NewHostDiscovery(targetsExpr string, portsExpr []string) (Task, error) {
	targets, err := types.NewTargets(targetsExpr)
	if err != nil {
		return Task{}, err
	}
	ports, err := ParsePorts(portsExpr)
	if err != nil {
		return Task{}, err
	}
	return Task{
		Type:    HostDiscovery,
		Targets: targets,
		Ports:   ports,
	}, nil
}

func NewPortScan(targetsExpr string, portsExpr []string) (Task, error) {
	targets, err := types.NewTargets(targetsExpr)
	if err != nil {
		return Task{}, err
	}
	ports, err := ParsePorts(portsExpr)
	if err != nil {
		return Task{}, err
	}
	return Task{
		Type:    PortScan,
		Targets: targets,
		Ports:   ports,
	}, nil
}

func ParsePorts(ports []string) ([]int, error) {
	portsInt := make([]int, len(ports))
	for i, port := range ports {
		p, err := strconv.Atoi(port)
		if err != nil {
			return []int{}, err
		}
		portsInt[i] = p
	}
	return portsInt, nil
}

type DiscoveryMethod string

const (
	DiscoveryTCP  DiscoveryMethod = "tcp"  // TCP connect to probe port(s)
	DiscoveryICMP DiscoveryMethod = "icmp" // ping
	DiscoveryARP  DiscoveryMethod = "arp"  // LAN-only
)

func (t Task) ExpandAllTargetsExpr() ([]string, error) {
	out := []string{}
	for _, target := range t.Targets {
		expand, err := target.ExpandAll()
		if err != nil {
			return nil, err
		}
		out = append(out, expand...)
	}
	return out, nil
}
