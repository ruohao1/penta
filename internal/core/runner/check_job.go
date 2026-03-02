package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Ruohao1/penta/internal/checks"
	"github.com/Ruohao1/penta/internal/core/events"
	"github.com/Ruohao1/penta/internal/core/sinks"
	"github.com/Ruohao1/penta/internal/core/types"
)

type CheckJob struct {
	StageName string
	HostKey   string

	Checker checks.Checker
	Input   any

	Sink sinks.Sink
}

func (j CheckJob) Key() string { return j.HostKey }

func (j CheckJob) Run(ctx context.Context) error {
	_ = j.Sink.Emit(ctx, events.Event{
		EmittedAt: time.Now().UTC(),
		Type:      events.EventProbeStart,
		Stage:     j.StageName,
		Target:    j.HostKey,
	})

	emit := func(x any) {
		switch v := x.(type) {
		case types.Finding:
			ev := events.NewFindingEvent(&v)
			ev.Stage = j.StageName
			ev.Target = j.HostKey
			ev.EmittedAt = time.Now().UTC()
			j.Sink.Emit(ctx, ev)

		default:
			j.Sink.Emit(ctx, events.Event{
				EmittedAt: time.Now().UTC(),
				Type:      events.EventUnknown,
				Stage:     j.StageName,
				Err:       fmt.Sprintf("unknown event type: %T", x),
			})
		}
	}
	err := j.Checker.Check()(ctx, j.Input, emit)
	if err != nil {
		_ = j.Sink.Emit(ctx, events.Event{
			EmittedAt: time.Now().UTC(),
			Type:      events.EventError,
			Stage:     j.StageName,
			Target:    j.HostKey,
			Err:       err.Error(),
		})
		return err
	}

	_ = j.Sink.Emit(ctx, events.Event{
		EmittedAt: time.Now().UTC(),
		Type:      events.EventProbeDone,
		Stage:     j.StageName,
		Target:    j.HostKey,
	})
	return nil
}
