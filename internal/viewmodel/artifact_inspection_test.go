package viewmodel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/apperr"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestBuildArtifactListDefaultsToLatestRunAndIndexesArtifacts(t *testing.T) {
	db := openEvidenceTestDB(t)
	older := createEvidenceTestRun(t, db, "run_old", -time.Hour)
	newer := createEvidenceTestRun(t, db, "run_new", 0)
	createArtifactInspectionItem(t, db, older.ID, "artifact_old", "/tmp/old.html", "http://old.example/", 0)
	createArtifactInspectionItem(t, db, newer.ID, "artifact_root", "/tmp/root.html", "http://localhost:8080/", 0)
	createArtifactInspectionItem(t, db, newer.ID, "artifact_secret", "/tmp/secret.html", "http://localhost:8080/secret", time.Second)

	list, err := BuildArtifactList(context.Background(), db, "")
	if err != nil {
		t.Fatalf("build artifact list: %v", err)
	}
	if list.Run.ID != newer.ID || !list.Latest {
		t.Fatalf("unexpected run context: %+v", list)
	}
	if len(list.Artifacts) != 2 || list.Artifacts[0].Index != 1 || list.Artifacts[0].Row.ID != "artifact_root" || list.Artifacts[1].Index != 2 || list.Artifacts[1].Row.ID != "artifact_secret" {
		t.Fatalf("unexpected indexed artifacts: %+v", list.Artifacts)
	}
	if list.Artifacts[1].Kind != "body" || list.Artifacts[1].Source != "/secret" {
		t.Fatalf("unexpected artifact summary: %+v", list.Artifacts[1])
	}
}

func TestResolveArtifactSupportsIndexIDAndBodySelector(t *testing.T) {
	db := openEvidenceTestDB(t)
	run := createEvidenceTestRun(t, db, "run_1", 0)
	createArtifactInspectionItem(t, db, run.ID, "artifact_root", "/tmp/root.html", "http://localhost:8080/", 0)
	createArtifactInspectionItem(t, db, run.ID, "artifact_secret", "/tmp/secret.html", "http://localhost:8080/secret", time.Second)

	for _, tc := range []struct {
		selector string
		wantID   string
	}{
		{selector: "2", wantID: "artifact_secret"},
		{selector: "artifact_root", wantID: "artifact_root"},
		{selector: "body:/secret", wantID: "artifact_secret"},
	} {
		t.Run(tc.selector, func(t *testing.T) {
			_, item, err := ResolveArtifact(context.Background(), db, run.ID, tc.selector)
			if err != nil {
				t.Fatalf("resolve artifact: %v", err)
			}
			if item.Row.ID != tc.wantID {
				t.Fatalf("unexpected artifact: got %s want %s", item.Row.ID, tc.wantID)
			}
		})
	}
}

func TestResolveArtifactIDCanSelectNonLatestRun(t *testing.T) {
	db := openEvidenceTestDB(t)
	older := createEvidenceTestRun(t, db, "run_old", -time.Hour)
	createArtifactInspectionItem(t, db, older.ID, "artifact_old", "/tmp/old.html", "http://old.example/", 0)
	newer := createEvidenceTestRun(t, db, "run_new", 0)
	createArtifactInspectionItem(t, db, newer.ID, "artifact_new", "/tmp/new.html", "http://new.example/", 0)

	list, item, err := ResolveArtifact(context.Background(), db, "", "artifact_old")
	if err != nil {
		t.Fatalf("resolve artifact id: %v", err)
	}
	if list.Run.ID != older.ID || list.Latest {
		t.Fatalf("unexpected run context: %+v", list)
	}
	if item.Row.ID != "artifact_old" {
		t.Fatalf("unexpected artifact: %+v", item)
	}
}

func TestResolveArtifactAmbiguousBodySelectorSuggestsList(t *testing.T) {
	db := openEvidenceTestDB(t)
	run := createEvidenceTestRun(t, db, "run_1", 0)
	createArtifactInspectionItem(t, db, run.ID, "artifact_one", "/tmp/one.html", "http://localhost:8080/secret", 0)
	createArtifactInspectionItem(t, db, run.ID, "artifact_two", "/tmp/two.html", "http://localhost:8081/secret", time.Second)

	_, _, err := ResolveArtifact(context.Background(), db, run.ID, "body:/secret")
	if err == nil {
		t.Fatal("expected ambiguous selector to fail")
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.KindConflict {
		t.Fatalf("expected conflict app error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "ambiguous artifact selector") || !strings.Contains(err.Error(), "penta artifacts list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func createArtifactInspectionItem(t *testing.T, db *sqlite.DB, runID, artifactID, path, url string, offset time.Duration) {
	t.Helper()
	createdAt := time.Now().UTC().Truncate(time.Second).Add(offset)
	taskID := "task_" + artifactID
	task := sqlite.Task{ID: taskID, RunID: runID, ActionType: actions.ActionType("http_request"), InputJSON: fmt.Sprintf(`{"url":%q}`, url), Status: actions.TaskStatusCompleted, CreatedAt: createdAt}
	if err := db.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	artifact := sqlite.Artifact{ID: artifactID, TaskID: taskID, Path: path, CreatedAt: createdAt}
	if err := db.CreateArtifact(context.Background(), artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}
	evidence := sqlite.Evidence{ID: "evidence_" + artifactID, RunID: runID, TaskID: taskID, Kind: "http_response", DataJSON: fmt.Sprintf(`{"url":%q,"status_code":200,"body_artifact_id":%q}`, url, artifactID), CreatedAt: createdAt}
	if err := db.CreateEvidence(context.Background(), evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
}
