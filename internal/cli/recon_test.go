package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ruohao1/penta/internal/storage/sqlite"
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
	if got := queryCount(t, app, "artifacts"); got != 1 {
		t.Fatalf("unexpected artifacts count: got %d want 1", got)
	}
	if got := queryCount(t, app, "evidence"); got != 1 {
		t.Fatalf("unexpected evidence count: got %d want 1", got)
	}

	var runStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM runs LIMIT 1").Scan(&runStatus); err != nil {
		t.Fatalf("query run status: %v", err)
	}
	if runStatus != "completed" {
		t.Fatalf("unexpected run status: got %q want %q", runStatus, "completed")
	}

	var taskStatus string
	if err := app.DB.QueryRowContext(context.Background(), "SELECT status FROM tasks LIMIT 1").Scan(&taskStatus); err != nil {
		t.Fatalf("query task status: %v", err)
	}
	if taskStatus != "completed" {
		t.Fatalf("unexpected task status: got %q want %q", taskStatus, "completed")
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
