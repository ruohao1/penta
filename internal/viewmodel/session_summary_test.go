package viewmodel

import (
	"context"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestBuildSessionSummaryAggregatesSessionRuns(t *testing.T) {
	db := openViewModelTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	session := sqlite.Session{ID: "session_1", Name: "Acme", Kind: sqlite.SessionKindBugBounty, Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	otherSession := sqlite.Session{ID: "session_2", Name: "Other", Kind: sqlite.SessionKindCTF, Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	for _, s := range []sqlite.Session{session, otherSession} {
		if err := db.CreateSession(ctx, s); err != nil {
			t.Fatalf("create session %s: %v", s.ID, err)
		}
	}
	if err := db.CreateScopeRule(ctx, sqlite.ScopeRule{ID: "scope_1", SessionID: session.ID, Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetDomain, Value: "*.example.com", CreatedAt: now}); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
	run := sqlite.Run{ID: "run_1", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now}
	standaloneRun := sqlite.Run{ID: "run_standalone", Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now}
	otherRun := sqlite.Run{ID: "run_other", SessionID: otherSession.ID, Mode: "recon", Status: actions.RunStatusCompleted, CreatedAt: now}
	for _, r := range []sqlite.Run{run, standaloneRun, otherRun} {
		if err := db.CreateRun(ctx, r); err != nil {
			t.Fatalf("create run %s: %v", r.ID, err)
		}
	}
	task := sqlite.Task{ID: "task_1", RunID: run.ID, ActionType: actions.ActionSeedTarget, InputJSON: `{}`, Status: actions.TaskStatusCompleted, CreatedAt: now}
	if err := db.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := db.CreateEvidence(ctx, sqlite.Evidence{ID: "evidence_1", RunID: run.ID, TaskID: task.ID, Kind: "target", DataJSON: `{"value":"example.com","type":"domain"}`, CreatedAt: now}); err != nil {
		t.Fatalf("create evidence: %v", err)
	}

	summary, err := BuildSessionSummary(ctx, db, session.ID)
	if err != nil {
		t.Fatalf("build session summary: %v", err)
	}
	if summary.Session.ID != session.ID || len(summary.ScopeRules) != 1 || len(summary.Runs) != 1 {
		t.Fatalf("unexpected summary identity: %+v", summary)
	}
	if summary.RunCounts[actions.RunStatusCompleted] != 1 || summary.TaskCounts[actions.TaskStatusCompleted] != 1 || summary.EvidenceCounts["target"] != 1 {
		t.Fatalf("unexpected summary counts: runs=%+v tasks=%+v evidence=%+v", summary.RunCounts, summary.TaskCounts, summary.EvidenceCounts)
	}
	if summary.Runs[0].ID != run.ID || !summary.LatestRunAt.Equal(now) {
		t.Fatalf("unexpected summary runs: %+v latest=%v", summary.Runs, summary.LatestRunAt)
	}
}
