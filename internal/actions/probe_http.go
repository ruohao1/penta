package actions

import "github.com/ruohao1/penta/internal/targets"

type ProbeHTTPInput = targets.TargetRef

type ServiceEvidence struct {
	Scheme string `json:"scheme,omitempty"`
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
}
