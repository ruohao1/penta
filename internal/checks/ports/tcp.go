package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/Ruohao1/penta/internal/core/types"
	"github.com/Ruohao1/penta/internal/netprobe"
)

type TCPScanner struct {
	Dialer netprobe.Dialer
}

func (s *TCPScanner) ScanPort(ctx context.Context, host *types.Host, port *types.Port, opts types.RunOptions) (types.Finding, error) {
	addr := fmt.Sprintf("%s:%d", host.Address(), port.Number)
	res := netprobe.TCPConnect(ctx, s.Dialer, addr, opts.Timeouts.TCP)

	f := types.Finding{
		ObservedAt: time.Now().UTC(),
		Check:      "tcp_port_scan",
		Proto:      types.ProtocolTCP,
		Endpoint:   types.NewEndpointNet(host.Address(), port.Number),
		RTTMs:      res.ElapsedMs,
		Status:     res.State,
		Meta:       map[string]any{"reason": res.Reason},
		Severity:   "info",
	}

	// You decide your semantics:
	// open => open
	// refused => closed (or “reachable but closed”)
	return f, nil
}
