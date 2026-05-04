package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	seedtarget "github.com/ruohao1/penta/internal/actions/seed_target"
	"github.com/ruohao1/penta/internal/apperr"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/execute"
	"github.com/ruohao1/penta/internal/model"
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

func createTestSession(t *testing.T, app *App, name, kind string) string {
	t.Helper()
	now := time.Now()
	session := sqlite.Session{ID: "session_" + generateID(), Name: name, Kind: sqlite.SessionKind(kind), Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := app.DB.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("create test session: %v", err)
	}
	return session.ID
}

func createTestScopeRule(t *testing.T, app *App, sessionID, id, effect, targetType, value string) {
	t.Helper()
	rule := sqlite.ScopeRule{ID: id, SessionID: sessionID, Effect: sqlite.ScopeEffect(effect), TargetType: sqlite.ScopeTargetType(targetType), Value: value, CreatedAt: time.Now()}
	if err := app.DB.CreateScopeRule(context.Background(), rule); err != nil {
		t.Fatalf("create test scope rule: %v", err)
	}
}

func runTask(t *testing.T, app *App, taskID string) error {
	t.Helper()

	executor := &execute.Executor{DB: app.DB}
	return executor.RunTask(context.Background(), taskID)
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

func assertAppErrorKind(t *testing.T, err error, kind apperr.Kind) {
	t.Helper()
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if appErr.Kind != kind {
		t.Fatalf("unexpected app error kind: got %s want %s", appErr.Kind, kind)
	}
}

func newReconHTTPServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("unexpected fetch path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func servicePartsFromURL(t *testing.T, rawURL string) (host, scheme string, port int) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	port, err = strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return parsed.Hostname(), parsed.Scheme, port
}

func TestReconCommandCreatesRunTaskArtifactAndEvidence(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	target := newReconHTTPServer(t)
	host, scheme, port := servicePartsFromURL(t, target)
	cmd.SetArgs([]string{target})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute recon command: %v", err)
	}

	if got := queryCount(t, app, "runs"); got != 1 {
		t.Fatalf("unexpected runs count: got %d want 1", got)
	}
	if got := queryCount(t, app, "tasks"); got != 3 {
		t.Fatalf("unexpected tasks count: got %d want 3", got)
	}
	if got := queryCount(t, app, "artifacts"); got != 0 {
		t.Fatalf("unexpected artifacts count: got %d want 0", got)
	}
	if got := queryCount(t, app, "evidence"); got != 3 {
		t.Fatalf("unexpected evidence count: got %d want 3", got)
	}

	var runStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM runs LIMIT 1").Scan(&runStatus); err != nil {
		t.Fatalf("query run status: %v", err)
	}
	if runStatus != string(actions.RunStatusCompleted) {
		t.Fatalf("unexpected run status: got %q want %q", runStatus, actions.RunStatusCompleted)
	}

	var runID string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT id FROM runs LIMIT 1").Scan(&runID); err != nil {
		t.Fatalf("query run id: %v", err)
	}

	rows, err := app.DB.ListTasksByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("unexpected listed task count: got %d want 3", len(rows))
	}
	for _, task := range rows {
		if task.Status != actions.TaskStatusCompleted {
			t.Fatalf("unexpected task status for %s: got %q want %q", task.ID, task.Status, actions.TaskStatusCompleted)
		}
	}

	var actionType string
	var inputJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT action_type, input_json FROM tasks WHERE action_type = ? LIMIT 1", string(actions.ActionSeedTarget)).Scan(&actionType, &inputJSON); err != nil {
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

	assertTargetEvidence(t, app, target, "url")
	assertServiceEvidence(t, app, host, scheme, port)
	assertRunEventTypes(t, app, runID, []events.EventType{
		events.EventRunCreated,
		events.EventActionRequested,
		events.EventTaskEnqueued,
		events.EventTaskEnqueued,
		events.EventActionResolved,
		events.EventTaskStarted,
		events.EventEvidenceCreated,
		events.EventTaskCompleted,
		events.EventTaskStarted,
		events.EventEvidenceCreated,
		events.EventTaskCompleted,
		events.EventTaskEnqueued,
		events.EventTaskStarted,
		events.EventEvidenceCreated,
		events.EventTaskCompleted,
		events.EventRunCompleted,
	})
}

func TestReconCommandRequiresTarget(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing target to fail")
	}
}

func TestReconCommandWritesMarkdownReport(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	target := newReconHTTPServer(t)
	host, scheme, port := servicePartsFromURL(t, target)
	cmd.SetArgs([]string{"--no-color", "-o", reportPath, target})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute recon command: %v", err)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	got := string(data)
	serviceURL := scheme + "://" + host + ":" + strconv.Itoa(port)
	for _, want := range []string{"# Penta Recon Report", "## Summary", "## Targets", "## Services", "## HTTP Responses", "- url " + target, "- [" + serviceURL + "](" + serviceURL + ")", "200"} {
		if !strings.Contains(got, want) {
			t.Fatalf("report missing %q in %q", want, got)
		}
	}
	if !strings.Contains(out.String(), "Report written: "+reportPath) {
		t.Fatalf("stdout missing report path: %q", out.String())
	}
}

func TestReconCommandRedactsFinalAndMarkdownReports(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	target := newReconHTTPServer(t) + "?token=target-secret"
	cmd.SetArgs([]string{"--no-color", "--redact-report", "-o", reportPath, target})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute recon command: %v", err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	stdoutReport := out.String()
	if idx := strings.LastIndex(stdoutReport, "Recon completed"); idx >= 0 {
		stdoutReport = stdoutReport[idx:]
	}
	for name, got := range map[string]string{"stdout final report": stdoutReport, "markdown": string(data)} {
		if strings.Contains(got, "target-secret") {
			t.Fatalf("%s report leaked secret: %q", name, got)
		}
		if !strings.Contains(got, "[REDACTED]") {
			t.Fatalf("%s report missing redaction marker: %q", name, got)
		}
	}
}

func TestReconCommandAttachesRunToSession(t *testing.T) {
	app := openTestApp(t)
	sessionID := createTestSession(t, app, "Acme", "bugbounty")
	target := newReconHTTPServer(t)
	createTestScopeRule(t, app, sessionID, "scope_include", "include", "url", target)
	cmd := newReconCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--no-color", "--session", sessionID, target})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute recon command: %v", err)
	}

	var storedSessionID string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT session_id FROM runs LIMIT 1").Scan(&storedSessionID); err != nil {
		t.Fatalf("query run session id: %v", err)
	}
	if storedSessionID != sessionID {
		t.Fatalf("unexpected run session id: got %q want %q", storedSessionID, sessionID)
	}
	if !strings.Contains(out.String(), "Session "+sessionID+" (Acme, bugbounty)") {
		t.Fatalf("session context missing from output: %q", out.String())
	}
}

func TestReconCommandBlocksOutOfScopeSessionTargetBeforeRunCreation(t *testing.T) {
	app := openTestApp(t)
	sessionID := createTestSession(t, app, "Acme", "bugbounty")
	createTestScopeRule(t, app, sessionID, "scope_include", "include", "domain", "*.example.com")
	cmd := newReconCommand(app)
	cmd.SetArgs([]string{"--session", sessionID, "1.2.3.4"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected out-of-scope target to fail")
	}
	if !strings.Contains(err.Error(), "target outside session scope") {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAppErrorKind(t, err, apperr.KindForbidden)
	if got := queryCount(t, app, "runs"); got != 0 {
		t.Fatalf("out-of-scope target created runs: got %d want 0", got)
	}
}

func TestReconCommandRejectsArchivedSessionBeforeRunCreation(t *testing.T) {
	app := openTestApp(t)
	sessionID := createTestSession(t, app, "Old", "ctf")
	if err := app.DB.ArchiveSession(context.Background(), sessionID, time.Now()); err != nil {
		t.Fatalf("archive session: %v", err)
	}
	cmd := newReconCommand(app)
	cmd.SetArgs([]string{"--session", sessionID, "1.2.3.4"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected archived session to fail")
	}
	if !strings.Contains(err.Error(), "is archived") {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAppErrorKind(t, err, apperr.KindConflict)
	if got := queryCount(t, app, "runs"); got != 0 {
		t.Fatalf("archived session target created runs: got %d want 0", got)
	}
}

func TestReconCommandDoesNotOverwriteExistingReport(t *testing.T) {
	app := openTestApp(t)
	cmd := newReconCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(reportPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing report: %v", err)
	}
	cmd.SetArgs([]string{"-q", "-o", reportPath, "1.2.3.4"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected existing report file to fail")
	}
	if !strings.Contains(err.Error(), "report file already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAppErrorKind(t, err, apperr.KindConflict)
	if strings.Contains(out.String(), "Recon started") || strings.Contains(out.String(), "Usage:") {
		t.Fatalf("existing report failure should not run recon or print usage: %q", out.String())
	}
	if got := queryCount(t, app, "runs"); got != 0 {
		t.Fatalf("existing report failure created runs: got %d want 0", got)
	}
	data, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("read existing report: %v", readErr)
	}
	if string(data) != "existing" {
		t.Fatalf("existing report was overwritten: %q", data)
	}
}

func TestRootCommandSuppressesUsageForRuntimeErrors(t *testing.T) {
	t.Setenv("PENTA_STORAGE_DB_PATH", filepath.Join(t.TempDir(), "penta.db"))
	reportPath := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(reportPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing report: %v", err)
	}
	cmd := NewPentaCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"recon", "1.2.3.4", "-o", reportPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected existing report file to fail")
	}
	if strings.Contains(out.String(), "Usage:") {
		t.Fatalf("runtime error printed usage: %q", out.String())
	}
}

func TestRootCommandHelpStillShowsUsage(t *testing.T) {
	t.Setenv("PENTA_STORAGE_DB_PATH", filepath.Join(t.TempDir(), "penta.db"))
	cmd := NewPentaCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"recon", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("help output missing usage: %q", out.String())
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

	if err := runTask(t, app, task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	storedTask, err := app.DB.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if storedTask.Status != actions.TaskStatusCompleted {
		t.Fatalf("unexpected task status after executeTask: got %q want %q", storedTask.Status, actions.TaskStatusCompleted)
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

	if err := runTask(t, app, task.ID); err != nil {
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

	if err := runTask(t, app, task.ID); err != nil {
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

	if err := runTask(t, app, task.ID); err != nil {
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

	if err := runTask(t, app, task.ID); err != nil {
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

	if err := runTask(t, app, task.ID); err != nil {
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

	err = runTask(t, app, task.ID)
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

	inputJSON, err := json.Marshal(probehttp.Input{Value: "example.com", Type: targets.TypeDomain})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{ID: "task_probe_domain", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := runTask(t, app, task.ID); err != nil {
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

	inputJSON, err := json.Marshal(probehttp.Input{Value: "1.2.3.4", Type: targets.TypeIP})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{ID: "task_probe_ip", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := runTask(t, app, task.ID); err != nil {
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

	inputJSON, err := json.Marshal(probehttp.Input{Value: "https://example.com:8443/foo?a=b", Type: targets.TypeURL})
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}

	task := sqlite.Task{ID: "task_probe_url", RunID: run.ID, ActionType: actions.ActionProbeHTTP, InputJSON: string(inputJSON), Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := app.DB.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := runTask(t, app, task.ID); err != nil {
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

	inputJSON, err := json.Marshal(probehttp.Input{Value: "10.0.0.0/24", Type: targets.TypeCIDR})
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

	err = runTask(t, app, task.ID)
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

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence WHERE kind = ? LIMIT 1", "service").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query service evidence: %v", err)
	}

	var payload model.Service
	if err := json.Unmarshal([]byte(evidenceJSON), &payload); err != nil {
		t.Fatalf("unmarshal service evidence: %v", err)
	}
	if payload.Host != host || payload.Scheme != scheme || payload.Port != port {
		t.Fatalf("unexpected service evidence: %+v", payload)
	}
}

func assertTargetEvidence(t *testing.T, app *App, value, targetType string) {
	t.Helper()

	var evidenceJSON string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT data_json FROM evidence WHERE kind = ? LIMIT 1", "target").Scan(&evidenceJSON); err != nil {
		t.Fatalf("query target evidence: %v", err)
	}

	var payload seedtarget.Evidence
	if err := json.Unmarshal([]byte(evidenceJSON), &payload); err != nil {
		t.Fatalf("unmarshal target evidence: %v", err)
	}
	if payload.Value != value || string(payload.Type) != targetType {
		t.Fatalf("unexpected target evidence: %+v", payload)
	}
}

func assertRunEventTypes(t *testing.T, app *App, runID string, want []events.EventType) {
	t.Helper()

	sink := &events.SQLiteSink{DB: app.DB}
	got, err := sink.ListByRunSinceSeq(context.Background(), runID, 0, 100)
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected event count: got %d want %d", len(got), len(want))
	}
	for i, evt := range got {
		if evt.Seq != int64(i+1) {
			t.Fatalf("unexpected event seq at %d: got %d want %d", i, evt.Seq, i+1)
		}
		if evt.EventType != want[i] {
			t.Fatalf("unexpected event type at %d: got %q want %q", i, evt.EventType, want[i])
		}
	}
}

func TestRunReconCommandMarksRunFailedWhenExecutionFails(t *testing.T) {
	app := openTestApp(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runReconCommand(cmd, app, "")
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

	if got := queryCount(t, app, "tasks"); got != 0 {
		t.Fatalf("unexpected tasks count after resolver failure: got %d want 0", got)
	}
}
