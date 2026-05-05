package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestArtifactsListUsesLatestRunContextAndMetadataOnlyRows(t *testing.T) {
	app := openTestApp(t)
	createCLIEvidenceRun(t, app, "run_old", -time.Hour)
	run := createCLIEvidenceRun(t, app, "run_new", 0)
	createCLIArtifactItem(t, app, run.ID, "artifact_root", "/tmp/root.html", "http://localhost:8080/", 0)
	createCLIArtifactItem(t, app, run.ID, "artifact_secret", "/tmp/secret.html", "http://localhost:8080/secret", time.Second)

	out := executeArtifactsCommand(t, app, "list")
	for _, want := range []string{"Run run_new (latest)", "#  Kind      Source   Path", "1  body      /", "/tmp/root.html", "2  body      /secret", "/tmp/secret.html"} {
		if !strings.Contains(out, want) {
			t.Fatalf("artifacts list missing %q in %q", want, out)
		}
	}
}

func TestArtifactsShowByIndexAndBodySelector(t *testing.T) {
	app := openTestApp(t)
	run := createCLIEvidenceRun(t, app, "run_1", 0)
	createCLIArtifactItem(t, app, run.ID, "artifact_root", "/tmp/root.html", "http://localhost:8080/", 0)
	createCLIArtifactItem(t, app, run.ID, "artifact_secret", "/tmp/secret.html", "http://localhost:8080/secret", time.Second)

	byIndex := executeArtifactsCommand(t, app, "show", "2")
	for _, want := range []string{"Run       run_1 (latest)", "Index     2", "ID        artifact_secret", "Task      task_artifact_secret", "Kind      body", "Source    /secret", "Path      /tmp/secret.html", "Created"} {
		if !strings.Contains(byIndex, want) {
			t.Fatalf("artifacts show by index missing %q in %q", want, byIndex)
		}
	}

	bySelector := executeArtifactsCommand(t, app, "show", "body:/secret")
	for _, want := range []string{"Index     2", "ID        artifact_secret", "Source    /secret"} {
		if !strings.Contains(bySelector, want) {
			t.Fatalf("artifacts show by selector missing %q in %q", want, bySelector)
		}
	}
}

func executeArtifactsCommand(t *testing.T, app *App, args ...string) string {
	t.Helper()
	cmd := newArtifactsCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute artifacts %v: %v\noutput: %s", args, err, out.String())
	}
	return out.String()
}

func createCLIArtifactItem(t *testing.T, app *App, runID, artifactID, path, url string, offset time.Duration) {
	t.Helper()
	createdAt := time.Now().UTC().Truncate(time.Second).Add(offset)
	taskID := "task_" + artifactID
	task := sqlite.Task{ID: taskID, RunID: runID, ActionType: actions.ActionType("http_request"), InputJSON: fmt.Sprintf(`{"url":%q}`, url), Status: actions.TaskStatusCompleted, CreatedAt: createdAt}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	artifact := sqlite.Artifact{ID: artifactID, TaskID: taskID, Path: path, CreatedAt: createdAt}
	if err := app.DB.CreateArtifact(context.Background(), artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}
	evidence := sqlite.Evidence{ID: "evidence_" + artifactID, RunID: runID, TaskID: taskID, Kind: "http_response", DataJSON: fmt.Sprintf(`{"url":%q,"status_code":200,"body_artifact_id":%q}`, url, artifactID), CreatedAt: createdAt}
	if err := app.DB.CreateEvidence(context.Background(), evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
}
