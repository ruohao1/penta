package scheduler

import "github.com/ruohao1/penta/internal/actions"

type CandidateTask struct {
	ActionType        actions.ActionType
	InputJSON         string
	Reason            string
	ParentEvidenceIDs []string
}
