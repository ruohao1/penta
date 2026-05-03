package execute

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	seedtarget "github.com/ruohao1/penta/internal/actions/seed_target"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestExecutorDerivesProbeHTTPFromSeedTargetEvidence(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_1", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON := mustMarshalTestJSON(t, seedtarget.Input{Raw: "example.com"})
	seedTask := sqlite.Task{ID: "task_seed", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: inputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("create seed task: %v", err)
	}

	executor := &Executor{DB: db, RunID: run.ID}
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
	if probeInput.Value != "example.com" || probeInput.Type != "domain" {
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

	seedInputJSON := mustMarshalTestJSON(t, seedtarget.Input{Raw: "example.com"})
	seedTask := sqlite.Task{ID: "task_seed", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: seedInputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("create seed task: %v", err)
	}

	probeInputJSON := mustMarshalTestJSON(t, probehttp.Input{Value: "example.com", Type: "domain"})
	existingProbeTask := sqlite.Task{ID: "task_probe", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: probeInputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, existingProbeTask); err != nil {
		t.Fatalf("create existing probe task: %v", err)
	}

	executor := &Executor{DB: db, RunID: run.ID}
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
