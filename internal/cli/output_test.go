package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
)

func TestVerbosityFromFlags(t *testing.T) {
	tests := []struct {
		name         string
		quiet        bool
		verboseCount int
		want         Verbosity
	}{
		{name: "default normal", want: VerbosityNormal},
		{name: "verbose", verboseCount: 1, want: VerbosityVerbose},
		{name: "debug", verboseCount: 2, want: VerbosityDebug},
		{name: "trace", verboseCount: 3, want: VerbosityTrace},
		{name: "trace clamps", verboseCount: 9, want: VerbosityTrace},
		{name: "quiet overrides verbose", quiet: true, verboseCount: 3, want: VerbosityQuiet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verbosityFromFlags(tt.quiet, tt.verboseCount); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestStdoutReporterQuietSuppressesLiveEvents(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityQuiet)
	reporter.RunStarted("run_1", "example.com")
	reporter.Event(taskStartedEvent(actions.ActionSeedTarget))
	reporter.RunCompleted("run_1")

	got := out.String()
	if strings.Contains(got, "Discovering target") || strings.Contains(got, "running") {
		t.Fatalf("quiet output included live events: %q", got)
	}
	if !strings.Contains(got, "Recon completed: run_1") {
		t.Fatalf("quiet output missing completion: %q", got)
	}
}

func TestStdoutReporterNormalPrintsDedupedPhases(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityNormal)
	reporter.Event(taskStartedEvent(actions.ActionSeedTarget))
	reporter.Event(taskStartedEvent(actions.ActionSeedTarget))
	reporter.Event(taskStartedEvent(actions.ActionProbeHTTP))

	got := out.String()
	if count := strings.Count(got, "Discovering target..."); count != 1 {
		t.Fatalf("expected one discovering phase, got %d in %q", count, got)
	}
	if !strings.Contains(got, "Checking services...") {
		t.Fatalf("normal output missing service phase: %q", got)
	}
	if strings.Contains(got, "payload=") {
		t.Fatalf("normal output should not contain payloads: %q", got)
	}
}

func TestStdoutReporterVerbosePrintsLifecycle(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityVerbose)
	reporter.Event(taskStartedEvent(actions.ActionProbeHTTP))
	reporter.Event(evidenceCreatedEvent("service"))

	got := out.String()
	if !strings.Contains(got, "running  probe_http") {
		t.Fatalf("verbose output missing task lifecycle: %q", got)
	}
	if !strings.Contains(got, "evidence service") {
		t.Fatalf("verbose output missing evidence lifecycle: %q", got)
	}
}

func TestStdoutReporterTracePrintsPayload(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityTrace)
	reporter.Event(taskStartedEvent(actions.ActionProbeHTTP))

	got := out.String()
	if !strings.Contains(got, "event task.started") || !strings.Contains(got, "payload=") {
		t.Fatalf("trace output missing event payload: %q", got)
	}
}

func taskStartedEvent(actionType actions.ActionType) events.Event {
	return events.Event{
		EventType:   events.EventTaskStarted,
		EntityKind:  events.EntityTask,
		EntityID:    "task_1",
		PayloadJSON: mustPayloadJSON(events.TaskStartedPayload{ActionType: actionType}),
		CreatedAt:   time.Now(),
	}
}

func evidenceCreatedEvent(kind string) events.Event {
	return events.Event{
		EventType:   events.EventEvidenceCreated,
		EntityKind:  events.EntityEvidence,
		EntityID:    "evidence_1",
		PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: kind}),
		CreatedAt:   time.Now(),
	}
}
