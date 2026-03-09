package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/flow"
	"github.com/ruohao1/penta/internal/sinks"
	"github.com/ruohao1/pipex"
)

type Plan = flow.Plan
type Sink = sinks.Sink

type Runtime struct {
	config Config
	sink   Sink
}

func New(config Config, sink Sink) *Runtime {
	if config == (Config{}) {
		config = DefaultConfig()
	}
	return &Runtime{config: config, sink: sink}
}

func (r *Runtime) Run(ctx context.Context, plan Plan, sink Sink) error {
	if sink == nil {
		sink = r.sink
	}

	opts := r.config.ToPipelineOpts(plan)
	findingStages := findingStagesForFeature(plan.Feature)
	if sink != nil && len(findingStages) > 0 {
		findingSinks := make([]pipex.Sink[Item], 0, len(findingStages))
		for _, stage := range findingStages {
			findingSinks = append(findingSinks, findingPipexSink{
				feature: plan.Feature,
				stage:   stage,
				out:     sink,
			})
		}
		opts = append(opts, pipex.WithSinks[Item](findingSinks...))
	}

	hooks := pipex.Hooks[Item]{
		RunStart: func(ctx context.Context, meta pipex.RunMeta) {
			emit(ctx, sink, events.Event{
				At:      time.Now(),
				Kind:    events.RunStarted,
				RunID:   meta.RunID,
				Feature: string(plan.Feature),
				Data: map[string]any{
					"seed_stages": meta.SeedStages,
					"seed_items":  meta.SeedItems,
					"stage_count": meta.StageCount,
				},
			})
		},
		RunEnd: func(ctx context.Context, meta pipex.RunMeta, err error) {
			kind := events.RunFinished
			msg := "run finished"
			errStr := ""
			if err != nil {
				errStr = err.Error()
				switch {
				case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
					kind = events.RunCanceled
					msg = "run canceled"
				default:
					kind = events.RunFailed
					msg = "run failed"
				}
			}
			emit(ctx, sink, events.Event{
				At:      time.Now(),
				Kind:    kind,
				RunID:   meta.RunID,
				Feature: string(plan.Feature),
				Message: msg,
				Err:     errStr,
			})
		},
		StageStart: func(ctx context.Context, e pipex.StageStartEvent[Item]) {
			emit(ctx, sink, events.Event{
				At:      e.StartedAt,
				Kind:    events.StageStarted,
				RunID:   e.RunID,
				Feature: string(plan.Feature),
				Stage:   e.Stage,
				Target:  e.Input.Target,
			})
		},
		StageFinish: func(ctx context.Context, e pipex.StageFinishEvent[Item]) {
			emit(ctx, sink, events.Event{
				At:      e.FinishedAt,
				Kind:    events.StageFinished,
				RunID:   e.RunID,
				Feature: string(plan.Feature),
				Stage:   e.Stage,
				Target:  e.Input.Target,
				Data: map[string]any{
					"out_count":   e.OutCount,
					"duration_ms": e.Duration.Milliseconds(),
				},
			})
		},
		StageRetry: func(ctx context.Context, e pipex.StageRetryEvent[Item]) {
			emit(ctx, sink, events.Event{
				At:      e.At,
				Kind:    events.StageRetry,
				RunID:   e.RunID,
				Feature: string(plan.Feature),
				Stage:   e.Stage,
				Target:  e.Input.Target,
				Err:     errorString(e.Err),
				Data: map[string]any{
					"attempt":    e.Attempt,
					"backoff_ms": e.Backoff.Milliseconds(),
				},
			})
		},
		StageExhausted: func(ctx context.Context, e pipex.StageExhaustedEvent[Item]) {
			emit(ctx, sink, events.Event{
				At:      e.At,
				Kind:    events.StageFailed,
				RunID:   e.RunID,
				Feature: string(plan.Feature),
				Stage:   e.Stage,
				Target:  e.Input.Target,
				Err:     errorString(e.Err),
				Data: map[string]any{
					"attempts": e.Attempts,
				},
			})
		},
		StageError: func(ctx context.Context, e pipex.StageErrorEvent[Item]) {
			emit(ctx, sink, events.Event{
				At:      e.FinishedAt,
				Kind:    events.StageFailed,
				RunID:   e.RunID,
				Feature: string(plan.Feature),
				Stage:   e.Stage,
				Target:  e.Input.Target,
				Err:     errorString(e.Err),
				Data: map[string]any{
					"duration_ms": e.Duration.Milliseconds(),
				},
			})
		},
	}
	opts = append(opts, pipex.WithHooks[Item](hooks))

	p := pipex.NewPipeline[Item]()

	for _, stage := range plan.Stages {
		if err := p.AddStage(stage); err != nil {
			return err
		}
	}

	for _, edge := range plan.Edges {
		if err := p.Connect(edge.From, edge.To); err != nil {
			return err
		}
	}

	_, err := p.Run(
		ctx,
		plan.Seeds,
		opts...,
	)

	if err != nil {
		return err
	}

	return nil
}

func emit(ctx context.Context, sink Sink, ev events.Event) {
	if sink == nil {
		return
	}
	if ev.At.IsZero() {
		ev.At = time.Now()
	}
	_ = sink.Emit(ctx, ev)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func payloadData(payload any) map[string]any {
	if payload == nil {
		return nil
	}
	v := reflect.ValueOf(payload)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	out := map[string]any{}
	switch v.Kind() {
	case reflect.Map:
		if m, ok := payload.(map[string]any); ok {
			for k, val := range m {
				out[k] = val
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			fv := v.Field(i)
			if !fv.IsValid() || (fv.Kind() == reflect.Pointer && fv.IsNil()) {
				continue
			}
			key := camelToSnake(f.Name)
			switch fv.Kind() {
			case reflect.String:
				out[key] = fv.String()
			case reflect.Bool:
				out[key] = fv.Bool()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				out[key] = fv.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				out[key] = fv.Uint()
			case reflect.Float32, reflect.Float64:
				out[key] = fv.Float()
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func camelToSnake(in string) string {
	if in == "" {
		return in
	}
	runes := []rune(in)
	var b strings.Builder
	b.Grow(len(runes) + 4)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				b.WriteByte('_')
			}
		}
		if unicode.IsUpper(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
