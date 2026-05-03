package seed_target

import "github.com/ruohao1/penta/internal/model"

type Input struct {
	Raw string `json:"raw"`
}

type Evidence = model.TargetRef
