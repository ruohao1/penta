package runner

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type HostGate interface {
	Acquire(ctx context.Context, key string) error
	Release(key string)
}

type Pool struct {
	MaxInFlight int
	Limiter     *rate.Limiter
	Gate        HostGate // optional
}

func (p Pool) Run(ctx context.Context, jobs []Job) error {
	if p.MaxInFlight < 1 {
		p.MaxInFlight = 1
	}

	ch := make(chan Job)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	worker := func() {
		defer wg.Done()
		for job := range ch {
			// global rate limit
			if p.Limiter != nil {
				if err := p.Limiter.Wait(ctx); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
			// per-host gate
			key := job.Key()
			if p.Gate != nil && key != "" {
				if err := p.Gate.Acquire(ctx, key); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
			err := job.Run(ctx)
			if p.Gate != nil && key != "" {
				p.Gate.Release(key)
			}
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}

	wg.Add(p.MaxInFlight)
	for i := 0; i < p.MaxInFlight; i++ {
		go worker()
	}

	go func() {
		defer close(ch)
		for _, j := range jobs {
			select {
			case <-ctx.Done():
				return
			case ch <- j:
			}
		}
	}()

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return ctx.Err()
	}
}
