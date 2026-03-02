package sinks

import (
	"context"

	"github.com/Ruohao1/penta/internal/core/events"
)

// ChannelSink forwards events to an in-memory channel, intended for TUI updates.
// It drops events when the channel is full to avoid blocking scan workers.
type ChannelSink struct {
	ch chan<- events.Event
}

func NewChannelSink(ch chan<- events.Event) *ChannelSink {
	return &ChannelSink{ch: ch}
}

func (s *ChannelSink) Emit(ctx context.Context, ev events.Event) error {
	if s == nil || s.ch == nil {
		return nil
	}
	select {
	case s.ch <- ev:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *ChannelSink) Close() error { return nil }
