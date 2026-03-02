package types

import (
	"time"
)

type GlobalOptions struct {
	Verbosity int
	Human     bool
	LogFile   string

	// TODO: add other global options here
}

type RunOptions struct {
	Privileged bool
	Scope      []string // allow-list enforcement

	ProbeOpts ProbeOptions
	Limits    Limits
	Timeouts  Timeouts
}

type Limits struct {
	MaxInFlight        int
	MaxInFlightPerHost int
	MaxRate            int
	MinRate            int
	MaxRetries         int
}

type Timeouts struct {
	Overall time.Duration
	TCP     time.Duration
	HTTP    time.Duration
	TLS     time.Duration
}

type ProbeOptions struct {
	TCP  bool
	ICMP bool
	ARP  bool
	HTTP bool
	TLS  bool

	Jitter     time.Duration
	UserAgent  string
	Proxy      string
	DNSServers []string
}
