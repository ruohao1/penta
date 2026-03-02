package stages

import (
	"context"

	"github.com/Ruohao1/penta/internal/core/runner"
	"github.com/Ruohao1/penta/internal/core/sinks"
	"github.com/Ruohao1/penta/internal/core/tasks"
	"github.com/Ruohao1/penta/internal/core/types"
)

type Stage interface {
	Name() string
	Build(ctx context.Context, task tasks.Task, opts types.RunOptions, sink sinks.Sink) ([]runner.Job, error)
	// Optional: called after pool completes for this stage
	After(ctx context.Context, task tasks.Task, opts types.RunOptions, sink sinks.Sink) error
}
