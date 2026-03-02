package types

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type TargetKind string

const (
	TargetIP          TargetKind = "ip"
	TargetCIDR        TargetKind = "cidr"
	TargetRange       TargetKind = "range"
	TargetIPv4Pattern TargetKind = "ipv4_pattern"
	TargetURL         TargetKind = "url"
	TargetHostname    TargetKind = "hostname"
)

type Target struct {
	Kind TargetKind `json:"kind"`
	Raw  string     `json:"raw"`

	IP    netip.Addr   `json:"ip,omitempty"`
	CIDR  netip.Prefix `json:"cidr,omitempty"`
	Start netip.Addr   `json:"start,omitempty"`
	End   netip.Addr   `json:"end,omitempty"`

	Octets [4]OctetRange `json:"octets,omitempty"`

	URL      string `json:"url,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     uint16 `json:"port,omitempty"` // optional, only if you accept host:port inputs
}

type OctetRange struct {
	Min uint8 `json:"min"`
	Max uint8 `json:"max"`
}

func (t Target) ExpandAll() ([]string, error) {
	return t.ExpandLimit(0) // 0 => no limit
}

// ExpandLimit returns a slice of IPs represented by this Target.
// limit == 0 means "no limit" (can allocate huge memory).
func (t Target) ExpandLimit(limit int) ([]string, error) {
	if limit < 0 {
		return nil, fmt.Errorf("limit must be >= 0")
	}

	var out []string
	emit := func(targetExpr string) bool {
		out = append(out, targetExpr)
		if limit > 0 && len(out) >= limit {
			return false
		}
		return true
	}

	if err := t.Expand(emit); err != nil {
		return nil, err
	}
	return out, nil
}

// Expand streams all IPs represented by this Target into emit.
// It does NOT allocate a huge slice.
//
// Return behavior:
//   - TargetURL => error (not expandable to IPs here)
//   - TargetIP => emits 1 IP
//   - TargetCIDR / TargetRange / TargetIPv4Pattern => emits many IPs
//   - TargetHostname => emits many IPs
//
// If emit returns false, expansion stops early (useful for caps/cancel).
func (t Target) Expand(emit func(string) bool) error {
	switch t.Kind {
	case TargetURL:
		return fmt.Errorf("target kind %q is not expandable to IPs", t.Kind)

	case TargetIP:
		if !t.IP.IsValid() {
			return fmt.Errorf("invalid ip target")
		}
		emit(t.IP.String())
		return nil

	case TargetCIDR:
		if !t.CIDR.IsValid() {
			return fmt.Errorf("invalid cidr target")
		}
		t.CIDR = t.CIDR.Masked()
		// Iterate all addresses in the prefix.
		// Note: includes network/broadcast for IPv4; that's usually what scanners want.
		for ip := t.CIDR.Addr(); ip.IsValid() && t.CIDR.Contains(ip); ip = ip.Next() {
			if !emit(ip.String()) {
				return nil
			}
		}
		return nil

	case TargetRange:
		if !t.Start.IsValid() || !t.End.IsValid() {
			return fmt.Errorf("invalid range target")
		}
		if t.Start.Is4() != t.End.Is4() {
			return fmt.Errorf("mixed IPv4/IPv6 range")
		}
		if compareAddr(t.Start, t.End) > 0 {
			return fmt.Errorf("range start > end")
		}

		for ip := t.Start; ip.IsValid() && compareAddr(ip, t.End) <= 0; ip = ip.Next() {
			if !emit(ip.String()) {
				return nil
			}
		}
		return nil

	case TargetIPv4Pattern:
		// Cartesian product of 4 octet ranges.
		o := t.Octets
		for a := o[0].Min; a <= o[0].Max; a++ {
			for b := o[1].Min; b <= o[1].Max; b++ {
				for c := o[2].Min; c <= o[2].Max; c++ {
					for d := o[3].Min; d <= o[3].Max; d++ {
						ip := netip.AddrFrom4([4]byte{a, b, c, d})
						if !emit(ip.String()) {
							return nil
						}
						if d == 255 { // avoid uint8 overflow in loop condition
							break
						}
					}
					if c == 255 {
						break
					}
				}
				if b == 255 {
					break
				}
			}
			if a == 255 {
				break
			}
		}
		return nil

	case TargetHostname:
		if !emit(t.Hostname) {
			return nil
		}
		return nil

	default:
		return fmt.Errorf("unknown target kind %q", t.Kind)
	}
}

func NewTargets(expr string) ([]Target, error) {
	targets := []Target{}
	for tok := range strings.SplitSeq(expr, ",") {
		t, err := NewTarget(tok)
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// NewTarget parses ONE atomic target token (no commas) and returns the proper Target.
// Supported:
//   - IP:            10.0.0.1
//   - CIDR:          10.0.0.0/24
//   - Range:         10.0.0.1-10.0.0.50
//   - Short range:   10.0.0.0-255
//   - IPv4 pattern:  10.0.0-255.0-255
//   - URL:           http(s)://host[:port][/path]
//   - Hostname:      example.com[:port]
func NewTarget(expr string) (Target, error) {
	token := strings.TrimSpace(expr)
	if token == "" {
		return Target{}, fmt.Errorf("empty target")
	}

	// URL (require scheme:// to avoid ambiguity)
	if strings.Contains(token, "://") {
		u, err := parseURL(token)
		if err != nil {
			return Target{}, err
		}
		return Target{Kind: TargetURL, Raw: token, URL: u}, nil
	}

	// CIDR
	if strings.Contains(token, "/") {
		pfx, err := netip.ParsePrefix(token)
		if err != nil {
			return Target{}, fmt.Errorf("invalid cidr: %w", err)
		}
		return Target{Kind: TargetCIDR, Raw: token, CIDR: pfx}, nil
	}

	// Range OR IPv4 pattern
	if strings.Contains(token, "-") {
		if looksLikeIPv4Pattern(token) {
			octets, err := parseIPv4Pattern(token)
			if err != nil {
				return Target{}, err
			}
			return Target{Kind: TargetIPv4Pattern, Raw: token, Octets: octets}, nil
		}

		start, end, err := parseIPRange(token)
		if err != nil {
			return Target{}, err
		}
		return Target{Kind: TargetRange, Raw: token, Start: start, End: end}, nil
	}

	// IP
	if ip, err := netip.ParseAddr(token); err == nil {
		return Target{Kind: TargetIP, Raw: token, IP: ip}, nil
	}

	// Hostname
	h, p, err := splitHostPortLoose(token)
	if err != nil {
		return Target{}, err
	}
	if err := validateHostname(h); err != nil {
		return Target{}, err
	}

	t := Target{Kind: TargetHostname, Raw: token, Hostname: h}
	if p != 0 {
		t.Port = p
	}
	return t, nil
}

// --- helpers ---

func splitHostPortLoose(s string) (host string, port uint16, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, fmt.Errorf("empty hostname")
	}

	// If it clearly looks like host:port, try to split.
	// net.SplitHostPort requires brackets for IPv6, but hostnames/IPv4 "a:443" works only if you add a scheme.
	// We'll implement a simple last-colon split for non-IPv6.
	if strings.Count(s, ":") == 1 && !strings.Contains(s, "://") {
		h, ps, ok := strings.Cut(s, ":")
		if ok && h != "" && ps != "" {
			n, e := strconv.Atoi(ps)
			if e != nil || n < 1 || n > 65535 {
				return "", 0, fmt.Errorf("invalid port %q", ps)
			}
			return h, uint16(n), nil
		}
	}

	// Otherwise treat as bare hostname.
	return s, 0, nil
}

func validateHostname(h string) error {
	h = strings.TrimSpace(h)
	if h == "" {
		return fmt.Errorf("empty hostname")
	}
	if len(h) > 253 {
		return fmt.Errorf("hostname too long")
	}

	// Allow single-label names (e.g., "router") for internal labs.
	labels := strings.Split(h, ".")
	for _, lab := range labels {
		if lab == "" {
			return fmt.Errorf("invalid hostname %q", h)
		}
		if len(lab) > 63 {
			return fmt.Errorf("hostname label too long in %q", h)
		}
		// RFC-ish: letters/digits/hyphen, not starting/ending with hyphen.
		if lab[0] == '-' || lab[len(lab)-1] == '-' {
			return fmt.Errorf("hostname label starts/ends with hyphen in %q", h)
		}
		for i := 0; i < len(lab); i++ {
			c := lab[i]
			isAZ := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
			is09 := (c >= '0' && c <= '9')
			if !(isAZ || is09 || c == '-') {
				return fmt.Errorf("invalid char %q in hostname %q", c, h)
			}
		}
	}
	return nil
}

func parseURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid url: want scheme://host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported url scheme %q (only http/https)", u.Scheme)
	}

	// Validate optional port
	if h, p, errSplit := net.SplitHostPort(u.Host); errSplit == nil {
		_ = h
		if p != "" {
			port, err := strconv.Atoi(p)
			if err != nil || port < 1 || port > 65535 {
				return "", fmt.Errorf("invalid url port %q", p)
			}
		}
	}

	return u.String(), nil
}

func parseIPRange(token string) (netip.Addr, netip.Addr, error) {
	left, right, ok := strings.Cut(token, "-")
	if !ok {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("invalid range syntax")
	}
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("invalid range syntax")
	}

	start, err := netip.ParseAddr(left)
	if err != nil {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("invalid range start: %w", err)
	}

	// Full end IP?
	if endIP, err2 := netip.ParseAddr(right); err2 == nil {
		if start.Is4() != endIP.Is4() {
			return netip.Addr{}, netip.Addr{}, fmt.Errorf("mixed IPv4/IPv6 range")
		}
		if compareAddr(start, endIP) > 0 {
			return netip.Addr{}, netip.Addr{}, fmt.Errorf("range start > end")
		}
		return start, endIP, nil
	}

	// Short form: start-lastOctet (IPv4 only)
	if !start.Is4() {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("short range form is IPv4-only")
	}
	n, err := strconv.Atoi(right)
	if err != nil || n < 0 || n > 255 {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("invalid short range end %q (want 0..255 or an IP)", right)
	}

	s4 := start.As4()
	e4 := s4
	e4[3] = byte(n)
	end := netip.AddrFrom4(e4)

	if compareAddr(start, end) > 0 {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("range start > end")
	}
	return start, end, nil
}

func looksLikeIPv4Pattern(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 4 {
		return false
	}
	hasDash := false
	for _, p := range parts {
		if strings.Contains(p, "-") {
			hasDash = true
		}
		if p == "" || strings.Count(p, "-") > 1 {
			return false
		}
		for _, r := range p {
			if (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return hasDash
}

func parseIPv4Pattern(token string) ([4]OctetRange, error) {
	var out [4]OctetRange
	parts := strings.Split(token, ".")
	if len(parts) != 4 {
		return out, fmt.Errorf("invalid ipv4_pattern: want 4 octets")
	}
	for i := 0; i < 4; i++ {
		rng, err := parseOctetRange(parts[i])
		if err != nil {
			return out, fmt.Errorf("octet %d: %w", i+1, err)
		}
		out[i] = rng
	}
	return out, nil
}

func parseOctetRange(s string) (OctetRange, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return OctetRange{}, fmt.Errorf("empty octet")
	}
	if !strings.Contains(s, "-") {
		n, err := parseUint8Dec(s)
		if err != nil {
			return OctetRange{}, err
		}
		return OctetRange{Min: n, Max: n}, nil
	}

	left, right, ok := strings.Cut(s, "-")
	if !ok {
		return OctetRange{}, fmt.Errorf("invalid octet range %q", s)
	}
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return OctetRange{}, fmt.Errorf("invalid octet range %q", s)
	}

	a, err := parseUint8Dec(left)
	if err != nil {
		return OctetRange{}, fmt.Errorf("invalid min: %w", err)
	}
	b, err := parseUint8Dec(right)
	if err != nil {
		return OctetRange{}, fmt.Errorf("invalid max: %w", err)
	}
	if a > b {
		return OctetRange{}, fmt.Errorf("min > max in %q", s)
	}
	return OctetRange{Min: a, Max: b}, nil
}

func parseUint8Dec(s string) (uint8, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return 0, fmt.Errorf("want 0..255, got %q", s)
	}
	return uint8(n), nil
}

// compareAddr returns -1/0/1 for a<b / a==b / a>b. Requires same family.
func compareAddr(a, b netip.Addr) int {
	ab := a.As16()
	bb := b.As16()
	for i := 0; i < 16; i++ {
		if ab[i] < bb[i] {
			return -1
		}
		if ab[i] > bb[i] {
			return 1
		}
	}
	return 0
}
