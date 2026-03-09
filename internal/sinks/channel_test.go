package sinks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/events"
)

func TestChannelSinkEmitDeliversEvent(t *testing.T) {
	t.Parallel()

	ch := make(chan events.Event, 1)
	sink := NewChannelSink(ch)
	ev := events.Event{Kind: events.Finding, Target: "https://example.com/"}

	if err := sink.Emit(context.Background(), ev); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	got := <-ch
	if got.Kind != ev.Kind {
		t.Fatalf("Kind = %q, want %q", got.Kind, ev.Kind)
	}
	if got.Target != ev.Target {
		t.Fatalf("Target = %q, want %q", got.Target, ev.Target)
	}
}

func TestChannelSinkEmitRespectsContextCancel(t *testing.T) {
	t.Parallel()

	ch := make(chan events.Event)
	sink := NewChannelSink(ch)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := sink.Emit(ctx, events.Event{Kind: events.Log})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Emit() error = %v, want deadline exceeded", err)
	}
}
