package sqlite

import (
	"context"
	"time"
)

type Evidence struct {
	ID        string    `db:"id"`
	RunID     string    `db:"run_id"`
	TaskID    string    `db:"task_id"`
	Kind      string    `db:"kind"`
	DataJSON  string    `db:"data_json"`
	CreatedAt time.Time `db:"created_at"`
}

func (db *DB) CreateEvidence(ctx context.Context, evidence Evidence) error {
	if err := validateEvidence(evidence); err != nil {
		return err
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO evidence (id, run_id, task_id, kind, data_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, evidence.ID, evidence.RunID, evidence.TaskID, evidence.Kind, evidence.DataJSON, evidence.CreatedAt)
	return err
}

func (db *DB) GetEvidence(ctx context.Context, id string) (*Evidence, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, run_id, task_id, kind, data_json, created_at
		FROM evidence
		WHERE id = ?
	`, id)

	var evidence Evidence
	if err := row.Scan(&evidence.ID, &evidence.RunID, &evidence.TaskID, &evidence.Kind, &evidence.DataJSON, &evidence.CreatedAt); err != nil {
		return nil, err
	}
	return &evidence, nil
}

func (db *DB) ListEvidenceByRun(ctx context.Context, runID string) ([]Evidence, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, run_id, task_id, kind, data_json, created_at
		FROM evidence
		WHERE run_id = ?
		ORDER BY created_at ASC
	`, runID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var evidence []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(&e.ID, &e.RunID, &e.TaskID, &e.Kind, &e.DataJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		evidence = append(evidence, e)
	}
	return evidence, rows.Err()
}

func (db *DB) ListEvidenceByRunAndKind(ctx context.Context, runID, kind string) ([]Evidence, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, run_id, task_id, kind, data_json, created_at
		FROM evidence
		WHERE run_id = ? AND kind = ?
		ORDER BY created_at ASC
	`, runID, kind)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var evidence []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(&e.ID, &e.RunID, &e.TaskID, &e.Kind, &e.DataJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		evidence = append(evidence, e)
	}
	return evidence, rows.Err()
}

func (db *DB) ListEvidenceByTask(ctx context.Context, taskID string) ([]Evidence, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, run_id, task_id, kind, data_json, created_at
		FROM evidence
		WHERE task_id = ?
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var evidence []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(&e.ID, &e.RunID, &e.TaskID, &e.Kind, &e.DataJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		evidence = append(evidence, e)
	}
	return evidence, rows.Err()
}
