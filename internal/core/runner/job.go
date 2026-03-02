package runner

import "context"

type Job interface {
	// Key is used for per-host gating. Empty means "no gate".
	Key() string
	Run(ctx context.Context) error
}
