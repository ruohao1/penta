package sqlite

import (
	"context"
	"time"
)

type Task struct {
	ID         string    `db:"id"`
	RunID      string    `db:"run_id"`
	ActionType string    `db:"action_type"`
	InputJSON  string    `db:"input_json"`
	Status     string    `db:"status"`
	CreatedAt  time.Time `db:"created_at"`
}

func (db *DB) CreateTask(ctx context.Context, task Task) error {
	if err := validateTask(task); err != nil {
		return err
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO tasks (id, run_id, action_type, input_json, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, task.ID, task.RunID, task.ActionType, task.InputJSON, task.Status, task.CreatedAt)
	return err
}

func (db *DB) ListTasksByRun(ctx context.Context, runID string) ([]Task, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, run_id, action_type, input_json, status, created_at
		FROM tasks
		WHERE run_id = ?
	`, runID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.RunID, &task.ActionType, &task.InputJSON, &task.Status, &task.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (db *DB) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE tasks
		SET status = ?
		WHERE id = ?
	`, status, taskID)
	return err
}
