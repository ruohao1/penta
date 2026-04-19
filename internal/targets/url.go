package targets

import (
	"fmt"
	"net"
	neturl "net/url"
)

var _ Target = (*URL)(nil)

type URL struct {
	Scheme string
	Host   string
	Port   string
	Path   string
	Query  string
}

func (u *URL) Type() Type {
	return TypeURL
}

func (u *URL) String() string {
	host := u.Host
	if u.Port != "" {
		host = net.JoinHostPort(host, u.Port)
	}
	parsed := neturl.URL{
		Scheme:   u.Scheme,
		Host:     host,
		Path:     u.Path,
		RawQuery: u.Query,
	}
	return parsed.String()
}

func parseURL(s string) (*URL, error) {
	parsed, err := neturl.Parse(s)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid url: %s", s)
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("invalid url: %s", s)
	}
	return &URL{
		Scheme: parsed.Scheme,
		Host:   host,
		Port:   parsed.Port(),
		Path:   parsed.EscapedPath(),
		Query:  parsed.RawQuery,
	}, nil
}
