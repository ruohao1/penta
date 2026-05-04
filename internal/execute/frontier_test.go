package execute

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/scheduler"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestFrontierRejectsUnsupportedCandidateAction(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_frontier", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	frontier := Frontier{DB: db, Registry: Registry{}}

	err := frontier.EnqueueCandidate(ctx, &run, nil, scheduler.CandidateTask{ActionType: actions.ActionCrawl, InputJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "derived unsupported action type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFrontierEmitsPolicyBlockForApprovalRequiredCandidate(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_policy", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	sink := &events.SQLiteSink{DB: db}
	frontier := Frontier{DB: db, Events: sink, Registry: Registry{
		actions.ActionCrawl: {Spec: actions.ActionSpec{Type: actions.ActionCrawl, Permission: actions.PermissionActiveScan, InputKind: "crawl.input"}, Handler: noopHandler},
	}}

	if err := frontier.EnqueueCandidate(ctx, &run, nil, scheduler.CandidateTask{ActionType: actions.ActionCrawl, InputJSON: `{"url":"https://example.com"}`}); err != nil {
		t.Fatalf("enqueue candidate: %v", err)
	}
	if got := countTasksByRun(t, db, run.ID); got != 0 {
		t.Fatalf("policy-blocked candidate created tasks: got %d want 0", got)
	}
	eventsRows, err := sink.ListByRunSinceSeq(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	blocked := findEventByType(t, eventsRows, events.EventCandidateBlocked)
	var payload events.CandidateBlockedPayload
	if err := json.Unmarshal([]byte(blocked.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal blocked payload: %v", err)
	}
	if payload.ActionType != actions.ActionCrawl || payload.Source != "policy" || !strings.Contains(payload.Reason, "requires approval") {
		t.Fatalf("unexpected policy block payload: %+v", payload)
	}
}

func countTasksByRun(t *testing.T, db *sqlite.DB, runID string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM tasks WHERE run_id = ?", runID).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	return count
}
