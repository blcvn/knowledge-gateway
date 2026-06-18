package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

type WriteIdentity struct {
	TenantID string
	AppID    string
}

type SessionScope struct {
	Identity      WriteIdentity
	Statements    []string
	Transactional bool
}

type SessionManager struct {
	DB        *sql.DB
	LastScope SessionScope
}

func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{DB: db}
}

func (m *SessionManager) Within(ctx context.Context, identity WriteIdentity, fn func(SessionScope) error) (SessionScope, error) {
	scope := SessionScope{
		Identity: identity,
		Statements: []string{
			"BEGIN",
			fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", identity.TenantID),
			fmt.Sprintf("SET LOCAL app.app_id = '%s'", identity.AppID),
			"COMMIT",
		},
		Transactional: true,
	}
	m.LastScope = scope

	if m.DB == nil {
		return scope, fn(scope)
	}

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return scope, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	for _, stmt := range scope.Statements[1:3] {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return scope, err
		}
	}

	if err := fn(scope); err != nil {
		return scope, err
	}
	if err := tx.Commit(); err != nil {
		return scope, err
	}
	rollback = false
	return scope, nil
}
