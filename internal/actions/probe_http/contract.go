package probe_http

import "github.com/ruohao1/penta/internal/targets"

type Input struct {
	Value string       `json:"value"`
	Type  targets.Type `json:"type"`
}

type ServiceEvidence struct {
	Scheme string `json:"scheme,omitempty"`
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
}
