package events

import "context"

type Sink interface {
	Append(ctx context.Context, evt Event) error
	ListByRunSinceSeq(ctx context.Context, runID string, seq int64, limit int) ([]Event, error)
}
