package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	seedtarget "github.com/ruohao1/penta/internal/actions/seed_target"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/targets"
)

type Request struct {
	Raw    string
	Action actions.ActionType
}

func (e *Executor) Resolve(ctx context.Context, runID string, request Request) error {
	if err := e.appendEvent(ctx, events.Event{
		RunID:       runID,
		EventType:   events.EventActionRequested,
		EntityKind:  events.EntityRun,
		EntityID:    runID,
		PayloadJSON: mustPayloadJSON(events.ActionRequestedPayload{Action: request.Action, Raw: request.Raw}),
		CreatedAt:   time.Now(),
	}); err != nil {
		return err
	}

	var enqueued []actions.ActionType
	switch request.Action {
	case actions.ActionSeedTarget:
		if err := e.enqueueSeedTarget(ctx, runID, request.Raw); err != nil {
			return err
		}
		enqueued = append(enqueued, actions.ActionSeedTarget)
	case actions.ActionProbeHTTP:
		target, err := targets.Parse(request.Raw)
		if err != nil {
			return err
		}

		if err := e.enqueueSeedTarget(ctx, runID, request.Raw); err != nil {
			return err
		}
		enqueued = append(enqueued, actions.ActionSeedTarget)

		if err := e.enqueueProbeHTTP(ctx, runID, probehttp.Input{
			Value: target.String(),
			Type:  target.Type(),
		}); err != nil {
			return err
		}
		enqueued = append(enqueued, actions.ActionProbeHTTP)
	default:
		return fmt.Errorf("unsupported requested action: %s", request.Action)
	}

	if len(enqueued) > 0 {
		if err := e.appendEvent(ctx, events.Event{
			RunID:       runID,
			EventType:   events.EventActionResolved,
			EntityKind:  events.EntityRun,
			EntityID:    runID,
			PayloadJSON: mustPayloadJSON(events.ActionResolvedPayload{RequestedAction: request.Action, EnqueuedActions: enqueued}),
			CreatedAt:   time.Now(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) enqueueSeedTarget(ctx context.Context, runID, raw string) error {
	inputJSON, err := json.Marshal(seedtarget.Input{Raw: raw})
	if err != nil {
		return err
	}
	return e.frontier().enqueueTask(ctx, runID, actions.ActionSeedTarget, string(inputJSON))
}

func (e *Executor) enqueueProbeHTTP(ctx context.Context, runID string, input probehttp.Input) error {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return e.frontier().enqueueTask(ctx, runID, actions.ActionProbeHTTP, string(inputJSON))
}
