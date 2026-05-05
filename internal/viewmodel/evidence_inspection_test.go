package viewmodel

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/apperr"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestBuildEvidenceListDefaultsToLatestRunAndIndexesEvidence(t *testing.T) {
	db := openEvidenceTestDB(t)
	older := createEvidenceTestRun(t, db, "run_old", -time.Hour)
	newer := createEvidenceTestRun(t, db, "run_new", 0)
	createEvidenceTestItem(t, db, older.ID, "evidence_old", "service", `{"scheme":"http","host":"old.example","port":80}`, 0)
	createEvidenceTestItem(t, db, newer.ID, "evidence_service", "service", `{"scheme":"http","host":"localhost","port":8080}`, 0)
	createEvidenceTestItem(t, db, newer.ID, "evidence_http", "http_response", `{"url":"http://localhost:8080/docs","status_code":200}`, time.Second)

	list, err := BuildEvidenceList(context.Background(), db, "")
	if err != nil {
		t.Fatalf("build evidence list: %v", err)
	}
	if list.Run.ID != newer.ID || !list.Latest {
		t.Fatalf("unexpected run context: %+v", list)
	}
	if len(list.Evidence) != 2 || list.Evidence[0].Index != 1 || list.Evidence[0].ID != "evidence_service" || list.Evidence[1].Index != 2 || list.Evidence[1].ID != "evidence_http" {
		t.Fatalf("unexpected indexed evidence: %+v", list.Evidence)
	}
}

func TestResolveEvidenceSupportsIndexIDAndSemanticSelectors(t *testing.T) {
	db := openEvidenceTestDB(t)
	run := createEvidenceTestRun(t, db, "run_1", 0)
	createEvidenceTestItem(t, db, run.ID, "evidence_service", "service", `{"scheme":"http","host":"localhost","port":8080}`, 0)
	createEvidenceTestItem(t, db, run.ID, "evidence_http", "http_response", `{"url":"http://localhost:8080/docs","status_code":200}`, time.Second)
	createEvidenceTestItem(t, db, run.ID, "evidence_crawl", "crawl", `{"source_url":"http://localhost:8080/","urls":["http://localhost:8080/docs"]}`, 2*time.Second)

	for _, tc := range []struct {
		selector string
		wantID   string
	}{
		{selector: "2", wantID: "evidence_http"},
		{selector: "evidence_service", wantID: "evidence_service"},
		{selector: "http_response:/docs", wantID: "evidence_http"},
		{selector: "service:http://localhost:8080", wantID: "evidence_service"},
		{selector: "crawl:/", wantID: "evidence_crawl"},
	} {
		t.Run(tc.selector, func(t *testing.T) {
			_, item, err := ResolveEvidence(context.Background(), db, run.ID, tc.selector)
			if err != nil {
				t.Fatalf("resolve evidence: %v", err)
			}
			if item.ID != tc.wantID {
				t.Fatalf("unexpected evidence: got %s want %s", item.ID, tc.wantID)
			}
		})
	}
}

func TestResolveEvidenceIDCanSelectNonLatestRun(t *testing.T) {
	db := openEvidenceTestDB(t)
	older := createEvidenceTestRun(t, db, "run_old", -time.Hour)
	createEvidenceTestItem(t, db, older.ID, "evidence_old", "service", `{"scheme":"http","host":"old.example","port":80}`, 0)
	newer := createEvidenceTestRun(t, db, "run_new", 0)
	createEvidenceTestItem(t, db, newer.ID, "evidence_new", "service", `{"scheme":"http","host":"new.example","port":80}`, 0)

	list, item, err := ResolveEvidence(context.Background(), db, "", "evidence_old")
	if err != nil {
		t.Fatalf("resolve evidence id: %v", err)
	}
	if list.Run.ID != older.ID || list.Latest {
		t.Fatalf("unexpected run context: %+v", list)
	}
	if item.ID != "evidence_old" {
		t.Fatalf("unexpected evidence: %+v", item)
	}
}

func TestResolveEvidenceAmbiguousSelectorSuggestsList(t *testing.T) {
	db := openEvidenceTestDB(t)
	run := createEvidenceTestRun(t, db, "run_1", 0)
	createEvidenceTestItem(t, db, run.ID, "evidence_one", "http_response", `{"url":"http://localhost:8080/docs","status_code":200}`, 0)
	createEvidenceTestItem(t, db, run.ID, "evidence_two", "http_response", `{"url":"http://localhost:8081/docs","status_code":200}`, time.Second)

	_, _, err := ResolveEvidence(context.Background(), db, run.ID, "http_response:/docs")
	if err == nil {
		t.Fatal("expected ambiguous selector to fail")
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.KindConflict {
		t.Fatalf("expected conflict app error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "ambiguous evidence selector") || !strings.Contains(err.Error(), "penta evidence list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func openEvidenceTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createEvidenceTestRun(t *testing.T, db *sqlite.DB, id string, offset time.Duration) sqlite.Run {
	t.Helper()
	run := sqlite.Run{ID: id, Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: time.Now().UTC().Truncate(time.Second).Add(offset)}
	if err := db.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

func createEvidenceTestItem(t *testing.T, db *sqlite.DB, runID, id, kind, dataJSON string, offset time.Duration) {
	t.Helper()
	createdAt := time.Now().UTC().Truncate(time.Second).Add(offset)
	taskID := "task_" + id
	task := sqlite.Task{ID: taskID, RunID: runID, ActionType: actions.ActionType("probe_http"), InputJSON: `{"target":"example.com"}`, Status: actions.TaskStatusCompleted, CreatedAt: createdAt}
	if err := db.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	evidence := sqlite.Evidence{ID: id, RunID: runID, TaskID: taskID, Kind: kind, DataJSON: dataJSON, CreatedAt: createdAt}
	if err := db.CreateEvidence(context.Background(), evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
}
