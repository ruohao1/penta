package targets

import (
	"fmt"
	"strconv"
	"strings"
)

var _ Target = (*IPRange)(nil)

type octetRange struct {
	start int
	end   int
}

type IPRange struct {
	value  string
	octets [4]octetRange
}

func (r *IPRange) Type() Type {
	return TypeIPRange
}

func (r *IPRange) String() string {
	return r.value
}

func parseIPRange(s string) (*IPRange, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid ip range: %s", s)
	}
	r := &IPRange{value: s}
	for idx, part := range parts {
		octet, err := parseRangeOctet(part)
		if err != nil {
			return nil, fmt.Errorf("invalid ip range: %s", s)
		}
		r.octets[idx] = octet
	}
	return r, nil
}

func parseRangeOctet(part string) (octetRange, error) {
	if strings.Contains(part, "-") {
		bounds := strings.Split(part, "-")
		if len(bounds) != 2 {
			return octetRange{}, fmt.Errorf("invalid range octet")
		}
		start, err := parseOctet(bounds[0])
		if err != nil {
			return octetRange{}, err
		}
		end, err := parseOctet(bounds[1])
		if err != nil {
			return octetRange{}, err
		}
		if start > end {
			return octetRange{}, fmt.Errorf("invalid range octet")
		}
		return octetRange{start: start, end: end}, nil
	}
	value, err := parseOctet(part)
	if err != nil {
		return octetRange{}, err
	}
	return octetRange{start: value, end: value}, nil
}

func parseOctet(part string) (int, error) {
	if part == "" {
		return 0, fmt.Errorf("invalid octet")
	}
	value, err := strconv.Atoi(part)
	if err != nil {
		return 0, err
	}
	if value < 0 || value > 255 {
		return 0, fmt.Errorf("invalid octet")
	}
	return value, nil
}

func (r *IPRange) Contains(ip *IP) bool {
	if r == nil || ip == nil || ip.ip == nil {
		return false
	}
	ipv4 := ip.ip.To4()
	if ipv4 == nil {
		return false
	}
	for idx, octet := range ipv4 {
		value := int(octet)
		if value < r.octets[idx].start || value > r.octets[idx].end {
			return false
		}
	}
	return true
}
