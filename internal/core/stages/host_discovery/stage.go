package host_discovery

import (
	"context"
	"strings"

	"github.com/Ruohao1/penta/internal/checks"
	"github.com/Ruohao1/penta/internal/checks/tcpconnect"
	"github.com/Ruohao1/penta/internal/core/runner"
	"github.com/Ruohao1/penta/internal/core/sinks"
	"github.com/Ruohao1/penta/internal/core/tasks"
	"github.com/Ruohao1/penta/internal/core/types"
)

type ProbeSet struct {
	ARP        checks.Checker
	ICMP       checks.Checker
	TCPSYN     checks.Checker
	TCPConnect checks.Checker
}

type stage struct {
	Probes ProbeSet
}

func New() stage {
	tcpconnect := tcpconnect.New()
	return stage{
		Probes: ProbeSet{
			TCPConnect: &tcpconnect,
		},
	}
}

func (s stage) Name() string { return "host_discovery" }

func (s stage) Build(ctx context.Context, task tasks.Task, opts types.RunOptions, sink sinks.Sink) ([]runner.Job, error) {
	// 1) expand targets
	jobs := []runner.Job{}
	targets, err := task.ExpandAllTargetsExpr()
	if err != nil {
		return jobs, err
	}

	checker := s.pick(opts)

	for _, t := range targets {
		hostKey := strings.Split(t, ":")[0]
		job := runner.CheckJob{
			StageName: s.Name(),
			HostKey:   hostKey,
			Checker:   checker,
			Sink:      sink,
		}
		if checker == s.Probes.TCPConnect {
			for _, p := range task.Ports {
				endpoint := types.NewEndpointNet(t, p)
				input := tcpconnect.Input{Endpoint: endpoint, Opts: opts}
				job.Input = input
				jobs = append(jobs, job)
			}
		} else {
			_ = t
			job = runner.CheckJob{}
		}

	}

	// 2) build jobs host×ports
	return jobs, nil
}

func (s stage) After(ctx context.Context, task tasks.Task, opts types.RunOptions, sink sinks.Sink) error {
	return nil
}

func (s stage) pick(opts types.RunOptions) checks.Checker {
	if opts.ProbeOpts.ICMP && opts.Privileged && s.Probes.ICMP != nil {
		return s.Probes.ICMP
	}
	if opts.ProbeOpts.TCP && opts.Privileged && s.Probes.TCPSYN != nil {
		return s.Probes.TCPSYN
	}
	return s.Probes.TCPConnect
}
