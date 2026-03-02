package types

type Protocol string

const (
	ProtocolTCP   Protocol = "tcp"
	ProtocolUDP   Protocol = "udp"
	ProtocolICMP  Protocol = "icmp"
	ProtocolARP   Protocol = "arp"
	ProtocolDNS   Protocol = "dns"
	ProtocolTLS   Protocol = "tls"
	ProtocolHTTP  Protocol = "http"
	ProtocolHTTPS Protocol = "https"
)
