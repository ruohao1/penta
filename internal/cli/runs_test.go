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

func TestRunsListPrintsNoRunsForEmptyDB(t *testing.T) {
	app := openTestApp(t)

	out := executeRunsCommand(t, app, "list")
	if out != "No runs\n" {
		t.Fatalf("unexpected empty output: %q", out)
	}
}

func TestRunsListPrintsNumberedRows(t *testing.T) {
	app := openTestApp(t)
	ctx := context.Background()
	createdAt := time.Date(2026, 5, 5, 12, 34, 56, 0, time.UTC)
	session := sqlite.Session{ID: "session_1", Name: "local-dev", Kind: sqlite.SessionKindLab, Status: sqlite.SessionStatusActive, CreatedAt: createdAt, UpdatedAt: createdAt}
	if err := app.DB.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	runs := []sqlite.Run{
		{ID: "run_f4ee77d0-fd15-4068-b3b8-7d0f7099a776", Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: createdAt.Add(-time.Hour)},
		{ID: "run_jkqtqegyjq", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: createdAt},
	}
	for _, run := range runs {
		if err := app.DB.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run %s: %v", run.ID, err)
		}
	}

	out := executeRunsCommand(t, app, "list")
	for _, want := range []string{"#  Run", "Mode", "1  run_jkqtqegyjq  recon", "running", "local-dev (lab)", "2026-05-05T12:34:56Z", "2  run_f4ee77d0f…  recon", "completed", "-"} {
		if !strings.Contains(out, want) {
			t.Fatalf("runs list missing %q in %q", want, out)
		}
	}
}

func TestDisplayRunIDKeepsShortIDsAndTruncatesLongIDs(t *testing.T) {
	if got := displayRunID("run_jkqtqegyjq"); got != "run_jkqtqegyjq" {
		t.Fatalf("short id display: got %q", got)
	}
	if got := displayRunID("run_f4ee77d0-fd15-4068-b3b8-7d0f7099a776"); got != "run_f4ee77d0f…" {
		t.Fatalf("long id display: got %q", got)
	}
}

func executeRunsCommand(t *testing.T, app *App, args ...string) string {
	t.Helper()
	cmd := newRunsCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute runs %v: %v\noutput: %s", args, err, out.String())
	}
	return out.String()
}
