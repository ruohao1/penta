package targets

import (
	"fmt"
	"strings"
)

type Type string

const (
	TypeDomain  Type = "domain"
	TypeIP      Type = "ip"
	TypeCIDR    Type = "cidr"
	TypeService Type = "service"
	TypeIPRange Type = "ip_range"
	TypeURL     Type = "url"
)

type TargetRef struct {
	Value string `json:"value"`
	Type  Type   `json:"type"`
}

type Target interface {
	Type() Type
	String() string
}

func Parse(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("target is required")
	}

	if target, err := parseURL(raw); err == nil {
		return target, nil
	}
	if target, err := parseCIDR(raw); err == nil {
		return target, nil
	}
	if target, err := parseIP(raw); err == nil {
		return target, nil
	}
	if target, err := parseService(raw); err == nil {
		return target, nil
	}
	if target, err := parseIPRange(raw); err == nil {
		return target, nil
	}
	if target, err := parseDomain(raw); err == nil {
		return target, nil
	}

	return nil, fmt.Errorf("invalid target: %s", raw)
}
