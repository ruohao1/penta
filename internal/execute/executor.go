package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/storage/sqlite"
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
	handler, ok := handlers()[task.ActionType]
	if !ok {
		return fmt.Errorf("unsupported action type: %s", task.ActionType)
	}
	return handler(ctx, e.DB, e.Events, task)
}

func (e *Executor) enqueueFollowOns(ctx context.Context, task *sqlite.Task) error {
	return nil
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
