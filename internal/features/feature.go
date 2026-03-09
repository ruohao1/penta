package features

import (
	"context"

	"github.com/ruohao1/penta/internal/flow"
)

type BuildInput = flow.BuildInput
type Plan = flow.Plan

type Feature interface {
	Type() flow.Type
	Build(ctx context.Context, in BuildInput) (Plan, error)
}
