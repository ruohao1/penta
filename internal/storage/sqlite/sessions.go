package sqlite

import (
	"context"
	"time"
)

type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusArchived SessionStatus = "archived"
)

type SessionKind string

const (
	SessionKindBugBounty SessionKind = "bugbounty"
	SessionKindCTF       SessionKind = "ctf"
	SessionKindPentest   SessionKind = "pentest"
	SessionKindLab       SessionKind = "lab"
	SessionKindOther     SessionKind = "other"
)

type ScopeEffect string

const (
	ScopeEffectInclude ScopeEffect = "include"
	ScopeEffectExclude ScopeEffect = "exclude"
)

type ScopeTargetType string

const (
	ScopeTargetDomain   ScopeTargetType = "domain"
	ScopeTargetIP       ScopeTargetType = "ip"
	ScopeTargetCIDR     ScopeTargetType = "cidr"
	ScopeTargetURL      ScopeTargetType = "url"
	ScopeTargetService  ScopeTargetType = "service"
	ScopeTargetWildcard ScopeTargetType = "wildcard"
)

type Session struct {
	ID        string        `db:"id"`
	Name      string        `db:"name"`
	Kind      SessionKind   `db:"kind"`
	Status    SessionStatus `db:"status"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}

type ScopeRule struct {
	ID         string          `db:"id"`
	SessionID  string          `db:"session_id"`
	Effect     ScopeEffect     `db:"effect"`
	TargetType ScopeTargetType `db:"target_type"`
	Value      string          `db:"value"`
	CreatedAt  time.Time       `db:"created_at"`
}

func (db *DB) CreateSession(ctx context.Context, session Session) error {
	if err := validateSession(session); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO sessions (id, name, kind, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, session.ID, session.Name, string(session.Kind), string(session.Status), session.CreatedAt, session.UpdatedAt)
	return err
}

func (db *DB) GetSession(ctx context.Context, id string) (*Session, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, name, kind, status, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`, id)
	var session Session
	if err := row.Scan(&session.ID, &session.Name, &session.Kind, &session.Status, &session.CreatedAt, &session.UpdatedAt); err != nil {
		return nil, err
	}
	return &session, nil
}

func (db *DB) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, kind, status, created_at, updated_at
		FROM sessions
		ORDER BY updated_at DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.Name, &session.Kind, &session.Status, &session.CreatedAt, &session.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (db *DB) ArchiveSession(ctx context.Context, id string, updatedAt time.Time) error {
	_, err := db.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, string(SessionStatusArchived), updatedAt, id)
	return err
}

func (db *DB) CreateScopeRule(ctx context.Context, rule ScopeRule) error {
	if err := validateScopeRule(rule); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO session_scope_rules (id, session_id, effect, target_type, value, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, rule.ID, rule.SessionID, string(rule.Effect), string(rule.TargetType), rule.Value, rule.CreatedAt)
	return err
}

func (db *DB) ListScopeRulesBySession(ctx context.Context, sessionID string) ([]ScopeRule, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, effect, target_type, value, created_at
		FROM session_scope_rules
		WHERE session_id = ?
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var rules []ScopeRule
	for rows.Next() {
		var rule ScopeRule
		if err := rows.Scan(&rule.ID, &rule.SessionID, &rule.Effect, &rule.TargetType, &rule.Value, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (db *DB) DeleteScopeRule(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM session_scope_rules WHERE id = ?`, id)
	return err
}
