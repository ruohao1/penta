package sqlite

import (
	"context"
	"fmt"
)

const currentSchemaVersion = 2

type migration struct {
	version int
	apply   func(context.Context, *DB) error
}

var migrations = []migration{
	{version: 2, apply: migrateSessions},
}

func migrateSessions(ctx context.Context, db *DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS session_scope_rules (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			effect TEXT NOT NULL,
			target_type TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_session_scope_rules_session_id ON session_scope_rules(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_scope_rules_session_effect_type ON session_scope_rules(session_id, effect, target_type);
	`); err != nil {
		return err
	}
	if ok, err := db.columnExists(ctx, "runs", "session_id"); err != nil {
		return err
	} else if !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE runs ADD COLUMN session_id TEXT`); err != nil {
			return err
		}
	}
	_, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_runs_session_id_created_at ON runs(session_id, created_at)`)
	return err
}

func (db *DB) Migrate(ctx context.Context) error {
	version, err := db.schemaVersion(ctx)
	if err != nil {
		return err
	}
	if version > currentSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, currentSchemaVersion)
	}

	for _, migration := range migrations {
		if migration.version <= version {
			continue
		}
		if migration.version > currentSchemaVersion {
			break
		}
		if err := migration.apply(ctx, db); err != nil {
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if err := db.setSchemaVersion(ctx, migration.version); err != nil {
			return err
		}
		version = migration.version
	}

	if version == 0 {
		return db.setSchemaVersion(ctx, currentSchemaVersion)
	}
	return nil
}

func (db *DB) schemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func (db *DB) setSchemaVersion(ctx context.Context, version int) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return fmt.Errorf("set schema version %d: %w", version, err)
	}
	return nil
}

func (db *DB) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("inspect columns for %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
