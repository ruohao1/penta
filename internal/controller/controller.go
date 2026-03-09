package controller

import (
	"context"
	"errors"

	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/features"
	"github.com/ruohao1/penta/internal/features/contentdiscovery"
	"github.com/ruohao1/penta/internal/flow"
	"github.com/ruohao1/penta/internal/runtime"
	"github.com/ruohao1/penta/internal/sinks"
)

type Session struct {
	Events <-chan events.Event
	Done   <-chan error
	Stop   func()
}

type StartInput struct {
	Feature flow.Type
	Task    flow.TaskOptions
	Run     runtime.Config
}
type Controller struct {
	runtime *runtime.Runtime
	sink    sinks.Sink
}

func New(rt *runtime.Runtime, sink sinks.Sink) *Controller {
	return &Controller{
		runtime: rt,
		sink:    sink,
	}
}

func (c *Controller) Start(ctx context.Context, in StartInput) (*Session, error) {
	ctx, cancel := context.WithCancel(ctx)
	eventsCh := make(chan events.Event, 256)
	done := make(chan error, 1)
	session := &Session{
		Events: eventsCh,
		Done:   done,
		Stop:   cancel,
	}

	plan, err := featureBuild(ctx, in)
	if err != nil {
		cancel()
		close(eventsCh)
		close(done)
		return nil, err
	}

	runner := c.runtime
	if runner == nil {
		runner = runtime.New(in.Run, c.sink)
	}

	var runSink sinks.Sink = sinks.NewChannelSink(eventsCh)
	if c.sink != nil {
		runSink = sinks.NewMultiSink(c.sink, runSink)
	}

	go func() {
		err := runner.Run(ctx, plan, runSink)
		_ = runSink.Close()
		done <- err
		close(done)
		close(eventsCh)
	}()

	return session, nil
}

func featureBuild(ctx context.Context, in StartInput) (flow.Plan, error) {
	var feature features.Feature
	switch in.Feature {
	case flow.ContentDiscovery:
		feature = contentdiscovery.New()
	default:
		return flow.Plan{}, errors.New("unsupported feature")
	}

	return feature.Build(ctx, flow.BuildInput{Task: in.Task})
}
