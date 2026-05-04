package viewmodel

import (
	"context"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type RunSummary struct {
	RunID          string
	Status         actions.RunStatus
	DBPath         string
	Session        *sqlite.Session
	ScopeRules     []sqlite.ScopeRule
	TaskCounts     map[actions.TaskStatus]int
	EvidenceCounts map[string]int
	Evidence       []EvidenceSummary
}

type EvidenceSummary struct {
	ID      string
	Kind    string
	Label   string
	URL     string
	Details []string
}

func BuildRunSummary(ctx context.Context, db *sqlite.DB, runID, dbPath string) (*RunSummary, error) {
	run, err := db.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	tasks, err := db.ListTasksByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	evidenceRows, err := db.ListEvidenceByRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	summary := &RunSummary{
		RunID:          run.ID,
		Status:         run.Status,
		DBPath:         dbPath,
		TaskCounts:     map[actions.TaskStatus]int{},
		EvidenceCounts: map[string]int{},
		Evidence:       make([]EvidenceSummary, 0, len(evidenceRows)),
	}
	if run.SessionID != "" {
		session, err := db.GetSession(ctx, run.SessionID)
		if err != nil {
			return nil, err
		}
		summary.Session = session
		rules, err := db.ListScopeRulesBySession(ctx, run.SessionID)
		if err != nil {
			return nil, err
		}
		summary.ScopeRules = rules
	}
	for _, task := range tasks {
		summary.TaskCounts[task.Status]++
	}
	for _, evidence := range evidenceRows {
		summary.EvidenceCounts[evidence.Kind]++
		evidenceSummary, err := EvidenceSummaryFor(evidence)
		if err != nil {
			return nil, err
		}
		summary.Evidence = append(summary.Evidence, evidenceSummary)
	}
	return summary, nil
}
