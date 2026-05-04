package sqlite

import "context"

const schemaSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS runs (
	id TEXT PRIMARY KEY,
	mode TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	action_type TEXT NOT NULL,
	input_json TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS artifacts (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL,
	path TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS evidence (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	task_id TEXT NOT NULL,
	kind TEXT NOT NULL,
	data_json TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
	FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	seq INTEGER NOT NULL,
	event_type TEXT NOT NULL,
	entity_kind TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	payload_json TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_run_id_status ON tasks(run_id, status);
CREATE INDEX IF NOT EXISTS idx_artifacts_task_id ON artifacts(task_id);
CREATE INDEX IF NOT EXISTS idx_evidence_run_id_kind ON evidence(run_id, kind);
CREATE INDEX IF NOT EXISTS idx_evidence_task_id ON evidence(task_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_events_run_seq ON events(run_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_run_created_at ON events(run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_events_entity ON events(entity_kind, entity_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
`

func (db *DB) Init(ctx context.Context) error {
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	return db.Migrate(ctx)
}
