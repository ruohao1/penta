package sinks

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ruohao1/penta/internal/events"
)

type ConsoleSink struct {
	out io.Writer
	mu  sync.Mutex
}

func NewConsoleSink(out io.Writer) *ConsoleSink {
	if out == nil {
		out = os.Stdout
	}
	return &ConsoleSink{out: out}
}

func (s *ConsoleSink) Emit(ctx context.Context, ev events.Event) error {
	_ = ctx
	if s == nil || s.out == nil {
		return nil
	}

	ts := ev.At
	if ts.IsZero() {
		ts = time.Now()
	}

	parts := []string{
		ts.Format(time.RFC3339),
		string(ev.Kind),
	}
	if ev.Feature != "" {
		parts = append(parts, "feature="+ev.Feature)
	}
	if ev.Stage != "" {
		parts = append(parts, "stage="+ev.Stage)
	}
	if ev.Target != "" {
		parts = append(parts, "target="+ev.Target)
	}
	if ev.Message != "" {
		parts = append(parts, "msg="+ev.Message)
	}
	if ev.Err != "" {
		parts = append(parts, "err="+ev.Err)
	}
	if ev.Kind == events.Finding && ev.Data != nil {
		if v, ok := ev.Data["status_code"]; ok {
			parts = append(parts, "status="+fmt.Sprint(v))
		}
		if v, ok := ev.Data["url"].(string); ok && v != "" {
			parts = append(parts, "url="+v)
		}
		if v, ok := ev.Data["depth"]; ok {
			parts = append(parts, "depth="+fmt.Sprint(v))
		}
		if v, ok := ev.Data["path"].(string); ok && v != "" {
			parts = append(parts, "path="+v)
		}
		if v, ok := ev.Data["content_length"]; ok {
			parts = append(parts, "size="+fmt.Sprint(v))
		}
		if v, ok := ev.Data["duration_ms"]; ok {
			parts = append(parts, "latency_ms="+fmt.Sprint(v))
		}
	}

	line := strings.Join(parts, " ")

	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := fmt.Fprintln(s.out, line)
	return err
}

func (s *ConsoleSink) Close() error { return nil }
