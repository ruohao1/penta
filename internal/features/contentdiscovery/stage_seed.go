package contentdiscovery

import (
	"context"
	"fmt"

	"github.com/ruohao1/penta/internal/flow"
	"github.com/ruohao1/penta/internal/targets"
)

type SeedPayload struct {
	RawTarget string
	Depth     int
}


type SeedStage struct {
	workers int
}

func NewSeedStage(workers int) *SeedStage {
	if workers <= 0 {
		workers = 1
	}
	return &SeedStage{workers: workers}
}

func (s *SeedStage) Name() string { return "content.seed" }

func (s *SeedStage) Workers() int { return s.workers }

func (s *SeedStage) Process(ctx context.Context, in flow.Item) ([]flow.Item, error) {
	_ = ctx
	p, ok := in.Payload.(SeedPayload)
	if !ok {
		ptr, okPtr := in.Payload.(*SeedPayload)
		if !okPtr || ptr == nil {
			return nil, fmt.Errorf("content seed: invalid payload type %T", in.Payload)
		}
		p = *ptr
	}

	parsed, err := targets.ParseOne(p.RawTarget)
	if err != nil {
		return nil, err
	}

	endpoints, err := expandTargetToEndpoints(parsed)
	if err != nil {
		return nil, err
	}

	out := make([]flow.Item, 0, len(endpoints))
	for _, endpoint := range endpoints {
		item := in
		item.Stage = "content.discover"
		item.Target = endpoint
		item.Key = endpoint
		item.Payload = DiscoverPayload{
			Endpoint: endpoint,
			Depth:    p.Depth,
		}
		out = append(out, item)
	}

	return out, nil
}

func expandTargetToEndpoints(t targets.Target) ([]string, error) {
	switch t.Kind {
	case targets.KindIP:
		if err := t.AssertKind(targets.KindIP); err != nil {
			return nil, err
		}
		return []string{"http://" + t.IP.String() + "/"}, nil
	case targets.KindURL:
		if err := t.AssertKind(targets.KindURL); err != nil {
			return nil, err
		}
		u := *t.URL
		if u.Path == "" {
			u.Path = "/"
		}
		return []string{u.String()}, nil
	case targets.KindCIDR:
		if err := t.AssertKind(targets.KindCIDR); err != nil {
			return nil, err
		}
		prefix := t.CIDR.Masked()
		out := make([]string, 0)
		for ip := prefix.Addr(); ip.IsValid() && prefix.Contains(ip); ip = ip.Next() {
			out = append(out, "http://"+ip.String()+"/")
		}
		return out, nil
	default:
		return nil, fmt.Errorf("content seed: unsupported target kind %q", t.Kind)
	}
}
