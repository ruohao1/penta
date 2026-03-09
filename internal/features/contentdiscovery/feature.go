package contentdiscovery

import (
	"context"
	"fmt"

	"github.com/ruohao1/penta/internal/flow"
	"github.com/ruohao1/pipex"
)

type Feature struct{}

func New() *Feature {
	return &Feature{}
}

func (f *Feature) Type() flow.Type {
	return flow.ContentDiscovery
}

func (f *Feature) Build(ctx context.Context, in flow.BuildInput) (flow.Plan, error) {
	_ = ctx
	opts, ok := in.Task.(Options)
	if !ok {
		ptr, okPtr := in.Task.(*Options)
		if !okPtr || ptr == nil {
			return flow.Plan{}, fmt.Errorf("content discovery: expected task options %T", in.Task)
		}
		opts = *ptr
	}
	if err := opts.Validate(); err != nil {
		return flow.Plan{}, err
	}

	seed := NewSeedStage(1)
	discover, err := NewDiscoverStage(opts)
	if err != nil {
		return flow.Plan{}, fmt.Errorf("create discover stage: %w", err)
	}

	seeds := make([]flow.Item, 0, len(opts.Targets))
	for _, target := range opts.Targets {
		seeds = append(seeds, flow.Item{
			Feature: string(flow.ContentDiscovery),
			Stage:   seed.Name(),
			Target:  target,
			Key:     target,
			Payload: SeedPayload{
				RawTarget: target,
				Depth:     opts.MaxDepth,
			},
		})
	}

	return flow.Plan{
		Feature: flow.ContentDiscovery,
		Stages: []pipex.Stage[flow.Item]{
			seed,
			discover,
		},
		Edges: []flow.Edge{
			{From: seed.Name(), To: discover.Name()},
		},
		Seeds: map[string][]flow.Item{
			seed.Name(): seeds,
		},
	}, nil
}
