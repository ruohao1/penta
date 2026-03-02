package controller

import (
	"context"
	"fmt"

	"github.com/Ruohao1/penta/internal/core/engine"
	"github.com/Ruohao1/penta/internal/core/events"
	"github.com/Ruohao1/penta/internal/core/runner"
	"github.com/Ruohao1/penta/internal/core/sinks"
	"github.com/Ruohao1/penta/internal/core/stages"
	"github.com/Ruohao1/penta/internal/core/stages/host_discovery"
	"github.com/Ruohao1/penta/internal/core/tasks"
	"github.com/Ruohao1/penta/internal/core/types"
)

type Session struct {
	Events <-chan events.Event
	Done   <-chan error
	Stop   func()
}

type Controller struct {
	Pool func(types.RunOptions) runner.Pool
}

func New(poolFn func(types.RunOptions) runner.Pool) *Controller {
	return &Controller{Pool: poolFn}
}

func (c *Controller) Start(ctx context.Context, task tasks.Task, opts types.RunOptions, base sinks.Sink) (*Session, error) {
	ctx, cancel := context.WithCancel(ctx)
	eventsCh := make(chan events.Event, 256)
	done := make(chan error, 1)

	if c.Pool == nil {
		c.Pool = engine.DefaultPool
	}

	stageList, err := c.stagesForTask(task)
	if err != nil {
		cancel()
		return nil, err
	}

	var sink sinks.Sink = base
	if sink == nil {
		sink = sinks.NewPentaSink(sinks.SinkOptions{})
	}
	sink = sinks.NewMultiSink(sink, sinks.NewChannelSink(eventsCh))

	eng := engine.Engine{
		Stages: stageList,
		Pool:   c.Pool,
		Sink:   sink,
	}

	go func() {
		err := eng.Run(ctx, task, opts)
		done <- err
		close(done)
		close(eventsCh)
	}()

	return &Session{
		Events: eventsCh,
		Done:   done,
		Stop:   cancel,
	}, nil
}

func (c *Controller) stagesForTask(task tasks.Task) ([]stages.Stage, error) {
	switch task.Type {
	case tasks.HostDiscovery:
		return []stages.Stage{host_discovery.New()}, nil
	case tasks.PortScan:
		return nil, fmt.Errorf("port scan stages not implemented")
	case tasks.WebProbe:
		return nil, fmt.Errorf("web probe stages not implemented")
	default:
		return nil, fmt.Errorf("unknown task type %q", task.Type)
	}
}
