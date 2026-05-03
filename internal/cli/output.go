package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
)

type Verbosity int

const (
	VerbosityQuiet Verbosity = iota
	VerbosityNormal
	VerbosityVerbose
	VerbosityDebug
	VerbosityTrace
)

func verbosityFromFlags(quiet bool, verboseCount int) Verbosity {
	if quiet {
		return VerbosityQuiet
	}
	switch {
	case verboseCount <= 0:
		return VerbosityNormal
	case verboseCount == 1:
		return VerbosityVerbose
	case verboseCount == 2:
		return VerbosityDebug
	default:
		return VerbosityTrace
	}
}

type RunReporter interface {
	RunStarted(runID, target string)
	Event(evt events.Event)
	RunCompleted(runID string)
	RunFailed(runID string, err error)
}

type stdoutReporter struct {
	out           io.Writer
	verbosity     Verbosity
	startedAt     time.Time
	printedPhases map[string]bool
	styles        cliStyles
}

func newStdoutReporter(out io.Writer, verbosity Verbosity, color bool) *stdoutReporter {
	return &stdoutReporter{out: out, verbosity: verbosity, startedAt: time.Now(), printedPhases: map[string]bool{}, styles: newCLIStyles(color)}
}

func (r *stdoutReporter) RunStarted(runID, target string) {
	if r.verbosity == VerbosityQuiet {
		return
	}
	fprintf(r.out, "%s\nRun     %s\nTarget  %s\n\n", r.styles.heading.Render("Recon started"), runID, target)
}

func (r *stdoutReporter) Event(evt events.Event) {
	switch r.verbosity {
	case VerbosityQuiet:
		return
	case VerbosityNormal:
		r.renderPhase(evt)
	case VerbosityVerbose:
		r.renderVerbose(evt)
	case VerbosityDebug:
		r.renderDebug(evt, false)
	default:
		r.renderDebug(evt, true)
	}
}

func (r *stdoutReporter) RunCompleted(runID string) {
	if r.verbosity == VerbosityQuiet {
		fprintf(r.out, "Recon completed: %s\n", runID)
		return
	}
	fprintf(r.out, "%s\n", r.styles.success.Render("Recon completed"))
}

func (r *stdoutReporter) RunFailed(runID string, err error) {
	if r.verbosity == VerbosityQuiet {
		fprintf(r.out, "Recon failed: %s: %v\n", runID, err)
		return
	}
	fprintf(r.out, "%s\n%s\n", r.styles.failure.Render("Recon failed"), err)
}

func (r *stdoutReporter) renderPhase(evt events.Event) {
	phase, ok := phaseForEvent(evt)
	if !ok || r.printedPhases[phase] {
		return
	}
	r.printedPhases[phase] = true
	fprintf(r.out, "%s\n", r.styles.phase.Render(phase+"..."))
}

func phaseForEvent(evt events.Event) (string, bool) {
	if evt.EventType != events.EventTaskStarted {
		return "", false
	}
	actionType, ok := actionTypeFromPayload(evt.PayloadJSON)
	if !ok {
		return "", false
	}
	switch actionType {
	case actions.ActionSeedTarget:
		return "Discovering target", true
	case actions.ActionResolveDNS:
		return "Resolving DNS", true
	case actions.ActionProbeHTTP:
		return "Checking services", true
	default:
		return "Running " + string(actionType), true
	}
}

func (r *stdoutReporter) renderVerbose(evt events.Event) {
	label, detail, ok := compactEvent(evt)
	if !ok {
		return
	}
	fprintf(r.out, "[%s] %-8s %s\n", r.elapsed(), r.styles.label(label), detail)
}

func (r *stdoutReporter) renderDebug(evt events.Event, includePayload bool) {
	line := fmt.Sprintf("[%s] event %s entity=%s id=%s", r.elapsed(), evt.EventType, evt.EntityKind, evt.EntityID)
	if includePayload && evt.PayloadJSON != "" {
		line += " payload=" + evt.PayloadJSON
	}
	fprintf(r.out, "%s\n", r.styles.debug.Render(line))
}

func compactEvent(evt events.Event) (string, string, bool) {
	switch evt.EventType {
	case events.EventTaskEnqueued:
		actionType, ok := actionTypeFromPayload(evt.PayloadJSON)
		return "queued", string(actionType), ok
	case events.EventTaskStarted:
		actionType, ok := actionTypeFromPayload(evt.PayloadJSON)
		return "running", string(actionType), ok
	case events.EventTaskCompleted:
		actionType, ok := actionTypeFromPayload(evt.PayloadJSON)
		return "done", string(actionType), ok
	case events.EventTaskFailed:
		actionType, ok := actionTypeFromPayload(evt.PayloadJSON)
		return "failed", string(actionType), ok
	case events.EventEvidenceCreated:
		kind, ok := evidenceKindFromPayload(evt.PayloadJSON)
		return "evidence", kind, ok
	default:
		return "", "", false
	}
}

func actionTypeFromPayload(payload string) (actions.ActionType, bool) {
	var data struct {
		ActionType actions.ActionType `json:"action_type"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil || data.ActionType == "" {
		return "", false
	}
	return data.ActionType, true
}

func evidenceKindFromPayload(payload string) (string, bool) {
	var data struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil || data.Kind == "" {
		return "", false
	}
	return data.Kind, true
}

func (r *stdoutReporter) elapsed() string {
	elapsed := time.Since(r.startedAt).Truncate(time.Second)
	minutes := int(elapsed.Minutes())
	seconds := int(elapsed.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

type reportingSink struct {
	inner    events.Sink
	reporter RunReporter
}

func (s reportingSink) Append(ctx context.Context, evt events.Event) error {
	if err := s.inner.Append(ctx, evt); err != nil {
		return err
	}
	if s.reporter != nil {
		s.reporter.Event(evt)
	}
	return nil
}

func (s reportingSink) ListByRunSinceSeq(ctx context.Context, runID string, seq int64, limit int) ([]events.Event, error) {
	return s.inner.ListByRunSinceSeq(ctx, runID, seq, limit)
}

func fprintf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format, args...)
}
