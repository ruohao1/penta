package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/ruohao1/penta/internal/actions"
)

type Run struct {
	ID        string            `db:"id"`
	SessionID string            `db:"session_id"`
	Mode      string            `db:"mode"`
	Status    actions.RunStatus `db:"status"`
	CreatedAt time.Time         `db:"created_at"`
}

func (db *DB) CreateRun(ctx context.Context, run Run) error {
	if err := validateRun(run); err != nil {
		return err
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO runs (id, session_id, mode, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, run.ID, nullableString(run.SessionID), run.Mode, string(run.Status), run.CreatedAt)
	return err
}

func (db *DB) GetRun(ctx context.Context, id string) (*Run, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, session_id, mode, status, created_at
		FROM runs
		WHERE id = ?
	`, id)

	var run Run
	var sessionID sql.NullString
	if err := row.Scan(&run.ID, &sessionID, &run.Mode, &run.Status, &run.CreatedAt); err != nil {
		return nil, err
	}
	run.SessionID = stringFromNull(sessionID)
	return &run, nil
}

func (db *DB) ListRuns(ctx context.Context) ([]Run, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, mode, status, created_at
		FROM runs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var runs []Run
	for rows.Next() {
		var run Run
		var sessionID sql.NullString
		if err := rows.Scan(&run.ID, &sessionID, &run.Mode, &run.Status, &run.CreatedAt); err != nil {
			return nil, err
		}
		run.SessionID = stringFromNull(sessionID)
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (db *DB) LatestRun(ctx context.Context) (*Run, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, session_id, mode, status, created_at
		FROM runs
		ORDER BY created_at DESC
		LIMIT 1
	`)

	var run Run
	var sessionID sql.NullString
	if err := row.Scan(&run.ID, &sessionID, &run.Mode, &run.Status, &run.CreatedAt); err != nil {
		return nil, err
	}
	run.SessionID = stringFromNull(sessionID)
	return &run, nil
}

func (db *DB) ListRunsBySession(ctx context.Context, sessionID string) ([]Run, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, mode, status, created_at
		FROM runs
		WHERE session_id = ?
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var runs []Run
	for rows.Next() {
		var run Run
		var storedSessionID sql.NullString
		if err := rows.Scan(&run.ID, &storedSessionID, &run.Mode, &run.Status, &run.CreatedAt); err != nil {
			return nil, err
		}
		run.SessionID = stringFromNull(storedSessionID)
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (db *DB) UpdateRunStatus(ctx context.Context, id string, status actions.RunStatus) error {
	_, err := db.ExecContext(ctx, `
		UPDATE runs
		SET status = ?
		WHERE id = ?
	`, string(status), id)
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func stringFromNull(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
