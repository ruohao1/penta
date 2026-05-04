package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
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
		return nil
	}
	scopeDecision := evaluateCandidateSessionScope(candidate, run.SessionID, rules)
	if !scopeDecision.Allowed {
		return f.appendEvent(ctx, events.Event{RunID: run.ID, EventType: events.EventCandidateBlocked, EntityKind: events.EntityRun, EntityID: run.ID, PayloadJSON: mustPayloadJSON(events.CandidateBlockedPayload{ActionType: candidate.ActionType, Reason: scopeDecision.Reason, Source: "session_scope", InputJSON: candidate.InputJSON}), CreatedAt: time.Now()})
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
