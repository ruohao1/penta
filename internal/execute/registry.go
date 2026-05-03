package execute

import (
	"context"
	"fmt"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	seedtarget "github.com/ruohao1/penta/internal/actions/seed_target"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type Handler func(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error

type RegisteredAction struct {
	Spec    actions.ActionSpec
	Handler Handler
}

func registry() map[actions.ActionType]RegisteredAction {
	return map[actions.ActionType]RegisteredAction{
		actions.ActionSeedTarget: {Spec: seedtarget.Spec, Handler: seedtarget.Execute},
		actions.ActionProbeHTTP:  {Spec: probehttp.Spec, Handler: probehttp.Execute},
	}
}

func handlers() map[actions.ActionType]Handler {
	registered := registry()
	out := make(map[actions.ActionType]Handler, len(registered))
	for actionType, action := range registered {
		out[actionType] = action.Handler
	}
	return out
}

func validateRegistry(registered map[actions.ActionType]RegisteredAction) error {
	seen := map[actions.ActionType]struct{}{}
	for actionType, action := range registered {
		if actionType == "" {
			return fmt.Errorf("registered action key is required")
		}
		if action.Spec.Type == "" {
			return fmt.Errorf("action %s spec type is required", actionType)
		}
		if action.Spec.Type != actionType {
			return fmt.Errorf("action %s spec type mismatch: %s", actionType, action.Spec.Type)
		}
		if action.Spec.Permission == "" {
			return fmt.Errorf("action %s permission is required", actionType)
		}
		if action.Spec.InputKind == "" {
			return fmt.Errorf("action %s input kind is required", actionType)
		}
		if action.Handler == nil {
			return fmt.Errorf("action %s handler is required", actionType)
		}
		if _, ok := seen[actionType]; ok {
			return fmt.Errorf("duplicate action registration: %s", actionType)
		}
		seen[actionType] = struct{}{}
	}
	return nil
}
