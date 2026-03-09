package sinks

import (
	"context"

	"github.com/ruohao1/penta/internal/events"
)

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
	}
}

func (s *ChannelSink) Close() error { return nil }
