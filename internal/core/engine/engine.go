// Package engine provides a generic engine for running checks
package engine

import (
	"context"
	"time"

	"github.com/Ruohao1/penta/internal/core/events"
	"github.com/Ruohao1/penta/internal/core/runner"
	"github.com/Ruohao1/penta/internal/core/sinks"
	"github.com/Ruohao1/penta/internal/core/stages"
	"github.com/Ruohao1/penta/internal/core/tasks"
	"github.com/Ruohao1/penta/internal/core/types"
	"golang.org/x/time/rate"
)

type Engine struct {
	Stages []stages.Stage
	Pool   func(opts types.RunOptions) runner.Pool
	Sink   sinks.Sink
}

func (e Engine) Run(ctx context.Context, task tasks.Task, opts types.RunOptions) error {
	pool := e.Pool(opts)
	if e.Sink != nil {
		_ = e.Sink.Emit(ctx, events.Event{
			EmittedAt: time.Now().UTC(),
			Type:      events.EventEngineStart,
			Message:   "engine started",
		})
	}
	for _, st := range e.Stages {
		jobs, err := st.Build(ctx, task, opts, e.Sink)
		if err != nil {
			if e.Sink != nil {
				_ = e.Sink.Emit(ctx, events.Event{
					EmittedAt: time.Now().UTC(),
					Type:      events.EventError,
					Stage:     st.Name(),
					Err:       err.Error(),
				})
			}
			return err
		}
		if e.Sink != nil {
			_ = e.Sink.Emit(ctx, events.Event{
				EmittedAt: time.Now().UTC(),
				Type:      events.EventScanStart,
				Stage:     st.Name(),
				Progress: &events.Progress{
					TotalTargets: len(jobs),
				},
			})
		}
		if err = pool.Run(ctx, jobs); err != nil {
			if e.Sink != nil {
				_ = e.Sink.Emit(ctx, events.Event{
					EmittedAt: time.Now().UTC(),
					Type:      events.EventError,
					Stage:     st.Name(),
					Err:       err.Error(),
				})
			}
			return err
		}
		if err = st.After(ctx, task, opts, e.Sink); err != nil {
			if e.Sink != nil {
				_ = e.Sink.Emit(ctx, events.Event{
					EmittedAt: time.Now().UTC(),
					Type:      events.EventError,
					Stage:     st.Name(),
					Err:       err.Error(),
				})
			}
			return err
		}
		if e.Sink != nil {
			_ = e.Sink.Emit(ctx, events.Event{
				EmittedAt: time.Now().UTC(),
				Type:      events.EventScanDone,
				Stage:     st.Name(),
			})
		}
	}
	if e.Sink != nil {
		_ = e.Sink.Emit(ctx, events.Event{
			EmittedAt: time.Now().UTC(),
			Type:      events.EventEngineDone,
			Message:   "engine completed",
		})
	}
	return nil
}

func DefaultPool(opts types.RunOptions) runner.Pool {
	var lim *rate.Limiter
	if opts.Limits.MaxRate > 0 {
		burst := opts.Limits.MaxRate
		lim = rate.NewLimiter(rate.Limit(opts.Limits.MaxRate), burst)
	}
	var gate runner.HostGate
	if opts.Limits.MaxInFlightPerHost > 0 {
		gate = &runner.PerHostGate{N: opts.Limits.MaxInFlightPerHost}
	}
	return runner.Pool{
		MaxInFlight: opts.Limits.MaxInFlight,
		Limiter:     lim,
		Gate:        gate,
	}
}
