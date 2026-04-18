package sqlite

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(ctx context.Context, path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	database := &DB{db}
	if err := database.Init(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return database, nil
}
