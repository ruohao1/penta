package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
	"github.com/spf13/cobra"
)

func openTestApp(t *testing.T) *App {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "penta.db")
	db, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return &App{DB: db}
}

func queryCount(t *testing.T, app *App, table string) int {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM " + table
	if err := app.DB.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return count
}

func TestReconCommandCreatesRunTaskArtifactAndEvidence(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	target := "example.com"
	cmd.SetArgs([]string{"example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute recon command: %v", err)
	}

	if got := queryCount(t, app, "runs"); got != 1 {
		t.Fatalf("unexpected runs count: got %d want 1", got)
	}
	if got := queryCount(t, app, "tasks"); got != 1 {
		t.Fatalf("unexpected tasks count: got %d want 1", got)
	}
	if got := queryCount(t, app, "artifacts"); got != 0 {
		t.Fatalf("unexpected artifacts count: got %d want 0", got)
	}
	if got := queryCount(t, app, "evidence"); got != 1 {
		t.Fatalf("unexpected evidence count: got %d want 1", got)
	}

	var runStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM runs LIMIT 1").Scan(&runStatus); err != nil {
		t.Fatalf("query run status: %v", err)
	}
	if runStatus != string(actions.RunStatusCompleted) {
		t.Fatalf("unexpected run status: got %q want %q", runStatus, actions.RunStatusCompleted)
	}

	var taskStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM tasks LIMIT 1").Scan(&taskStatus); err != nil {
		t.Fatalf("query task status: %v", err)
	}
	if taskStatus != string(actions.TaskStatusCompleted) {
		t.Fatalf("unexpected task status: got %q want %q", taskStatus, actions.TaskStatusCompleted)
	}

	var actionType string
	var inputJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT action_type, input_json FROM tasks LIMIT 1").Scan(&actionType, &inputJSON); err != nil {
		t.Fatalf("query task payload: %v", err)
	}
	if actionType != string(actions.ActionSeedTarget) {
		t.Fatalf("unexpected action type: got %q want %q", actionType, actions.ActionSeedTarget)
	}

	var inputPayload map[string]string
	if err := json.Unmarshal([]byte(inputJSON), &inputPayload); err != nil {
		t.Fatalf("unmarshal task input json: %v", err)
	}
	if inputPayload["raw"] != target {
		t.Fatalf("unexpected task raw input: got %q want %q", inputPayload["raw"], target)
	}

	var evidenceKind string
	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT kind, data_json FROM evidence LIMIT 1").Scan(&evidenceKind, &evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}
	if evidenceKind != "target" {
		t.Fatalf("unexpected evidence kind: got %q want %q", evidenceKind, "target")
	}

	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != target {
		t.Fatalf("unexpected evidence value: got %q want %q", evidencePayload["value"], target)
	}
	if evidencePayload["type"] != "domain" {
		t.Fatalf("unexpected evidence type: got %q want %q", evidencePayload["type"], "domain")
	}
}

func TestReconCommandRequiresTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing target to fail")
	}
}

func TestExecuteTaskCreatesArtifactsAndEvidenceForSeedTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{
		ID:        "run_exec",
		Mode:      "recon",
		Status:    actions.RunStatusRunning,
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(map[string]string{"raw": "example.com"})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{
		ID:         "task_exec",
		RunID:      run.ID,
		ActionType: actions.ActionSeedTarget,
		InputJSON:  string(inputJSON),
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	storedTask, err := app.DB.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if storedTask.Status != actions.TaskStatusRunning {
		t.Fatalf("unexpected task status after executeTask: got %q want %q", storedTask.Status, actions.TaskStatusRunning)
	}

	if got := queryCount(t, app, "artifacts"); got != 0 {
		t.Fatalf("unexpected artifacts count: got %d want 0", got)
	}
	if got := queryCount(t, app, "evidence"); got != 1 {
		t.Fatalf("unexpected evidence count: got %d want 1", got)
	}

	var evidenceKind string
	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT kind, data_json FROM evidence LIMIT 1").Scan(&evidenceKind, &evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}
	if evidenceKind != "target" {
		t.Fatalf("unexpected evidence kind: got %q want %q", evidenceKind, "target")
	}

	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != "example.com" {
		t.Fatalf("unexpected evidence value: got %q want %q", evidencePayload["value"], "example.com")
	}
	if evidencePayload["type"] != "domain" {
		t.Fatalf("unexpected evidence type: got %q want %q", evidencePayload["type"], "domain")
	}
}

func TestExecuteTaskClassifiesIPTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{
		ID:        "run_ip",
		Mode:      "recon",
		Status:    actions.RunStatusRunning,
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(map[string]string{"raw": "1.2.3.4"})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{
		ID:         "task_ip",
		RunID:      run.ID,
		ActionType: actions.ActionSeedTarget,
		InputJSON:  string(inputJSON),
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence LIMIT 1").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}

	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != "1.2.3.4" {
		t.Fatalf("unexpected evidence value: got %q want %q", evidencePayload["value"], "1.2.3.4")
	}
	if evidencePayload["type"] != "ip" {
		t.Fatalf("unexpected evidence type: got %q want %q", evidencePayload["type"], "ip")
	}
}

func TestExecuteTaskClassifiesURLTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{
		ID:        "run_url",
		Mode:      "recon",
		Status:    actions.RunStatusRunning,
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	target := "https://example.com/foo?a=b"
	inputJSON, err := json.Marshal(map[string]string{"raw": target})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{
		ID:         "task_url",
		RunID:      run.ID,
		ActionType: actions.ActionSeedTarget,
		InputJSON:  string(inputJSON),
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence LIMIT 1").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}

	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != target {
		t.Fatalf("unexpected evidence value: got %q want %q", evidencePayload["value"], target)
	}
	if evidencePayload["type"] != "url" {
		t.Fatalf("unexpected evidence type: got %q want %q", evidencePayload["type"], "url")
	}
}

func TestExecuteTaskClassifiesCIDRTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_cidr", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	target := "10.0.0.0/24"
	inputJSON, err := json.Marshal(map[string]string{"raw": target})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{ID: "task_cidr", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence LIMIT 1").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}
	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != target || evidencePayload["type"] != "cidr" {
		t.Fatalf("unexpected evidence payload: %+v", evidencePayload)
	}
}

func TestExecuteTaskClassifiesServiceTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_service", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	target := "example.com:443"
	inputJSON, err := json.Marshal(map[string]string{"raw": target})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{ID: "task_service", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence LIMIT 1").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}
	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != target || evidencePayload["type"] != "service" {
		t.Fatalf("unexpected evidence payload: %+v", evidencePayload)
	}
}

func TestExecuteTaskClassifiesIPRangeTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_iprange", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	target := "1-255.1-255.1-255.1-255"
	inputJSON, err := json.Marshal(map[string]string{"raw": target})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{ID: "task_iprange", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence LIMIT 1").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query evidence payload: %v", err)
	}
	var evidencePayload map[string]string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence json: %v", err)
	}
	if evidencePayload["value"] != target || evidencePayload["type"] != "ip_range" {
		t.Fatalf("unexpected evidence payload: %+v", evidencePayload)
	}
}

func TestExecuteTaskMarksUnknownActionFailed(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{
		ID:        "run_unknown",
		Mode:      "recon",
		Status:    actions.RunStatusRunning,
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(map[string]string{"raw": "example.com"})
	if err != nil {
		t.Fatalf("marshal input json: %v", err)
	}

	task := sqlite.Task{
		ID:         "task_unknown",
		RunID:      run.ID,
		ActionType: actions.ActionType("unknown_action"),
		InputJSON:  string(inputJSON),
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	err = executeTask(cmd, app, task.ID)
	if err == nil {
		t.Fatal("expected unknown action execution to fail")
	}

	storedTask, err := app.DB.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if storedTask.Status != actions.TaskStatusFailed {
		t.Fatalf("unexpected task status after failed executeTask: got %q want %q", storedTask.Status, actions.TaskStatusFailed)
	}
}

func TestExecuteTaskHandlesProbeHTTPDomain(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_probe_domain", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(actions.ProbeHTTPInput{Value: "example.com", Type: targets.TypeDomain})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{ID: "task_probe_domain", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	assertServiceEvidence(t, app, "example.com", "https", 443)
}

func TestExecuteTaskHandlesProbeHTTPIP(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_probe_ip", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(actions.ProbeHTTPInput{Value: "1.2.3.4", Type: targets.TypeIP})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{ID: "task_probe_ip", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	assertServiceEvidence(t, app, "1.2.3.4", "https", 443)
}

func TestExecuteTaskHandlesProbeHTTPURL(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_probe_url", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(actions.ProbeHTTPInput{Value: "https://example.com:8443/foo?a=b", Type: targets.TypeURL})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{ID: "task_probe_url", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := executeTask(cmd, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	assertServiceEvidence(t, app, "example.com", "https", 8443)
}

func TestExecuteTaskRejectsProbeHTTPCIDR(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	run := sqlite.Run{ID: "run_probe_cidr", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := app.DB.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	inputJSON, err := json.Marshal(actions.ProbeHTTPInput{Value: "10.0.0.0/24", Type: targets.TypeCIDR})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{
		ID:         "task_probe_cidr",
		RunID:      run.ID,
		ActionType: actions.ActionProbeHTTP,
		InputJSON:  string(inputJSON),
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	err = executeTask(cmd, app, task.ID)
	if err == nil {
		t.Fatal("expected probe_http cidr execution to fail")
	}

	storedTask, err := app.DB.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if storedTask.Status != actions.TaskStatusFailed {
		t.Fatalf("unexpected task status after failed executeTask: got %q want %q", storedTask.Status, actions.TaskStatusFailed)
	}
}

func assertServiceEvidence(t *testing.T, app *App, host, scheme string, port int) {
	t.Helper()

	var evidenceKind string
	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT kind, data_json FROM evidence LIMIT 1").Scan(&evidenceKind, &evidenceJSON); err != nil {
		t.Fatalf("query service evidence: %v", err)
	}
	if evidenceKind != "service" {
		t.Fatalf("unexpected evidence kind: got %q want %q", evidenceKind, "service")
	}

	var payload actions.ServiceEvidence
	if err := json.Unmarshal([]byte(evidenceJSON), &payload); err != nil {
		t.Fatalf("unmarshal service evidence: %v", err)
	}
	if payload.Host != host || payload.Scheme != scheme || payload.Port != port {
		t.Fatalf("unexpected service evidence: %+v", payload)
	}
}

func TestRunReconCommandMarksRunFailedWhenExecutionFails(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	originalExecutor := taskExecutor
	taskExecutor = func(cmd *cobra.Command, app *App, taskID string) error {
		return fmt.Errorf("forced executor failure")
	}
	t.Cleanup(func() {
		taskExecutor = originalExecutor
	})

	err := runReconCommand(cmd, app, "example.com")
	if err == nil {
		t.Fatal("expected runReconCommand to fail when executor fails")
	}

	var runStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM runs LIMIT 1").Scan(&runStatus); err != nil {
		t.Fatalf("query run status: %v", err)
	}
	if runStatus != string(actions.RunStatusFailed) {
		t.Fatalf("unexpected run status after execution failure: got %q want %q", runStatus, actions.RunStatusFailed)
	}

	var taskStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM tasks LIMIT 1").Scan(&taskStatus); err != nil {
		t.Fatalf("query task status: %v", err)
	}
	if taskStatus != string(actions.TaskStatusPending) {
		t.Fatalf("unexpected task status after execution failure: got %q want %q", taskStatus, actions.TaskStatusPending)
	}
}
