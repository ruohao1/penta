package tcpconnect

import (
	"context"
	"fmt"
	"time"

	"github.com/Ruohao1/penta/internal/checks"
	"github.com/Ruohao1/penta/internal/core/types"
	"github.com/Ruohao1/penta/internal/netprobe"
)

var _ checks.Checker = (*Checker)(nil)

type Input struct {
	Endpoint types.Endpoint
	Opts     types.RunOptions
}

type Checker struct {
	Dialer netprobe.Dialer
}

func New() Checker {
	return Checker{Dialer: netprobe.NetDialer{}}
}

func (p *Checker) Name() string { return "tcp_connect" }

func (p *Checker) Check() checks.CheckFn {
	return func(ctx context.Context, in any, emit checks.EmitFn) error {
		req, ok := in.(Input)
		if !ok {
			return fmt.Errorf("%s: want %T, got %T", p.Name(), Input{}, in)
		}

		finding := types.Finding{
			ObservedAt: time.Now().UTC(),
			Check:      p.Name(),
			Proto:      types.ProtocolTCP,
			Endpoint:   req.Endpoint,
			Severity:   "info",
			Meta:       map[string]any{},
		}

		if req.Endpoint.Kind != types.EndpointNet {
			finding.Severity = "error"
			finding.Status = "unsupported_endpoint_kind"
			finding.Meta["endpoint_kind"] = req.Endpoint.Kind
			emit(finding)
			return nil
		}

		res := netprobe.TCPConnect(ctx, p.Dialer, req.Endpoint.String(), req.Opts.Timeouts.TCP)
		finding.RTTMs = res.ElapsedMs
		finding.Status = res.State
		finding.Meta["ok"] = res.OK
		finding.Meta["reason"] = res.Reason

		emit(finding)
		return nil
	}
}
