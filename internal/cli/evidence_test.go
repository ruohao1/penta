package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestEvidenceListUsesLatestRunContextAndNumberedRows(t *testing.T) {
	app := openTestApp(t)
	createCLIEvidenceRun(t, app, "run_old", -time.Hour)
	run := createCLIEvidenceRun(t, app, "run_new", 0)
	createCLIEvidenceItem(t, app, run.ID, "evidence_service", "service", `{"scheme":"http","host":"localhost","port":8080}`, 0)
	createCLIEvidenceItem(t, app, run.ID, "evidence_http", "http_response", `{"url":"http://localhost:8080/docs","status_code":200,"content_type":"text/html"}`, time.Second)

	out := executeEvidenceCommand(t, app, "list")
	for _, want := range []string{"Run run_new (latest)", "#  Kind           Label", "1  service", "http://localhost:8080", "2  http_response", "http://localhost:8080/docs 200"} {
		if !strings.Contains(out, want) {
			t.Fatalf("evidence list missing %q in %q", want, out)
		}
	}
}

func TestEvidenceShowByIndexAndSemanticSelector(t *testing.T) {
	app := openTestApp(t)
	run := createCLIEvidenceRun(t, app, "run_1", 0)
	createCLIEvidenceItem(t, app, run.ID, "evidence_service", "service", `{"scheme":"http","host":"localhost","port":8080}`, 0)
	createCLIEvidenceItem(t, app, run.ID, "evidence_http", "http_response", `{"url":"http://localhost:8080/docs","status_code":200,"content_type":"text/html"}`, time.Second)

	byIndex := executeEvidenceCommand(t, app, "show", "2")
	for _, want := range []string{"Run       run_1 (latest)", "Index     2", "ID        evidence_http", "Kind      http_response", "Label     http://localhost:8080/docs 200", "Details", "- content-type: text/html"} {
		if !strings.Contains(byIndex, want) {
			t.Fatalf("evidence show by index missing %q in %q", want, byIndex)
		}
	}

	bySelector := executeEvidenceCommand(t, app, "show", "service:http://localhost:8080")
	for _, want := range []string{"Index     1", "ID        evidence_service", "Label     http://localhost:8080"} {
		if !strings.Contains(bySelector, want) {
			t.Fatalf("evidence show by selector missing %q in %q", want, bySelector)
		}
	}
}

func executeEvidenceCommand(t *testing.T, app *App, args ...string) string {
	t.Helper()
	cmd := newEvidenceCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute evidence %v: %v\noutput: %s", args, err, out.String())
	}
	return out.String()
}

func createCLIEvidenceRun(t *testing.T, app *App, id string, offset time.Duration) sqlite.Run {
	t.Helper()
	run := sqlite.Run{ID: id, Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: time.Now().UTC().Truncate(time.Second).Add(offset)}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

func createCLIEvidenceItem(t *testing.T, app *App, runID, id, kind, dataJSON string, offset time.Duration) {
	t.Helper()
	createdAt := time.Now().UTC().Truncate(time.Second).Add(offset)
	taskID := "task_" + id
	task := sqlite.Task{ID: taskID, RunID: runID, ActionType: actions.ActionType("probe_http"), InputJSON: `{"target":"example.com"}`, Status: actions.TaskStatusCompleted, CreatedAt: createdAt}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	evidence := sqlite.Evidence{ID: id, RunID: runID, TaskID: taskID, Kind: kind, DataJSON: dataJSON, CreatedAt: createdAt}
	if err := app.DB.CreateEvidence(context.Background(), evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
}
