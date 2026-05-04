package targets

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

var _ Target = (*Service)(nil)

type Service struct {
	Host string
	Port string
}

func (s *Service) Type() Type {
	return TypeService
}

func (s *Service) String() string {
	if strings.Contains(s.Host, ":") {
		return net.JoinHostPort(s.Host, s.Port)
	}
	return s.Host + ":" + s.Port
}

func parseService(s string) (*Service, error) {
	host, port, err := splitService(s)
	if err != nil {
		return nil, fmt.Errorf("invalid service: %s", s)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid service: %s", s)
	}
	portNum, _ := strconv.Atoi(port)
	if portNum < 1 || portNum > 65535 {
		return nil, fmt.Errorf("invalid service: %s", s)
	}
	if !strings.EqualFold(host, "localhost") {
		if _, err := parseIP(host); err != nil {
			if _, err := parseDomain(host); err != nil {
				return nil, fmt.Errorf("invalid service: %s", s)
			}
		}
	}
	return &Service{Host: host, Port: strconv.Itoa(portNum)}, nil
}

func splitService(s string) (string, string, error) {
	if strings.HasPrefix(s, "[") {
		host, port, err := net.SplitHostPort(s)
		return host, port, err
	}
	if strings.Count(s, ":") != 1 {
		return "", "", fmt.Errorf("invalid service: %s", s)
	}
	parts := strings.SplitN(s, ":", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid service: %s", s)
	}
	return parts[0], parts[1], nil
}
