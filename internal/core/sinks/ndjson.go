package sinks

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/Ruohao1/penta/internal/core/events"
)

type NDJSONSink struct {
	mu sync.Mutex
	w  io.Writer
}

func NewNDJSONSink(w io.Writer) *NDJSONSink { return &NDJSONSink{w: w} }

func (s *NDJSONSink) Emit(ctx context.Context, ev events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err = s.w.Write(b); err != nil {
		return err
	}
	_, err = s.w.Write([]byte("\n"))
	return err
}

func (s *NDJSONSink) Close() error { return nil }
