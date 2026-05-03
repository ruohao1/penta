package execute

import (
	"context"
	"strings"
	"testing"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestRegistryValidatesCurrentActions(t *testing.T) {
	if err := validateRegistry(registry()); err != nil {
		t.Fatalf("validate registry: %v", err)
	}
}

func TestValidateRegistryRejectsMissingSpecType(t *testing.T) {
	registered := map[actions.ActionType]RegisteredAction{
		actions.ActionSeedTarget: {
			Spec: actions.ActionSpec{
				Permission: actions.PermissionPassive,
				InputKind:  "seed_target.input",
			},
			Handler: noopHandler,
		},
	}

	err := validateRegistry(registered)
	if err == nil || !strings.Contains(err.Error(), "spec type is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRegistryRejectsMissingPermission(t *testing.T) {
	registered := map[actions.ActionType]RegisteredAction{
		actions.ActionSeedTarget: {
			Spec: actions.ActionSpec{
				Type:      actions.ActionSeedTarget,
				InputKind: "seed_target.input",
			},
			Handler: noopHandler,
		},
	}

	err := validateRegistry(registered)
	if err == nil || !strings.Contains(err.Error(), "permission is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRegistryRejectsMissingHandler(t *testing.T) {
	registered := map[actions.ActionType]RegisteredAction{
		actions.ActionSeedTarget: {
			Spec: actions.ActionSpec{
				Type:       actions.ActionSeedTarget,
				Permission: actions.PermissionPassive,
				InputKind:  "seed_target.input",
			},
		},
	}

	err := validateRegistry(registered)
	if err == nil || !strings.Contains(err.Error(), "handler is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRegistryRejectsSpecTypeMismatch(t *testing.T) {
	registered := map[actions.ActionType]RegisteredAction{
		actions.ActionSeedTarget: {
			Spec: actions.ActionSpec{
				Type:       actions.ActionProbeHTTP,
				Permission: actions.PermissionPassive,
				InputKind:  "seed_target.input",
			},
			Handler: noopHandler,
		},
	}

	err := validateRegistry(registered)
	if err == nil || !strings.Contains(err.Error(), "spec type mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func noopHandler(context.Context, *sqlite.DB, events.Sink, *sqlite.Task) error {
	return nil
}
