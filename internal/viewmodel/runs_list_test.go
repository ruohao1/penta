package viewmodel

import (
	"context"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestBuildRunListOrdersNewestFirstAndFormatsSessions(t *testing.T) {
	db := openViewModelTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	session := sqlite.Session{ID: "session_1", Name: "local-dev", Kind: sqlite.SessionKindLab, Status: sqlite.SessionStatusActive, CreatedAt: base, UpdatedAt: base}
	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	runs := []sqlite.Run{
		{ID: "run_old", Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: base.Add(-time.Hour)},
		{ID: "run_new", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: base},
	}
	for _, run := range runs {
		if err := db.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run %s: %v", run.ID, err)
		}
	}

	list, err := BuildRunList(ctx, db)
	if err != nil {
		t.Fatalf("build run list: %v", err)
	}
	if len(list.Runs) != 2 {
		t.Fatalf("unexpected run count: %d", len(list.Runs))
	}
	if list.Runs[0].Index != 1 || list.Runs[0].ID != "run_new" || list.Runs[0].Session != "local-dev (lab)" {
		t.Fatalf("unexpected newest row: %+v", list.Runs[0])
	}
	if list.Runs[1].Index != 2 || list.Runs[1].ID != "run_old" || list.Runs[1].Session != "-" {
		t.Fatalf("unexpected standalone row: %+v", list.Runs[1])
	}
}
