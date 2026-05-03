package events

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type SQLiteSink struct {
	DB *sqlite.DB
}

func (s *SQLiteSink) Append(ctx context.Context, evt Event) error {
	if s == nil || s.DB == nil {
		return nil
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var nextSeq int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(seq), 0) + 1
		FROM events
		WHERE run_id = ?
	`, evt.RunID).Scan(&nextSeq); err != nil {
		return err
	}

	if evt.ID == "" {
		evt.ID = "event_" + uuid.NewString()
	}
	evt.Seq = nextSeq

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO events (id, run_id, seq, event_type, entity_kind, entity_id, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, evt.ID, evt.RunID, evt.Seq, string(evt.EventType), string(evt.EntityKind), evt.EntityID, evt.PayloadJSON, evt.CreatedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteSink) ListByRunSinceSeq(ctx context.Context, runID string, seq int64, limit int) ([]Event, error) {
	if s == nil || s.DB == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, run_id, seq, event_type, entity_kind, entity_id, payload_json, created_at
		FROM events
		WHERE run_id = ? AND seq > ?
		ORDER BY seq ASC
		LIMIT ?
	`, runID, seq, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Event
	for rows.Next() {
		var evt Event
		var eventType string
		var entityKind string
		if err := rows.Scan(&evt.ID, &evt.RunID, &evt.Seq, &eventType, &entityKind, &evt.EntityID, &evt.PayloadJSON, &evt.CreatedAt); err != nil {
			return nil, err
		}
		evt.EventType = EventType(eventType)
		evt.EntityKind = EntityKind(entityKind)
		out = append(out, evt)
	}
	if err := rows.Err(); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return out, nil
}
