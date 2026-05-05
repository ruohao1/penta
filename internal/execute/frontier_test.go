package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	httprequest "github.com/ruohao1/penta/internal/actions/http_request"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/model"
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

func TestFrontierBlocksCrawlDerivedRequestBeyondMaxDepth(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_depth", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	sink := &events.SQLiteSink{DB: db}
	frontier := Frontier{DB: db, Events: sink}
	inputJSON := mustJSON(t, httprequest.Input{Method: "GET", URL: "https://example.com/deep", Depth: defaultMaxCrawlDepth + 1})

	if err := frontier.EnqueueCandidate(ctx, &run, nil, scheduler.CandidateTask{ActionType: actions.ActionHTTPRequest, InputJSON: inputJSON, Target: &model.TargetRef{Value: "https://example.com/deep", Type: "url"}, CrawlDerived: true, Depth: defaultMaxCrawlDepth + 1}); err != nil {
		t.Fatalf("enqueue candidate: %v", err)
	}
	if got := countTasksByRun(t, db, run.ID); got != 0 {
		t.Fatalf("depth-blocked candidate created tasks: got %d want 0", got)
	}
	assertBlockedCandidate(t, sink, run.ID, actions.ActionHTTPRequest, "crawl_depth", "exceeds max depth")
}

func TestFrontierEnqueuesCrawlDerivedRequestWithinMaxDepth(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_depth_allowed", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	frontier := Frontier{DB: db}
	inputJSON := mustJSON(t, httprequest.Input{Method: "GET", URL: "https://example.com/page", Depth: defaultMaxCrawlDepth})

	if err := frontier.EnqueueCandidate(ctx, &run, nil, scheduler.CandidateTask{ActionType: actions.ActionHTTPRequest, InputJSON: inputJSON, Target: &model.TargetRef{Value: "https://example.com/page", Type: "url"}, CrawlDerived: true, Depth: defaultMaxCrawlDepth}); err != nil {
		t.Fatalf("enqueue candidate: %v", err)
	}
	if got := countTasksByRun(t, db, run.ID); got != 1 {
		t.Fatalf("unexpected task count: got %d want 1", got)
	}
}

func TestFrontierBlocksCrawlDerivedRequestWhenBudgetReached(t *testing.T) {
	db := openExecutorTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_budget", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	for i := 0; i < defaultMaxCrawlURLsPerRun; i++ {
		inputJSON := mustJSON(t, httprequest.Input{Method: "GET", URL: fmt.Sprintf("https://example.com/%d", i), Depth: 1})
		task := sqlite.Task{ID: fmt.Sprintf("task_budget_%d", i), RunID: run.ID, ActionType: actions.ActionHTTPRequest, InputJSON: inputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
		if err := db.CreateTask(ctx, task); err != nil {
			t.Fatalf("create task: %v", err)
		}
	}
	sink := &events.SQLiteSink{DB: db}
	frontier := Frontier{DB: db, Events: sink}
	inputJSON := mustJSON(t, httprequest.Input{Method: "GET", URL: "https://example.com/over", Depth: 1})

	if err := frontier.EnqueueCandidate(ctx, &run, nil, scheduler.CandidateTask{ActionType: actions.ActionHTTPRequest, InputJSON: inputJSON, Target: &model.TargetRef{Value: "https://example.com/over", Type: "url"}, CrawlDerived: true, Depth: 1}); err != nil {
		t.Fatalf("enqueue candidate: %v", err)
	}
	if got := countTasksByRun(t, db, run.ID); got != defaultMaxCrawlURLsPerRun {
		t.Fatalf("budget-blocked candidate changed tasks: got %d want %d", got, defaultMaxCrawlURLsPerRun)
	}
	assertBlockedCandidate(t, sink, run.ID, actions.ActionHTTPRequest, "crawl_budget", "budget")
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

func assertBlockedCandidate(t *testing.T, sink *events.SQLiteSink, runID string, actionType actions.ActionType, source, reasonContains string) {
	t.Helper()
	eventsRows, err := sink.ListByRunSinceSeq(context.Background(), runID, 0, 100)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	blocked := findEventByType(t, eventsRows, events.EventCandidateBlocked)
	var payload events.CandidateBlockedPayload
	if err := json.Unmarshal([]byte(blocked.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal blocked payload: %v", err)
	}
	if payload.ActionType != actionType || payload.Source != source || !strings.Contains(payload.Reason, reasonContains) {
		t.Fatalf("unexpected blocked payload: %+v", payload)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(data)
}
