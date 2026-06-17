package postgres

import "fmt"

type WriteIdentity struct {
	TenantID string
	AppID    string
}

type SessionScope struct {
	Identity   WriteIdentity
	Statements []string
}

type SessionManager struct {
	LastScope SessionScope
}

func (m *SessionManager) Begin(identity WriteIdentity) SessionScope {
	scope := SessionScope{
		Identity: identity,
		Statements: []string{
			fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", identity.TenantID),
			fmt.Sprintf("SET LOCAL app.app_id = '%s'", identity.AppID),
		},
	}
	m.LastScope = scope
	return scope
}
