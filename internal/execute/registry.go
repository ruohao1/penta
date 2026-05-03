package execute

import (
	"context"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	seedtarget "github.com/ruohao1/penta/internal/actions/seed_target"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type Handler func(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error

func handlers() map[actions.ActionType]Handler {
	return map[actions.ActionType]Handler{
		actions.ActionSeedTarget: seedtarget.Execute,
		actions.ActionProbeHTTP:  probehttp.Execute,
	}
}
