package sqlite

import (
	"context"
	"time"
)

type Artifact struct {
	ID        string    `db:"id"`
	TaskID    string    `db:"task_id"`
	Path      string    `db:"path"`
	CreatedAt time.Time `db:"created_at"`
}

func (db *DB) CreateArtifact(ctx context.Context, artifact Artifact) error {
	if err := validateArtifact(artifact); err != nil {
		return err
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO artifacts (id, task_id, path, created_at)
		VALUES (?, ?, ?, ?)
	`, artifact.ID, artifact.TaskID, artifact.Path, artifact.CreatedAt)
	return err
}

func (db *DB) GetArtifact(ctx context.Context, id string) (*Artifact, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, task_id, path, created_at
		FROM artifacts
		WHERE id = ?
	`, id)
	var artifact Artifact
	if err := row.Scan(&artifact.ID, &artifact.TaskID, &artifact.Path, &artifact.CreatedAt); err != nil {
		return nil, err
	}
	return &artifact, nil
}

func (db *DB) ListArtifactsByTask(ctx context.Context, taskID string) ([]Artifact, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, task_id, path, created_at
		FROM artifacts
		WHERE task_id = ?
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var artifacts []Artifact
	for rows.Next() {
		var artifact Artifact
		if err := rows.Scan(&artifact.ID, &artifact.TaskID, &artifact.Path, &artifact.CreatedAt); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func (db *DB) ListArtifactsByRun(ctx context.Context, runID string) ([]Artifact, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT artifacts.id, artifacts.task_id, artifacts.path, artifacts.created_at
		FROM artifacts
		JOIN tasks ON artifacts.task_id = tasks.id
		WHERE tasks.run_id = ?
		ORDER BY artifacts.created_at ASC
	`, runID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var artifacts []Artifact
	for rows.Next() {
		var artifact Artifact
		if err := rows.Scan(&artifact.ID, &artifact.TaskID, &artifact.Path, &artifact.CreatedAt); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}
