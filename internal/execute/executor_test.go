package execute

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	seedtarget "github.com/ruohao1/penta/internal/actions/seed_target"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestExecutorDerivesProbeHTTPFromSeedTargetEvidence(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_1", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON := mustMarshalTestJSON(t, seedtarget.Input{Raw: "1.2.3.4"})
	seedTask := sqlite.Task{ID: "task_seed", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: inputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("create seed task: %v", err)
	}

	sink := &events.SQLiteSink{DB: db}
	executor := &Executor{DB: db, RunID: run.ID, Events: sink}
	if err := executor.RunTask(ctx, seedTask.ID); err != nil {
		t.Fatalf("run seed task: %v", err)
	}

	tasks, err := db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("unexpected task count: got %d want 2", len(tasks))
	}

	probeTask := findTaskByAction(t, tasks, actions.ActionProbeHTTP)
	if probeTask.Status != actions.TaskStatusPending {
		t.Fatalf("unexpected probe task status: got %q want %q", probeTask.Status, actions.TaskStatusPending)
	}

	var probeInput probehttp.Input
	if err := json.Unmarshal([]byte(probeTask.InputJSON), &probeInput); err != nil {
		t.Fatalf("unmarshal probe input: %v", err)
	}
	if probeInput.Value != "1.2.3.4" || probeInput.Type != "ip" {
		t.Fatalf("unexpected probe input: %+v", probeInput)
	}
}

func TestExecutorSkipsDuplicateDerivedTask(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_1", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	seedInputJSON := mustMarshalTestJSON(t, seedtarget.Input{Raw: "1.2.3.4"})
	seedTask := sqlite.Task{ID: "task_seed", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: seedInputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("create seed task: %v", err)
	}

	probeInputJSON := mustMarshalTestJSON(t, probehttp.Input{Value: "1.2.3.4", Type: "ip"})
	existingProbeTask := sqlite.Task{ID: "task_probe", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: probeInputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, existingProbeTask); err != nil {
		t.Fatalf("create existing probe task: %v", err)
	}

	sink := &events.SQLiteSink{DB: db}
	executor := &Executor{DB: db, RunID: run.ID, Events: sink}
	if err := executor.RunTask(ctx, seedTask.ID); err != nil {
		t.Fatalf("run seed task: %v", err)
	}

	tasks, err := db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("unexpected task count: got %d want 2", len(tasks))
	}
}

func TestExecutorSkipsOutOfScopeDerivedTaskForSessionRun(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	now := time.Now()
	session := sqlite.Session{ID: "session_1", Name: "Acme", Kind: sqlite.SessionKindBugBounty, Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := db.CreateScopeRule(ctx, sqlite.ScopeRule{ID: "scope_1", SessionID: session.ID, Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetDomain, Value: "*.example.com", CreatedAt: now}); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
	run := sqlite.Run{ID: "run_1", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: now}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	seedInputJSON := mustMarshalTestJSON(t, seedtarget.Input{Raw: "1.2.3.4"})
	seedTask := sqlite.Task{ID: "task_seed", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: seedInputJSON, Status: actions.TaskStatusPending, CreatedAt: now}
	if err := db.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("create seed task: %v", err)
	}

	sink := &events.SQLiteSink{DB: db}
	executor := &Executor{DB: db, RunID: run.ID, Events: sink}
	if err := executor.RunTask(ctx, seedTask.ID); err != nil {
		t.Fatalf("run seed task: %v", err)
	}
	tasks, err := db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("unexpected task count after blocked derivation: got %d want 1: %+v", len(tasks), tasks)
	}
	eventsRows, err := sink.ListByRunSinceSeq(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	blocked := findEventByType(t, eventsRows, events.EventCandidateBlocked)
	var payload events.CandidateBlockedPayload
	if err := json.Unmarshal([]byte(blocked.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal blocked payload: %v", err)
	}
	if payload.ActionType != actions.ActionProbeHTTP || payload.Source != "session_scope" || !strings.Contains(payload.Reason, "not included in session scope") {
		t.Fatalf("unexpected blocked payload: %+v", payload)
	}
}

func TestExecutorAllowsInScopeDerivedTaskForSessionRun(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	now := time.Now()
	session := sqlite.Session{ID: "session_1", Name: "Acme", Kind: sqlite.SessionKindBugBounty, Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := db.CreateScopeRule(ctx, sqlite.ScopeRule{ID: "scope_1", SessionID: session.ID, Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetIP, Value: "1.2.3.4", CreatedAt: now}); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
	run := sqlite.Run{ID: "run_1", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: now}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	seedInputJSON := mustMarshalTestJSON(t, seedtarget.Input{Raw: "1.2.3.4"})
	seedTask := sqlite.Task{ID: "task_seed", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: seedInputJSON, Status: actions.TaskStatusPending, CreatedAt: now}
	if err := db.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("create seed task: %v", err)
	}

	sink := &events.SQLiteSink{DB: db}
	executor := &Executor{DB: db, RunID: run.ID, Events: sink}
	if err := executor.RunTask(ctx, seedTask.ID); err != nil {
		t.Fatalf("run seed task: %v", err)
	}
	tasks, err := db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("unexpected task count after allowed derivation: got %d want 2: %+v", len(tasks), tasks)
	}
	eventsRows, err := sink.ListByRunSinceSeq(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if hasEventType(eventsRows, events.EventCandidateBlocked) {
		t.Fatalf("allowed derivation emitted blocked event: %+v", eventsRows)
	}
}

func openExecutorTestDB(t *testing.T) *sqlite.DB {
	t.Helper()

	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustMarshalTestJSON(t *testing.T, v any) string {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(data)
}

func findTaskByAction(t *testing.T, tasks []sqlite.Task, actionType actions.ActionType) sqlite.Task {
	t.Helper()

	for _, task := range tasks {
		if task.ActionType == actionType {
			return task
		}
	}
	t.Fatalf("task action %q not found in %+v", actionType, tasks)
	return sqlite.Task{}
}

func findEventByType(t *testing.T, eventsRows []events.Event, eventType events.EventType) events.Event {
	t.Helper()
	for _, event := range eventsRows {
		if event.EventType == eventType {
			return event
		}
	}
	t.Fatalf("event type %q not found in %+v", eventType, eventsRows)
	return events.Event{}
}

func hasEventType(eventsRows []events.Event, eventType events.EventType) bool {
	for _, event := range eventsRows {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
