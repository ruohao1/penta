package scheduler

import (
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/model"
)

type CandidateTask struct {
	ActionType        actions.ActionType
	InputJSON         string
	Reason            string
	ParentEvidenceIDs []string
	Target            *model.TargetRef
}
