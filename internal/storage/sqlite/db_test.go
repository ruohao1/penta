package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	_ "modernc.org/sqlite"
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
		Status:    actions.RunStatusRunning,
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

func TestListRunsAndLatestRunOrderByNewestFirst(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	older := Run{ID: "run_old", Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now.Add(-time.Hour)}
	newer := Run{ID: "run_new", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: now}

	if err := db.CreateRun(ctx, older); err != nil {
		t.Fatalf("create older run: %v", err)
	}
	if err := db.CreateRun(ctx, newer); err != nil {
		t.Fatalf("create newer run: %v", err)
	}

	runs, err := db.ListRuns(ctx)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 || runs[0].ID != newer.ID || runs[1].ID != older.ID {
		t.Fatalf("unexpected run order: %+v", runs)
	}
	latest, err := db.LatestRun(ctx)
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if latest.ID != newer.ID {
		t.Fatalf("unexpected latest run: %+v", latest)
	}
}

func TestLatestRunReturnsNoRowsWhenEmpty(t *testing.T) {
	db := openTestDB(t)
	_, err := db.LatestRun(context.Background())
	if err != sql.ErrNoRows {
		t.Fatalf("latest empty error: got %v want sql.ErrNoRows", err)
	}
}

func TestOpenSetsSchemaVersion(t *testing.T) {
	db := openTestDB(t)

	var version int
	if err := db.QueryRowContext(context.Background(), `PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("unexpected schema version: got %d want %d", version, currentSchemaVersion)
	}
}

func TestOpenMigratesLegacySchemaAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, schemaSQL); err != nil {
		_ = rawDB.Close()
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close migrated db: %v", err)
	}

	db, err = Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("reopen migrated db: %v", err)
	}
	defer func() { _ = db.Close() }()

	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("unexpected schema version after reopen: got %d want %d", version, currentSchemaVersion)
	}
	if ok, err := db.columnExists(ctx, "runs", "session_id"); err != nil {
		t.Fatalf("inspect runs.session_id: %v", err)
	} else if !ok {
		t.Fatal("legacy migration did not add runs.session_id")
	}
}

func TestOpenRejectsNewerSchemaVersion(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "future.db")
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, `PRAGMA user_version = 999`); err != nil {
		_ = rawDB.Close()
		t.Fatalf("set future schema version: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	_, err = Open(ctx, dbPath)
	if err == nil {
		t.Fatal("expected newer schema version to fail")
	}
}

func TestTaskArtifactAndEvidenceCRUD(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	run := Run{
		ID:        "run_1",
		Mode:      "ctf",
		Status:    actions.RunStatusRunning,
		CreatedAt: now,
	}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	task := Task{
		ID:         "task_1",
		RunID:      run.ID,
		ActionType: actions.ActionType("probe_http"),
		InputJSON:  `{"host":"example.com"}`,
		Status:     actions.TaskStatusPending,
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

	gotTask, err := db.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if gotTask.ID != task.ID || gotTask.RunID != task.RunID || gotTask.ActionType != task.ActionType || gotTask.Status != task.Status || gotTask.InputJSON != task.InputJSON {
		t.Fatalf("unexpected get task result: %+v", gotTask)
	}
	if !gotTask.CreatedAt.Equal(task.CreatedAt) {
		t.Fatalf("unexpected task created_at: got %v want %v", gotTask.CreatedAt, task.CreatedAt)
	}

	if err := db.UpdateTaskStatus(ctx, task.ID, actions.TaskStatusCompleted); err != nil {
		t.Fatalf("update task status: %v", err)
	}

	tasks, err = db.ListTasksByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list tasks after update: %v", err)
	}
	if tasks[0].Status != actions.TaskStatusCompleted {
		t.Fatalf("unexpected updated status: got %q want %q", tasks[0].Status, actions.TaskStatusCompleted)
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

	runArtifacts, err := db.ListArtifactsByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list artifacts by run: %v", err)
	}
	if len(runArtifacts) != 1 || runArtifacts[0].ID != artifact.ID || runArtifacts[0].TaskID != task.ID {
		t.Fatalf("unexpected run artifacts: %+v", runArtifacts)
	}

	evidence := Evidence{
		ID:        "ev_1",
		RunID:     run.ID,
		TaskID:    task.ID,
		Kind:      "service",
		DataJSON:  `{"host":"example.com","port":443}`,
		CreatedAt: now,
	}
	if err := db.CreateEvidence(ctx, evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}

	gotEvidence, err := db.GetEvidence(ctx, evidence.ID)
	if err != nil {
		t.Fatalf("get evidence: %v", err)
	}
	if gotEvidence.ID != evidence.ID || gotEvidence.RunID != evidence.RunID || gotEvidence.TaskID != evidence.TaskID || gotEvidence.Kind != evidence.Kind || gotEvidence.DataJSON != evidence.DataJSON {
		t.Fatalf("unexpected get evidence result: %+v", gotEvidence)
	}

	evidenceRows, err := db.ListEvidenceByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidenceRows) != 1 {
		t.Fatalf("unexpected evidence count: got %d want 1", len(evidenceRows))
	}
	if evidenceRows[0].ID != evidence.ID || evidenceRows[0].TaskID != evidence.TaskID || evidenceRows[0].Kind != evidence.Kind || evidenceRows[0].DataJSON != evidence.DataJSON {
		t.Fatalf("unexpected evidence: %+v", evidenceRows[0])
	}

	taskEvidenceRows, err := db.ListEvidenceByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("list evidence by task: %v", err)
	}
	if len(taskEvidenceRows) != 1 || taskEvidenceRows[0].ID != evidence.ID {
		t.Fatalf("unexpected task evidence: %+v", taskEvidenceRows)
	}
	if !evidenceRows[0].CreatedAt.Equal(evidence.CreatedAt) {
		t.Fatalf("unexpected evidence created_at: got %v want %v", evidenceRows[0].CreatedAt, evidence.CreatedAt)
	}
}

func TestListArtifactsByRunFiltersRunAndOrdersByArtifactCreatedAt(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for _, runID := range []string{"run_1", "run_2"} {
		run := Run{ID: runID, Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now}
		if err := db.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run %s: %v", runID, err)
		}
	}
	for _, task := range []Task{
		{ID: "task_1", RunID: "run_1", ActionType: actions.ActionType("http_request"), InputJSON: `{"url":"http://example.com/one"}`, Status: actions.TaskStatusCompleted, CreatedAt: now},
		{ID: "task_2", RunID: "run_1", ActionType: actions.ActionType("http_request"), InputJSON: `{"url":"http://example.com/two"}`, Status: actions.TaskStatusCompleted, CreatedAt: now},
		{ID: "task_other", RunID: "run_2", ActionType: actions.ActionType("http_request"), InputJSON: `{"url":"http://other.example/"}`, Status: actions.TaskStatusCompleted, CreatedAt: now},
	} {
		if err := db.CreateTask(ctx, task); err != nil {
			t.Fatalf("create task %s: %v", task.ID, err)
		}
	}
	for _, artifact := range []Artifact{
		{ID: "artifact_later", TaskID: "task_1", Path: "/tmp/later.html", CreatedAt: now.Add(time.Second)},
		{ID: "artifact_other", TaskID: "task_other", Path: "/tmp/other.html", CreatedAt: now.Add(-time.Hour)},
		{ID: "artifact_earlier", TaskID: "task_2", Path: "/tmp/earlier.html", CreatedAt: now},
	} {
		if err := db.CreateArtifact(ctx, artifact); err != nil {
			t.Fatalf("create artifact %s: %v", artifact.ID, err)
		}
	}

	artifacts, err := db.ListArtifactsByRun(ctx, "run_1")
	if err != nil {
		t.Fatalf("list artifacts by run: %v", err)
	}
	if len(artifacts) != 2 || artifacts[0].ID != "artifact_earlier" || artifacts[1].ID != "artifact_later" {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
}

func TestSessionScopeAndRunsCRUD(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	session := Session{
		ID:        "session_1",
		Name:      "Acme Bug Bounty",
		Kind:      SessionKindBugBounty,
		Status:    SessionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	gotSession, err := db.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if gotSession.ID != session.ID || gotSession.Name != session.Name || gotSession.Kind != session.Kind || gotSession.Status != session.Status {
		t.Fatalf("unexpected session: %+v", gotSession)
	}

	rule := ScopeRule{
		ID:         "scope_1",
		SessionID:  session.ID,
		Effect:     ScopeEffectInclude,
		TargetType: ScopeTargetDomain,
		Value:      "*.example.com",
		CreatedAt:  now,
	}
	if err := db.CreateScopeRule(ctx, rule); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
	rules, err := db.ListScopeRulesBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list scope rules: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != rule.ID || rules[0].Value != rule.Value {
		t.Fatalf("unexpected scope rules: %+v", rules)
	}

	attachedRun := Run{ID: "run_session", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now}
	standaloneRun := Run{ID: "run_standalone", Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now}
	if err := db.CreateRun(ctx, attachedRun); err != nil {
		t.Fatalf("create attached run: %v", err)
	}
	if err := db.CreateRun(ctx, standaloneRun); err != nil {
		t.Fatalf("create standalone run: %v", err)
	}

	gotRun, err := db.GetRun(ctx, attachedRun.ID)
	if err != nil {
		t.Fatalf("get attached run: %v", err)
	}
	if gotRun.SessionID != session.ID {
		t.Fatalf("unexpected attached run session id: %+v", gotRun)
	}
	gotStandalone, err := db.GetRun(ctx, standaloneRun.ID)
	if err != nil {
		t.Fatalf("get standalone run: %v", err)
	}
	if gotStandalone.SessionID != "" {
		t.Fatalf("standalone run should not have session id: %+v", gotStandalone)
	}

	runs, err := db.ListRunsBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list runs by session: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != attachedRun.ID {
		t.Fatalf("unexpected session runs: %+v", runs)
	}

	if err := db.DeleteScopeRule(ctx, rule.ID); err != nil {
		t.Fatalf("delete scope rule: %v", err)
	}
	rules, err = db.ListScopeRulesBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list scope rules after delete: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("scope rule was not deleted: %+v", rules)
	}

	archiveTime := now.Add(time.Minute)
	if err := db.ArchiveSession(ctx, session.ID, archiveTime); err != nil {
		t.Fatalf("archive session: %v", err)
	}
	gotSession, err = db.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("get archived session: %v", err)
	}
	if gotSession.Status != SessionStatusArchived || !gotSession.UpdatedAt.Equal(archiveTime) {
		t.Fatalf("unexpected archived session: %+v", gotSession)
	}
}

func TestCreateTaskRejectsInvalidJSON(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	run := Run{
		ID:        "run_1",
		Mode:      "ctf",
		Status:    actions.RunStatusRunning,
		CreatedAt: now,
	}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	err := db.CreateTask(ctx, Task{
		ID:         "task_bad",
		RunID:      run.ID,
		ActionType: actions.ActionType("probe_http"),
		InputJSON:  `{"target":`,
		Status:     actions.TaskStatusPending,
		CreatedAt:  now,
	})
	if err == nil {
		t.Fatal("expected invalid JSON to be rejected")
	}
}
