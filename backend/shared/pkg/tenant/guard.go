package tenant

import (
	"context"
	"fmt"
)

// Guard enforces tenant-scoped repository access.
// It wraps repository calls to automatically inject the tenant ID from context,
// ensuring cross-tenant data leakage is impossible at the infrastructure layer.
//
// Usage:
//
//	func (r *pgRepo) ListMemories(ctx context.Context, limit int) ([]*Memory, error) {
//	    tenantID, err := tenant.Guard(ctx)
//	    if err != nil { return nil, err }
//	    return r.db.Query("SELECT * FROM memories WHERE tenant_id = $1 LIMIT $2", tenantID, limit)
//	}
//
// SOL-ENT-004 / TASK-ENT-002
func Guard(ctx context.Context) (string, error) {
	tenantID, err := FromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("tenant guard: %w — repository access denied", err)
	}
	if tenantID == "" {
		return "", fmt.Errorf("tenant guard: empty tenant_id — cross-tenant data access prevented")
	}
	return tenantID, nil
}

// GuardProject returns both tenantID and projectID from context.
// Use this when queries need project-level scoping.
func GuardProject(ctx context.Context) (tenantID, projectID string, err error) {
	tenantID, err = Guard(ctx)
	if err != nil {
		return "", "", err
	}
	projectID, ok := ctx.Value(ProjectIDKey).(string)
	if !ok || projectID == "" {
		return tenantID, "", nil // projectID is optional
	}
	return tenantID, projectID, nil
}

// MustGuard panics if the context does not contain a valid tenant ID.
// Use only in scenarios where a missing tenant ID is a programming error (e.g., tests).
func MustGuard(ctx context.Context) string {
	tenantID, err := Guard(ctx)
	if err != nil {
		panic(fmt.Sprintf("tenant guard fatal: %v", err))
	}
	return tenantID
}

// WithTenant injects a tenant ID into a context (for testing or internal service calls).
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return InjectIntoContext(ctx, tenantID, "")
}

// WithTenantProject injects tenant + project IDs into a context.
func WithTenantProject(ctx context.Context, tenantID, projectID string) context.Context {
	return InjectIntoContext(ctx, tenantID, projectID)
}
