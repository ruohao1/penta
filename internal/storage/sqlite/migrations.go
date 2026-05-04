package sqlite

import (
	"context"
	"fmt"
)

const currentSchemaVersion = 1

type migration struct {
	version int
	apply   func(context.Context, *DB) error
}

var migrations = []migration{}

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
