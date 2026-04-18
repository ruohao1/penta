package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "penta.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestOpenInitializesSchema(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	run := Run{
		ID:        "run_1",
		Mode:      "bugbounty",
		Status:    "running",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	got, err := db.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}

	if got.ID != run.ID || got.Mode != run.Mode || got.Status != run.Status {
		t.Fatalf("unexpected run: %+v", got)
	}

	if !got.CreatedAt.Equal(run.CreatedAt) {
		t.Fatalf("unexpected created_at: got %v want %v", got.CreatedAt, run.CreatedAt)
	}
}

func TestTaskArtifactAndEvidenceCRUD(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	run := Run{
		ID:        "run_1",
		Mode:      "ctf",
		Status:    "running",
		CreatedAt: now,
	}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	task := Task{
		ID:         "task_1",
		RunID:      run.ID,
		ActionType: "probe_http",
		InputJSON:  `{"host":"example.com"}`,
		Status:     "pending",
		CreatedAt:  now,
	}
	if err := db.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	tasks, err := db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("unexpected task count: got %d want 1", len(tasks))
	}
	if tasks[0].ID != task.ID || tasks[0].Status != task.Status {
		t.Fatalf("unexpected task: %+v", tasks[0])
	}

	if err := db.UpdateTaskStatus(ctx, task.ID, "done"); err != nil {
		t.Fatalf("update task status: %v", err)
	}

	tasks, err = db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks after update: %v", err)
	}
	if tasks[0].Status != "done" {
		t.Fatalf("unexpected updated status: got %q want %q", tasks[0].Status, "done")
	}

	artifact := Artifact{
		ID:        "art_1",
		TaskID:    task.ID,
		Path:      "/tmp/httpx.json",
		CreatedAt: now,
	}
	if err := db.CreateArtifact(ctx, artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	artifacts, err := db.ListArtifactsByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("unexpected artifact count: got %d want 1", len(artifacts))
	}
	if artifacts[0].ID != artifact.ID || artifacts[0].Path != artifact.Path {
		t.Fatalf("unexpected artifact: %+v", artifacts[0])
	}

	evidence := Evidence{
		ID:        "ev_1",
		RunID:     run.ID,
		Kind:      "service",
		DataJSON:  `{"host":"example.com","port":443}`,
		CreatedAt: now,
	}
	if err := db.CreateEvidence(ctx, evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}

	evidenceRows, err := db.ListEvidenceByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidenceRows) != 1 {
		t.Fatalf("unexpected evidence count: got %d want 1", len(evidenceRows))
	}
	if evidenceRows[0].ID != evidence.ID || evidenceRows[0].Kind != evidence.Kind || evidenceRows[0].DataJSON != evidence.DataJSON {
		t.Fatalf("unexpected evidence: %+v", evidenceRows[0])
	}
	if !evidenceRows[0].CreatedAt.Equal(evidence.CreatedAt) {
		t.Fatalf("unexpected evidence created_at: got %v want %v", evidenceRows[0].CreatedAt, evidence.CreatedAt)
	}
}

func TestCreateTaskRejectsInvalidJSON(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	run := Run{
		ID:        "run_1",
		Mode:      "ctf",
		Status:    "running",
		CreatedAt: now,
	}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	err := db.CreateTask(ctx, Task{
		ID:         "task_bad",
		RunID:      run.ID,
		ActionType: "probe_http",
		InputJSON:  `{"target":`,
		Status:     "pending",
		CreatedAt:  now,
	})
	if err == nil {
		t.Fatal("expected invalid JSON to be rejected")
	}
}
