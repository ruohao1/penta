package viewmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type RunList struct {
	Runs []IndexedRun
}

type IndexedRun struct {
	Index     int
	ID        string
	Mode      string
	Status    actions.RunStatus
	Session   string
	CreatedAt time.Time
}

func BuildRunList(ctx context.Context, db *sqlite.DB) (*RunList, error) {
	runs, err := db.ListRuns(ctx)
	if err != nil {
		return nil, err
	}

	list := &RunList{Runs: make([]IndexedRun, 0, len(runs))}
	for i, run := range runs {
		sessionDisplay := "-"
		if run.SessionID != "" {
			session, err := db.GetSession(ctx, run.SessionID)
			if err != nil {
				return nil, err
			}
			sessionDisplay = fmt.Sprintf("%s (%s)", session.Name, session.Kind)
		}
		list.Runs = append(list.Runs, IndexedRun{Index: i + 1, ID: run.ID, Mode: run.Mode, Status: run.Status, Session: sessionDisplay, CreatedAt: run.CreatedAt})
	}

	return list, nil
}
