package types

type WorkType string

const (
	WorkHost WorkType = "host" // host discovery/liveness
	WorkPort WorkType = "port" // port connectivity
	WorkHTTP WorkType = "http" // app-level over http(s)
	WorkTLS  WorkType = "tls"  // tls inventory
)

type WorkItem struct {
	Type WorkType
	// Target Target
	Port   int
	Proto  string         // "tcp", "udp", "http", "https"
	Method string         // host discovery method: "icmp"/"arp"/"tcp"
	Meta   map[string]any // extra context (e.g. "url", "sni")
}
