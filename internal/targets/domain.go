package targets

import (
	"fmt"
	"regexp"
	"strings"
)

var _ Target = (*Domain)(nil)

var domainPattern = regexp.MustCompile(`^(?i)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+\.?$`)

type Domain struct {
	name string
}

func (d *Domain) Type() Type {
	return TypeDomain
}

func (d *Domain) String() string {
	return d.name
}

func parseDomain(s string) (*Domain, error) {
	if strings.ContainsAny(s, "/:@? ") {
		return nil, fmt.Errorf("invalid domain: %s", s)
	}
	if !domainPattern.MatchString(s) {
		return nil, fmt.Errorf("invalid domain: %s", s)
	}
	return &Domain{name: strings.ToLower(strings.TrimSuffix(s, "."))}, nil
}
