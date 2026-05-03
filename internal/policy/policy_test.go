package policy

import (
	"testing"

	"github.com/ruohao1/penta/internal/actions"
)

func TestEvaluateAllowsPassive(t *testing.T) {
	assertDecision(t, actions.PermissionPassive, DecisionAllowed)
}

func TestEvaluateAllowsSafeProbe(t *testing.T) {
	assertDecision(t, actions.PermissionSafeProbe, DecisionAllowed)
}

func TestEvaluateRequiresApprovalForActiveScan(t *testing.T) {
	assertDecision(t, actions.PermissionActiveScan, DecisionApprovalRequired)
}

func TestEvaluateRequiresApprovalForIntrusive(t *testing.T) {
	assertDecision(t, actions.PermissionIntrusive, DecisionApprovalRequired)
}

func TestEvaluateRequiresApprovalForManualOnly(t *testing.T) {
	assertDecision(t, actions.PermissionManualOnly, DecisionApprovalRequired)
}

func TestEvaluateBlocksUnknownPermission(t *testing.T) {
	assertDecision(t, actions.PermissionLevel(""), DecisionBlocked)
	assertDecision(t, actions.PermissionLevel("unknown"), DecisionBlocked)
}

func assertDecision(t *testing.T, permission actions.PermissionLevel, want Decision) {
	t.Helper()

	evaluation := Evaluate(actions.ActionSpec{Permission: permission})
	if evaluation.Decision != want {
		t.Fatalf("unexpected decision for %q: got %q want %q", permission, evaluation.Decision, want)
	}
	if evaluation.Reason == "" {
		t.Fatal("expected evaluation reason")
	}
}
