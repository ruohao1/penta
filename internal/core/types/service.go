package types

import "time"

type Service struct {
	// Generic service info
	Name    string `json:"name,omitempty"`    // "ssh","http","redis"
	Product string `json:"product,omitempty"` // "OpenSSH","nginx","Redis"
	Version string `json:"version,omitempty"` // "9.6p1","1.24.0"
	Banner  string `json:"banner,omitempty"`  // raw banner, possibly truncated

	// Optional HTTP-specific info (if this looks like HTTP)
	HTTP *HTTPInfo `json:"http,omitempty"`

	// Optional TLS-specific info (if TLS handshake succeeded)
	TLS *TLSInfo `json:"tls,omitempty"`

	// Extra check-specific data you don't want to model yet
	Meta map[string]any `json:"meta,omitempty"`
}

// Optional HTTP-layer details for an HTTP(S) service.
type HTTPInfo struct {
	Path      string   `json:"path,omitempty"`      // usually "/"
	Status    int      `json:"status,omitempty"`    // 200, 301, 404...
	Title     string   `json:"title,omitempty"`     // <title> of the page
	Server    string   `json:"server,omitempty"`    // Server: header
	Redirects []string `json:"redirects,omitempty"` // URL chain
	Tech      []string `json:"tech,omitempty"`      // "nginx","wordpress","react"
}

// Optional TLS-layer details for a TLS-wrapped service.
type TLSInfo struct {
	Version    string    `json:"version,omitempty"` // "tls12","tls13"
	Cipher     string    `json:"cipher,omitempty"`  // negotiated cipher
	ALPN       []string  `json:"alpn,omitempty"`    // "h2","http/1.1"
	SNI        string    `json:"sni,omitempty"`     // SNI used during probe
	CN         string    `json:"cn,omitempty"`      // leaf CN
	SANs       []string  `json:"sans,omitempty"`    // DNS names
	Issuer     string    `json:"issuer,omitempty"`  // issuer CN
	NotBefore  time.Time `json:"not_before,omitempty"`
	NotAfter   time.Time `json:"not_after,omitempty"`
	SelfSigned bool      `json:"self_signed,omitempty"`
	KeyType    string    `json:"key_type,omitempty"` // "rsa","ecdsa"
	KeyBits    int       `json:"key_bits,omitempty"` // 2048, 4096, ...
}
