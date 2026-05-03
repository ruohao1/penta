package probe_http

import (
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/targets"
)

type Input struct {
	Value string       `json:"value"`
	Type  targets.Type `json:"type"`
}

type Evidence = model.Service
