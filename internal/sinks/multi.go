package sinks

import (
	"context"
	"errors"

	"github.com/ruohao1/penta/internal/events"
)

type MultiSink struct {
	sinks []Sink
}

func NewMultiSink(sinks ...Sink) *MultiSink {
	list := make([]Sink, 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			list = append(list, s)
		}
	}
	return &MultiSink{sinks: list}
}

func (m *MultiSink) Emit(ctx context.Context, ev events.Event) error {
	if m == nil {
		return nil
	}
	var errs []error
	for _, s := range m.sinks {
		if err := s.Emit(ctx, ev); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *MultiSink) Close() error {
	if m == nil {
		return nil
	}
	var errs []error
	for _, s := range m.sinks {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
