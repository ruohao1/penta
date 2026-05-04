package execute

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/policy"
	"github.com/ruohao1/penta/internal/scheduler"
	"github.com/ruohao1/penta/internal/scope"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

type Frontier struct {
	DB       *sqlite.DB
	Events   events.Sink
	Registry Registry
}

func (f Frontier) EnqueueCandidate(ctx context.Context, run *sqlite.Run, rules []sqlite.ScopeRule, candidate scheduler.CandidateTask) error {
	action, ok := f.registry()[candidate.ActionType]
	if !ok {
		return fmt.Errorf("derived unsupported action type: %s", candidate.ActionType)
	}
	evaluation := policy.Evaluate(action.Spec)
	if evaluation.Decision != policy.DecisionAllowed {
		return f.blockCandidate(ctx, run.ID, candidate, "policy", evaluation.Reason)
	}
	scopeDecision := evaluateCandidateSessionScope(candidate, run.SessionID, rules)
	if !scopeDecision.Allowed {
		return f.blockCandidate(ctx, run.ID, candidate, "session_scope", scopeDecision.Reason)
	}
	exists, err := f.DB.TaskExistsByRunActionInput(ctx, run.ID, candidate.ActionType, candidate.InputJSON)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return f.enqueueTask(ctx, run.ID, candidate.ActionType, candidate.InputJSON)
}

func (f Frontier) blockCandidate(ctx context.Context, runID string, candidate scheduler.CandidateTask, source, reason string) error {
	return f.appendEvent(ctx, events.Event{RunID: runID, EventType: events.EventCandidateBlocked, EntityKind: events.EntityRun, EntityID: runID, PayloadJSON: mustPayloadJSON(events.CandidateBlockedPayload{ActionType: candidate.ActionType, Reason: reason, Source: source, InputJSON: candidate.InputJSON}), CreatedAt: time.Now()})
}

func (f Frontier) registry() Registry {
	if f.Registry != nil {
		return f.Registry
	}
	return defaultRegistry
}

func (f Frontier) enqueueTask(ctx context.Context, runID string, actionType actions.ActionType, inputJSON string) error {
	task := sqlite.Task{
		ID:         "task_" + uuid.NewString(),
		RunID:      runID,
		ActionType: actionType,
		InputJSON:  inputJSON,
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := f.DB.CreateTask(ctx, task); err != nil {
		return err
	}
	return f.appendEvent(ctx, events.Event{
		RunID:       runID,
		EventType:   events.EventTaskEnqueued,
		EntityKind:  events.EntityTask,
		EntityID:    task.ID,
		PayloadJSON: mustPayloadJSON(events.TaskEnqueuedPayload{ActionType: task.ActionType, Status: task.Status}),
		CreatedAt:   time.Now(),
	})
}

func (f Frontier) appendEvent(ctx context.Context, evt events.Event) error {
	if f.Events == nil {
		return nil
	}
	return f.Events.Append(ctx, evt)
}

type candidateScopeDecision struct {
	Allowed bool
	Reason  string
}

func evaluateCandidateSessionScope(candidate scheduler.CandidateTask, sessionID string, rules []sqlite.ScopeRule) candidateScopeDecision {
	if sessionID == "" {
		return candidateScopeDecision{Allowed: true}
	}
	if candidate.Target == nil {
		return candidateScopeDecision{Allowed: true}
	}
	target, err := targets.Parse(candidate.Target.Value)
	if err != nil {
		return candidateScopeDecision{Allowed: false, Reason: fmt.Sprintf("candidate target could not be evaluated: %v", err)}
	}
	decision := scope.EvaluateTarget(target, rules)
	return candidateScopeDecision{Allowed: decision.Allowed, Reason: decision.Reason}
}
