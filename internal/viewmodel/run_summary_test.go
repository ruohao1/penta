package viewmodel

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestBuildRunSummaryCountsTasksAndEvidence(t *testing.T) {
	db := openViewModelTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_1", Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	tasks := []sqlite.Task{
		{ID: "task_1", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: `{}`, Status: actions.TaskStatusCompleted, CreatedAt: time.Now()},
		{ID: "task_2", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: `{}`, Status: actions.TaskStatusCompleted, CreatedAt: time.Now()},
		{ID: "task_3", RunID: run.ID, ActionType: actions.ActionResolveDNS, InputJSON: `{}`, Status: actions.TaskStatusFailed, CreatedAt: time.Now()},
	}
	for _, task := range tasks {
		if err := db.CreateTask(ctx, task); err != nil {
			t.Fatalf("create task %s: %v", task.ID, err)
		}
	}

	evidenceRows := []sqlite.Evidence{
		{ID: "evidence_1", RunID: run.ID, TaskID: "task_1", Kind: "target", DataJSON: `{"value":"example.com","type":"domain"}`, CreatedAt: time.Now()},
		{ID: "evidence_2", RunID: run.ID, TaskID: "task_2", Kind: "service", DataJSON: `{"scheme":"https","host":"example.com","port":443}`, CreatedAt: time.Now()},
		{ID: "evidence_3", RunID: run.ID, TaskID: "task_2", Kind: "service", DataJSON: `{"scheme":"http","host":"example.com","port":80}`, CreatedAt: time.Now()},
	}
	for _, evidence := range evidenceRows {
		if err := db.CreateEvidence(ctx, evidence); err != nil {
			t.Fatalf("create evidence %s: %v", evidence.ID, err)
		}
	}

	summary, err := BuildRunSummary(ctx, db, run.ID, "/tmp/penta.db")
	if err != nil {
		t.Fatalf("build summary: %v", err)
	}
	if summary.RunID != run.ID || summary.Status != actions.RunStatusCompleted || summary.DBPath != "/tmp/penta.db" {
		t.Fatalf("unexpected summary identity: %+v", summary)
	}
	if summary.TaskCounts[actions.TaskStatusCompleted] != 2 || summary.TaskCounts[actions.TaskStatusFailed] != 1 {
		t.Fatalf("unexpected task counts: %+v", summary.TaskCounts)
	}
	if summary.EvidenceCounts["target"] != 1 || summary.EvidenceCounts["service"] != 2 {
		t.Fatalf("unexpected evidence counts: %+v", summary.EvidenceCounts)
	}
	if len(summary.Evidence) != 3 {
		t.Fatalf("unexpected evidence summary count: got %d want 3", len(summary.Evidence))
	}
	if summary.Evidence[0].Label != "domain example.com" || summary.Evidence[1].Label != "https example.com:443" {
		t.Fatalf("unexpected evidence summaries: %+v", summary.Evidence)
	}
}

func openViewModelTestDB(t *testing.T) *sqlite.DB {
	t.Helper()

	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
