package viewmodel

import (
	"context"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type SessionSummary struct {
	Session        sqlite.Session
	ScopeRules     []sqlite.ScopeRule
	Runs           []RunListItem
	RunCounts      map[actions.RunStatus]int
	TaskCounts     map[actions.TaskStatus]int
	EvidenceCounts map[string]int
	LatestRunAt    time.Time
}

type RunListItem struct {
	ID        string
	Status    actions.RunStatus
	Mode      string
	CreatedAt time.Time
}

func BuildSessionSummary(ctx context.Context, db *sqlite.DB, sessionID string) (*SessionSummary, error) {
	session, err := db.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	rules, err := db.ListScopeRulesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	runs, err := db.ListRunsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	summary := &SessionSummary{
		Session:        *session,
		ScopeRules:     rules,
		Runs:           make([]RunListItem, 0, len(runs)),
		RunCounts:      map[actions.RunStatus]int{},
		TaskCounts:     map[actions.TaskStatus]int{},
		EvidenceCounts: map[string]int{},
	}
	for _, run := range runs {
		summary.Runs = append(summary.Runs, RunListItem{ID: run.ID, Status: run.Status, Mode: run.Mode, CreatedAt: run.CreatedAt})
		summary.RunCounts[run.Status]++
		if run.CreatedAt.After(summary.LatestRunAt) {
			summary.LatestRunAt = run.CreatedAt
		}
		tasks, err := db.ListTasksByRun(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		for _, task := range tasks {
			summary.TaskCounts[task.Status]++
		}
		evidenceRows, err := db.ListEvidenceByRun(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		for _, evidence := range evidenceRows {
			summary.EvidenceCounts[evidence.Kind]++
		}
	}
	return summary, nil
}
