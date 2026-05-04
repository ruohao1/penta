package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/viewmodel"
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
	reporter := newStdoutReporter(&out, VerbosityQuiet, false)
	reporter.RunStarted("run_1", "example.com")
	reporter.Event(taskStartedEvent(actions.ActionSeedTarget))
	reporter.RunCompleted(&viewmodel.RunSummary{RunID: "run_1"})

	got := out.String()
	if strings.Contains(got, "Discovering target") || strings.Contains(got, "running") {
		t.Fatalf("quiet output included live events: %q", got)
	}
	if !strings.Contains(got, "Recon completed: run_1") {
		t.Fatalf("quiet output missing completion: %q", got)
	}
}

func TestStdoutReporterNormalPrintsRunSummary(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityNormal, false)
	reporter.RunCompleted(&viewmodel.RunSummary{
		RunID:  "run_1",
		Status: actions.RunStatusCompleted,
		DBPath: "/tmp/penta.db",
		TaskCounts: map[actions.TaskStatus]int{
			actions.TaskStatusCompleted: 2,
			actions.TaskStatusFailed:    1,
		},
		EvidenceCounts: map[string]int{
			"target":  1,
			"service": 2,
		},
	})

	got := out.String()
	for _, want := range []string{"Recon completed", "Run        run_1", "Status     completed", "Tasks      2 completed / 1 failed / 0 pending", "Evidence   1 target / 2 service", "Database   /tmp/penta.db"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary output missing %q in %q", want, got)
		}
	}
}

func TestStdoutReporterNormalPrintsDedupedPhases(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityNormal, false)
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

func TestStdoutReporterNormalPrintsDiscoveries(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityNormal, false)
	reporter.Event(evidenceCreatedEventWithLabel("service", "https example.com:443"))

	got := out.String()
	if !strings.Contains(got, "Found service: https example.com:443") {
		t.Fatalf("normal output missing discovery: %q", got)
	}
}

func TestStdoutReporterVerbosePrintsLifecycle(t *testing.T) {
	var out bytes.Buffer
	reporter := newStdoutReporter(&out, VerbosityVerbose, false)
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
	reporter := newStdoutReporter(&out, VerbosityTrace, false)
	reporter.Event(taskStartedEvent(actions.ActionProbeHTTP))

	got := out.String()
	if !strings.Contains(got, "event task.started") || !strings.Contains(got, "payload=") {
		t.Fatalf("trace output missing event payload: %q", got)
	}
}

func TestCLIStylesDisabledRenderPlainText(t *testing.T) {
	styles := newCLIStyles(false)
	if got := styles.success.Render("Recon completed"); got != "Recon completed" {
		t.Fatalf("disabled style rendered %q", got)
	}
	if got := styles.label("running"); got != "running" {
		t.Fatalf("disabled label style rendered %q", got)
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
	return evidenceCreatedEventWithLabel(kind, "")
}

func evidenceCreatedEventWithLabel(kind, label string) events.Event {
	return events.Event{
		EventType:   events.EventEvidenceCreated,
		EntityKind:  events.EntityEvidence,
		EntityID:    "evidence_1",
		PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: kind, Label: label}),
		CreatedAt:   time.Now(),
	}
}
