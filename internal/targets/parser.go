package targets

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

type Kind string

const (
	KindIP   Kind = "ip"
	KindCIDR Kind = "cidr"
	KindURL  Kind = "url"
)

type Target struct {
	Kind Kind
	Raw  string

	IP   netip.Addr
	CIDR netip.Prefix
	URL  *url.URL
}

func (t Target) AssertKind(want Kind) error {
	if t.Kind != want {
		return fmt.Errorf("target kind %q, want %q", t.Kind, want)
	}

	switch want {
	case KindIP:
		if !t.IP.IsValid() {
			return fmt.Errorf("target kind %q has invalid ip value", want)
		}
	case KindCIDR:
		if !t.CIDR.IsValid() {
			return fmt.Errorf("target kind %q has invalid cidr value", want)
		}
	case KindURL:
		if t.URL == nil {
			return fmt.Errorf("target kind %q has nil url value", want)
		}
	default:
		return fmt.Errorf("unsupported target kind %q", want)
	}

	return nil
}

func ParseMany(inputs []string) ([]Target, error) {
	out := make([]Target, 0, len(inputs))
	for _, in := range inputs {
		t, err := ParseOne(in)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func ParseOne(input string) (Target, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Target{}, fmt.Errorf("empty target")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Target{}, fmt.Errorf("target %q: invalid url: %w", raw, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return Target{}, fmt.Errorf("target %q: unsupported url scheme %q", raw, u.Scheme)
		}
		if strings.TrimSpace(u.Host) == "" {
			return Target{}, fmt.Errorf("target %q: url host is required", raw)
		}
		return Target{Kind: KindURL, Raw: raw, URL: u}, nil
	}

	if strings.Contains(raw, "/") {
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return Target{}, fmt.Errorf("target %q: invalid cidr: %w", raw, err)
		}
		return Target{Kind: KindCIDR, Raw: raw, CIDR: prefix.Masked()}, nil
	}

	ip, err := netip.ParseAddr(raw)
	if err == nil {
		return Target{Kind: KindIP, Raw: raw, IP: ip}, nil
	}

	return Target{}, fmt.Errorf("target %q: unsupported target format (use ip, cidr, or http(s) url)", raw)
}
