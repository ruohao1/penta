package actions

import "github.com/ruohao1/penta/internal/targets"

type ResolveDNSInput = targets.TargetRef

type ResolveDNSEvidence struct {
	IPs []string `json:"ips"`
}
