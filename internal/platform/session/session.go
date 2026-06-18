package session

import (
	"context"
	"database/sql"
)

type WriteIdentity struct {
	TenantID string
	AppID    string
}

type SessionScope struct {
	Identity      WriteIdentity
	Statements    []string
	Transactional bool
	Tx            *sql.Tx
}

type Manager interface {
	Within(ctx context.Context, identity WriteIdentity, fn func(SessionScope) error) (SessionScope, error)
}
