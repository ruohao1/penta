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
	TaskCounts     map[actions.TaskStatus]int
	EvidenceCounts map[string]int
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
	}
	for _, task := range tasks {
		summary.TaskCounts[task.Status]++
	}
	for _, evidence := range evidenceRows {
		summary.EvidenceCounts[evidence.Kind]++
	}
	return summary, nil
}
