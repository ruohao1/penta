package types

import "time"

type Finding struct {
	ObservedAt time.Time `json:"observed_at"`
	Check      string    `json:"check"` // "tcp_connect", "tls_handshake", "http_probe"
	Proto      Protocol  `json:"proto"`

	Endpoint Endpoint `json:"endpoint"`           // where it happened
	Severity string   `json:"severity,omitempty"` // info/low/med/high
	Status   string   `json:"status"`             // "ok"|"fail"|"timeout"|"refused"|"unreachable"...

	RTTMs float64        `json:"rtt_ms,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"` // check-specific payload
}
