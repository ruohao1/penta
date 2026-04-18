package sqlite

import (
	"context"
	"time"
)

type Run struct {
	ID        string    `db:"id"`
	Mode      string    `db:"mode"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

func (db *DB) CreateRun(ctx context.Context, run Run) error {
	if err := validateRun(run); err != nil {
		return err
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO runs (id, mode, status, created_at)
		VALUES (?, ?, ?, ?)
	`, run.ID, run.Mode, run.Status, run.CreatedAt)
	return err
}

func (db *DB) GetRun(ctx context.Context, id string) (*Run, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, mode, status, created_at
		FROM runs
		WHERE id = ?
	`, id)

	var run Run
	if err := row.Scan(&run.ID, &run.Mode, &run.Status, &run.CreatedAt); err != nil {
		return nil, err
	}
	return &run, nil
}

func (db *DB) UpdateRunStatus(ctx context.Context, id string, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE runs
		SET status = ?
		WHERE id = ?
	`, status, id)
	return err
}
