package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	fetchroot "github.com/ruohao1/penta/internal/actions/fetch_root"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	resolvedns "github.com/ruohao1/penta/internal/actions/resolve_dns"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/policy"
	"github.com/ruohao1/penta/internal/scheduler"
	"github.com/ruohao1/penta/internal/scope"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

type Executor struct {
	DB     *sqlite.DB
	RunID  string
	Events events.Sink
}

func (e *Executor) RunTask(ctx context.Context, taskID string) error {
	task, err := e.DB.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if err := e.DB.UpdateTaskStatus(ctx, taskID, actions.TaskStatusRunning); err != nil {
		return err
	}
	if err := e.appendEvent(ctx, events.Event{
		RunID:       task.RunID,
		EventType:   events.EventTaskStarted,
		EntityKind:  events.EntityTask,
		EntityID:    task.ID,
		PayloadJSON: mustPayloadJSON(events.TaskStartedPayload{ActionType: task.ActionType}),
		CreatedAt:   time.Now(),
	}); err != nil {
		return err
	}

	if err := e.executeTask(ctx, task); err != nil {
		if markErr := e.DB.UpdateTaskStatus(ctx, taskID, actions.TaskStatusFailed); markErr != nil {
			return fmt.Errorf("%w: mark failed: %v", err, markErr)
		}
		if emitErr := e.appendEvent(ctx, events.Event{
			RunID:       task.RunID,
			EventType:   events.EventTaskFailed,
			EntityKind:  events.EntityTask,
			EntityID:    task.ID,
			PayloadJSON: mustPayloadJSON(events.TaskFailedPayload{ActionType: task.ActionType, Error: err.Error()}),
			CreatedAt:   time.Now(),
		}); emitErr != nil {
			return fmt.Errorf("%w: append task.failed: %v", err, emitErr)
		}
		return err
	}

	if err := e.DB.UpdateTaskStatus(ctx, task.ID, actions.TaskStatusCompleted); err != nil {
		return err
	}
	if err := e.appendEvent(ctx, events.Event{
		RunID:       task.RunID,
		EventType:   events.EventTaskCompleted,
		EntityKind:  events.EntityTask,
		EntityID:    task.ID,
		PayloadJSON: mustPayloadJSON(events.TaskCompletedPayload{ActionType: task.ActionType}),
		CreatedAt:   time.Now(),
	}); err != nil {
		return err
	}
	if err := e.enqueueFollowOns(ctx, task); err != nil {
		return err
	}

	return nil
}

func (e *Executor) RunOnce(ctx context.Context) (bool, error) {
	var (
		task *sqlite.Task
		err  error
	)
	if e.RunID != "" {
		task, err = e.DB.NextPendingTaskByRun(ctx, e.RunID)
	} else {
		task, err = e.DB.NextPendingTask(ctx)
	}
	if err != nil {
		return false, err
	}
	if task == nil {
		return false, nil
	}
	if err := e.RunTask(ctx, task.ID); err != nil {
		return true, err
	}
	return true, nil
}

func (e *Executor) RunUntilIdle(ctx context.Context) error {
	for {
		progressed, err := e.RunOnce(ctx)
		if err != nil {
			return err
		}
		if !progressed {
			return nil
		}
	}
}

func (e *Executor) executeTask(ctx context.Context, task *sqlite.Task) error {
	registered := registry()
	if err := validateRegistry(registered); err != nil {
		return err
	}
	action, ok := registered[task.ActionType]
	if !ok {
		return fmt.Errorf("unsupported action type: %s", task.ActionType)
	}
	return action.Handler(ctx, e.DB, e.Events, task)
}

func (e *Executor) enqueueFollowOns(ctx context.Context, task *sqlite.Task) error {
	evidenceRows, err := e.DB.ListEvidenceByTask(ctx, task.ID)
	if err != nil {
		return err
	}
	registered := registry()
	run, err := e.DB.GetRun(ctx, task.RunID)
	if err != nil {
		return err
	}
	var scopeRules []sqlite.ScopeRule
	if run.SessionID != "" {
		scopeRules, err = e.DB.ListScopeRulesBySession(ctx, run.SessionID)
		if err != nil {
			return err
		}
	}
	for _, evidence := range evidenceRows {
		candidates, err := scheduler.DeriveFromEvidence(evidence)
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			action, ok := registered[candidate.ActionType]
			if !ok {
				return fmt.Errorf("derived unsupported action type: %s", candidate.ActionType)
			}
			evaluation := policy.Evaluate(action.Spec)
			if evaluation.Decision != policy.DecisionAllowed {
				continue
			}
			scopeDecision := evaluateCandidateSessionScope(candidate, run.SessionID, scopeRules)
			if !scopeDecision.Allowed {
				if err := e.appendEvent(ctx, events.Event{RunID: task.RunID, EventType: events.EventCandidateBlocked, EntityKind: events.EntityRun, EntityID: task.RunID, PayloadJSON: mustPayloadJSON(events.CandidateBlockedPayload{ActionType: candidate.ActionType, Reason: scopeDecision.Reason, Source: "session_scope", InputJSON: candidate.InputJSON}), CreatedAt: time.Now()}); err != nil {
					return err
				}
				continue
			}
			exists, err := e.DB.TaskExistsByRunActionInput(ctx, task.RunID, candidate.ActionType, candidate.InputJSON)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			if err := e.enqueueTask(ctx, task.RunID, candidate.ActionType, candidate.InputJSON); err != nil {
				return err
			}
		}
	}
	return nil
}

type candidateScopeDecision struct {
	Allowed bool
	Reason  string
}

func evaluateCandidateSessionScope(candidate scheduler.CandidateTask, sessionID string, rules []sqlite.ScopeRule) candidateScopeDecision {
	if sessionID == "" {
		return candidateScopeDecision{Allowed: true}
	}
	target, ok, err := targetFromCandidate(candidate)
	if err != nil {
		return candidateScopeDecision{Allowed: false, Reason: fmt.Sprintf("candidate target could not be evaluated: %v", err)}
	}
	if !ok {
		return candidateScopeDecision{Allowed: true}
	}
	decision := scope.EvaluateTarget(target, rules)
	return candidateScopeDecision{Allowed: decision.Allowed, Reason: decision.Reason}
}

func targetFromCandidate(candidate scheduler.CandidateTask) (targets.Target, bool, error) {
	switch candidate.ActionType {
	case actions.ActionProbeHTTP:
		var input probehttp.Input
		if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
			return nil, false, err
		}
		target, err := targets.Parse(input.Value)
		return target, true, err
	case actions.ActionResolveDNS:
		var input resolvedns.Input
		if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
			return nil, false, err
		}
		target, err := targets.Parse(input.Domain)
		return target, true, err
	case actions.ActionFetchRoot:
		var input fetchroot.Input
		if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
			return nil, false, err
		}
		target, err := targets.Parse(serviceURL(input))
		return target, true, err
	default:
		return nil, false, nil
	}
}

func serviceURL(input fetchroot.Input) string {
	scheme := input.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := input.Host
	if input.Port > 0 {
		host = net.JoinHostPort(input.Host, strconv.Itoa(input.Port))
	}
	return (&url.URL{Scheme: scheme, Host: host, Path: "/"}).String()
}

func (e *Executor) appendEvent(ctx context.Context, evt events.Event) error {
	if e == nil || e.Events == nil {
		return nil
	}
	return e.Events.Append(ctx, evt)
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
