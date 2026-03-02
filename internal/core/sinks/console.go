package sinks

import (
	"context"
	"io"

	"github.com/Ruohao1/penta/internal/core/events"
)

// ConsoleSink writes human-readable event lines to output streams.
type ConsoleSink struct {
	out io.Writer
	err io.Writer
}

func NewConsoleSink(out io.Writer, err io.Writer) *ConsoleSink {
	return &ConsoleSink{out: out, err: err}
}

func (s *ConsoleSink) Emit(ctx context.Context, ev events.Event) error {
	_ = ctx
	line := ev.String() + "\n"
	if ev.Type == events.EventError {
		if s.err != nil {
			_, _ = io.WriteString(s.err, line)
			return nil
		}
	}
	if s.out != nil {
		_, _ = io.WriteString(s.out, line)
	}
	return nil
}

func (s *ConsoleSink) Close() error { return nil }
