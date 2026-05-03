package seed_target

import "github.com/ruohao1/penta/internal/targets"

type Input struct {
	Raw string `json:"raw"`
}

type Evidence = targets.TargetRef
