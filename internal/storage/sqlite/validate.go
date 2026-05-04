package sqlite

import (
	"encoding/json"
	"errors"
	"strings"
)

func requireNonEmpty(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " is required")
	}
	return nil
}

func requireValidJSON(field string, value string) error {
	if err := requireNonEmpty(field, value); err != nil {
		return err
	}
	if !json.Valid([]byte(value)) {
		return errors.New(field + " must be valid JSON")
	}
	return nil
}

func validateRun(run Run) error {
	if err := requireNonEmpty("run.id", run.ID); err != nil {
		return err
	}
	if err := requireNonEmpty("run.mode", run.Mode); err != nil {
		return err
	}
	return requireNonEmpty("run.status", string(run.Status))
}

func validateSession(session Session) error {
	if err := requireNonEmpty("session.id", session.ID); err != nil {
		return err
	}
	if err := requireNonEmpty("session.name", session.Name); err != nil {
		return err
	}
	if err := requireNonEmpty("session.kind", string(session.Kind)); err != nil {
		return err
	}
	return requireNonEmpty("session.status", string(session.Status))
}

func validateScopeRule(rule ScopeRule) error {
	if err := requireNonEmpty("scope_rule.id", rule.ID); err != nil {
		return err
	}
	if err := requireNonEmpty("scope_rule.session_id", rule.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("scope_rule.effect", string(rule.Effect)); err != nil {
		return err
	}
	if err := requireNonEmpty("scope_rule.target_type", string(rule.TargetType)); err != nil {
		return err
	}
	return requireNonEmpty("scope_rule.value", rule.Value)
}

func validateTask(task Task) error {
	if err := requireNonEmpty("task.id", task.ID); err != nil {
		return err
	}
	if err := requireNonEmpty("task.run_id", task.RunID); err != nil {
		return err
	}
	if err := requireValidJSON("task.input_json", task.InputJSON); err != nil {
		return err
	}
	if err := requireNonEmpty("task.action_type", string(task.ActionType)); err != nil {
		return err
	}
	return requireNonEmpty("task.status", string(task.Status))
}

func validateArtifact(artifact Artifact) error {
	if err := requireNonEmpty("artifact.id", artifact.ID); err != nil {
		return err
	}
	if err := requireNonEmpty("artifact.task_id", artifact.TaskID); err != nil {
		return err
	}
	return requireNonEmpty("artifact.path", artifact.Path)
}

func validateEvidence(evidence Evidence) error {
	if err := requireNonEmpty("evidence.id", evidence.ID); err != nil {
		return err
	}
	if err := requireNonEmpty("evidence.run_id", evidence.RunID); err != nil {
		return err
	}
	if err := requireNonEmpty("evidence.task_id", evidence.TaskID); err != nil {
		return err
	}
	if err := requireNonEmpty("evidence.kind", evidence.Kind); err != nil {
		return err
	}
	return requireValidJSON("evidence.data_json", evidence.DataJSON)
}
