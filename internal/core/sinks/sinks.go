package sinks

import (
	"context"
	"errors"
	"io"

	"github.com/Ruohao1/penta/internal/core/events"
)

type Sink interface {
	Emit(context.Context, events.Event) error
	Close() error
}

type SinkOptions struct {
	Human   bool
	Verbose int
	Out     io.Writer
	Err     io.Writer
	NDJSON  io.Writer
	Summary bool
}

type PentaSink struct {
	sinks []Sink
}

func NewPentaSink(opts SinkOptions) *PentaSink {
	var sinkList []Sink
	if opts.Human {
		sinkList = append(sinkList, NewConsoleSink(opts.Out, opts.Err))
	}
	return &PentaSink{sinks: sinkList}
}

func (p *PentaSink) Emit(ctx context.Context, ev events.Event) error {
	var errs []error
	for _, s := range p.sinks {
		if err := s.Emit(ctx, ev); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *PentaSink) Close() error {
	var errs []error
	for _, s := range p.sinks {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
