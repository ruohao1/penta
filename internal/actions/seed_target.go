package actions

import "github.com/ruohao1/penta/internal/targets"

type SeedTargetInput struct {
	Raw string `json:"raw"`
}

type SeedTargetEvidence = targets.TargetRef
