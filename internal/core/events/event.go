package events

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Ruohao1/penta/internal/core/types"
)

// Event is the core event type emitted by the engine.
// It is intentionally typed for in-process efficiency.
type Event struct {
	EmittedAt time.Time `json:"emitted_at"`
	Type      Type      `json:"type"`
	Stage     string    `json:"stage,omitempty"`
	Target    string    `json:"target,omitempty"`

	// Payloads (only one of these is typically non-nil per event)
	Finding   *types.Finding  `json:"finding,omitempty"`
	HostState *HostStateEvent `json:"host_state,omitempty"`
	Progress  *Progress       `json:"progress,omitempty"`

	// For log/error events
	Message string `json:"message,omitempty"`
	Err     string `json:"err,omitempty"`
}

// Type tells the consumer what kind of event this is.
type Type string

const (
	// Findings / results
	EventFinding Type = "finding"

	// Host lifecycle
	EventHostState Type = "host_state"

	// Port / service
	EventPortOpen   Type = "port_open"
	EventPortClosed Type = "port_closed"

	// Probe execution
	EventProbeStart Type = "probe_start"
	EventProbeDone  Type = "probe_done"

	// Scan execution
	EventScanStart Type = "scan_start"
	EventScanDone  Type = "scan_done"

	// Engine lifecycle
	EventEngineStart Type = "engine_start"
	EventEngineStop  Type = "engine_stop"
	EventEngineDone  Type = "engine_done"

	// State / control
	EventIdle Type = "idle"
	EventDone Type = "done"

	// Observability
	EventError   Type = "error"
	EventLog     Type = "log"
	EventUnknown Type = "unknown"
)

type HostStateEvent struct {
	Host   string          `json:"host"`
	State  types.HostState `json:"state"` // up/down
	Via    string          `json:"via,omitempty"`
	Port   int             `json:"port,omitempty"`
	Reason string          `json:"reason,omitempty"`
	Meta   map[string]any  `json:"meta,omitempty"`
}

func NewEvent(t Type) Event {
	return Event{Type: t}
}

func NewEventWithProgress(t Type, total int) Event {
	progress := &Progress{TotalTargets: total}
	return Event{Type: t, Progress: progress}
}

func NewFindingEvent(f *types.Finding) Event {
	return Event{Type: EventFinding, Finding: f}
}

// Progress is for high-level progress reporting (TUI / verbose mode).
type Progress struct {
	TotalTargets   int `json:"total_targets,omitempty"`
	ProcessedHosts int `json:"processed_hosts,omitempty"`
	ActiveHosts    int `json:"active_hosts,omitempty"`

	// Optional fine-grained metrics
	TotalFindings int     `json:"total_findings,omitempty"`
	Percent       float64 `json:"percent,omitempty"`
}

func (ev Event) String() string {
	ts := ev.EmittedAt
	if ts.IsZero() {
		ts = time.Now()
	}

	parts := []string{
		ts.Format(time.RFC3339),
		string(ev.Type),
	}

	if ev.Stage != "" {
		parts = append(parts, "stage="+ev.Stage)
	}
	if ev.Target != "" {
		parts = append(parts, "target="+ev.Target)
	}

	switch ev.Type {
	case EventFinding:
		if ev.Finding != nil {
			parts = append(parts, findingSummary(*ev.Finding)...)
		}

	case EventHostState:
		if ev.HostState != nil {
			hs := ev.HostState
			if hs.Host != "" {
				parts = append(parts, "host="+hs.Host)
			}
			parts = append(parts, "state="+string(hs.State))
			if hs.Via != "" {
				parts = append(parts, "via="+hs.Via)
			}
			if hs.Port != 0 {
				parts = append(parts, fmt.Sprintf("port=%d", hs.Port))
			}
			if hs.Reason != "" {
				parts = append(parts, "reason="+short(hs.Reason, 80))
			}
		}

	case EventError:
		if ev.Err != "" {
			parts = append(parts, "err="+short(ev.Err, 140))
		} else if ev.Message != "" {
			parts = append(parts, "err="+short(ev.Message, 140))
		}

	case EventLog:
		if ev.Message != "" {
			parts = append(parts, "msg="+short(ev.Message, 140))
		}
		if ev.Err != "" {
			parts = append(parts, "err="+short(ev.Err, 140))
		}

	default:
		if ev.Progress != nil {
			parts = append(parts, progressSummary(*ev.Progress)...)
		}
		if ev.Message != "" {
			parts = append(parts, "msg="+short(ev.Message, 140))
		}
		if ev.Err != "" {
			parts = append(parts, "err="+short(ev.Err, 140))
		}
	}

	return strings.Join(parts, " ")
}

func findingSummary(f types.Finding) []string {
	out := make([]string, 0, 10)

	if !f.ObservedAt.IsZero() {
		out = append(out, "obs="+f.ObservedAt.Format(time.RFC3339))
	}

	if f.Check != "" {
		out = append(out, "check="+f.Check)
	}

	if string(f.Proto) != "" {
		out = append(out, "proto="+string(f.Proto))
	}

	if !f.Endpoint.IsZero() {
		if ep := f.Endpoint.String(); ep != "" {
			out = append(out, "ep="+ep)
		}
	}

	if f.Status != "" {
		out = append(out, "status="+f.Status)
	}

	if f.Severity != "" {
		out = append(out, "sev="+f.Severity)
	}

	if f.RTTMs > 0 {
		out = append(out, fmt.Sprintf("rtt=%.2fms", f.RTTMs))
	}

	if len(f.Meta) > 0 {
		out = append(out, "meta="+metaHint(f.Meta, 3))
	}

	return out
}

func progressSummary(p Progress) []string {
	out := make([]string, 0, 6)
	if p.TotalTargets != 0 {
		out = append(out, fmt.Sprintf("total=%d", p.TotalTargets))
	}
	if p.ProcessedHosts != 0 {
		out = append(out, fmt.Sprintf("done=%d", p.ProcessedHosts))
	}
	if p.ActiveHosts != 0 {
		out = append(out, fmt.Sprintf("active=%d", p.ActiveHosts))
	}
	if p.TotalFindings != 0 {
		out = append(out, fmt.Sprintf("findings=%d", p.TotalFindings))
	}
	if p.Percent != 0 {
		out = append(out, fmt.Sprintf("pct=%.1f", p.Percent))
	}
	return out
}

func metaHint(m map[string]any, maxKeys int) string {
	if maxKeys <= 0 {
		return "{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > maxKeys {
		keys = keys[:maxKeys]
	}
	return "{" + strings.Join(keys, ",") + "}"
}

func short(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
