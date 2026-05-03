package policy

import "github.com/ruohao1/penta/internal/actions"

type Decision string

const (
	DecisionAllowed          Decision = "allowed"
	DecisionBlocked          Decision = "blocked"
	DecisionApprovalRequired Decision = "approval_required"
	DecisionRateLimited      Decision = "rate_limited"
)

type Evaluation struct {
	Decision Decision
	Reason   string
}

func Evaluate(spec actions.ActionSpec) Evaluation {
	switch spec.Permission {
	case actions.PermissionPassive:
		return Evaluation{Decision: DecisionAllowed, Reason: "passive action is allowed"}
	case actions.PermissionSafeProbe:
		return Evaluation{Decision: DecisionAllowed, Reason: "safe probe action is allowed"}
	case actions.PermissionActiveScan:
		return Evaluation{Decision: DecisionApprovalRequired, Reason: "active scan requires approval"}
	case actions.PermissionIntrusive:
		return Evaluation{Decision: DecisionApprovalRequired, Reason: "intrusive action requires approval"}
	case actions.PermissionManualOnly:
		return Evaluation{Decision: DecisionApprovalRequired, Reason: "manual-only action requires approval"}
	default:
		return Evaluation{Decision: DecisionBlocked, Reason: "unknown permission level"}
	}
}
