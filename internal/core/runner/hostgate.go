package runner

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

type PerHostGate struct {
	N int // max in-flight per host
	m sync.Map
}

func (g *PerHostGate) sem(key string) *semaphore.Weighted {
	if g.N < 1 {
		g.N = 1
	}
	v, _ := g.m.LoadOrStore(key, semaphore.NewWeighted(int64(g.N)))
	return v.(*semaphore.Weighted)
}

func (g *PerHostGate) Acquire(ctx context.Context, key string) error {
	return g.sem(key).Acquire(ctx, 1)
}

func (g *PerHostGate) Release(key string) {
	g.sem(key).Release(1)
}
