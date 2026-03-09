package sinks

import (
	"context"

	"github.com/ruohao1/penta/internal/events"
)

type Sink interface {
	Emit(ctx context.Context, ev events.Event) error
	Close() error
}
