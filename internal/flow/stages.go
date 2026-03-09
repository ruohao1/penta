package flow

import "context"

type Stage interface {
	Name() string
	Process(ctx context.Context, item Item) ([]Item, error)
}
