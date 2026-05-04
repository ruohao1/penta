package execute

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
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
