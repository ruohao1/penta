package events

import (
	"time"

	"github.com/ruohao1/penta/internal/actions"
)

type EventType string

const (
	EventRunCreated      EventType = "run.created"
	EventRunCompleted    EventType = "run.completed"
	EventRunFailed       EventType = "run.failed"
	EventActionRequested EventType = "action.requested"
	EventActionResolved  EventType = "action.resolved"
	EventTaskEnqueued    EventType = "task.enqueued"
	EventTaskStarted     EventType = "task.started"
	EventTaskCompleted   EventType = "task.completed"
	EventTaskFailed      EventType = "task.failed"
	EventEvidenceCreated EventType = "evidence.created"
)

type EntityKind string

const (
	EntityRun      EntityKind = "run"
	EntityTask     EntityKind = "task"
	EntityEvidence EntityKind = "evidence"
)

type Event struct {
	ID          string
	RunID       string
	Seq         int64
	EventType   EventType
	EntityKind  EntityKind
	EntityID    string
	PayloadJSON string
	CreatedAt   time.Time
}

type RunCreatedPayload struct {
	Mode string `json:"mode"`
}

type ActionRequestedPayload struct {
	Action actions.ActionType `json:"action"`
	Raw    string             `json:"raw"`
}

type ActionResolvedPayload struct {
	RequestedAction actions.ActionType   `json:"requested_action"`
	EnqueuedActions []actions.ActionType `json:"enqueued_actions"`
}

type TaskEnqueuedPayload struct {
	ActionType actions.ActionType `json:"action_type"`
	Status     actions.TaskStatus `json:"status"`
}

type TaskStartedPayload struct {
	ActionType actions.ActionType `json:"action_type"`
}

type TaskCompletedPayload struct {
	ActionType actions.ActionType `json:"action_type"`
}

type TaskFailedPayload struct {
	ActionType actions.ActionType `json:"action_type"`
	Error      string             `json:"error"`
}

type EvidenceCreatedPayload struct {
	Kind string `json:"kind"`
}
