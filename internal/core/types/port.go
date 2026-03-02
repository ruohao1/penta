package types

type PortState string

const (
	PortStateOpen     PortState = "open"
	PortStateClosed   PortState = "closed"
	PortStateFiltered PortState = "filtered"
	PortStateUnknown  PortState = "unknown"
)

type Port struct {
	Number int       `json:"port"`  // 1–65535
	Proto  string    `json:"proto"` // "tcp" or "udp"
	State  PortState `json:"state"` // open/closed/filtered
	// Why we think it's in this state
	Reason string  `json:"reason,omitempty"` // "syn-ack","rst","timeout"
	RTTms  float64 `json:"rtt_ms,omitempty"` // probe RTT

	// One or more services detected on this port
	Services []Service `json:"services,omitempty"`
}
