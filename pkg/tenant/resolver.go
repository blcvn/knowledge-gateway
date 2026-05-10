// Package tenant provides a unified tenant context resolver for all VNP Memory engines.
//
// Problem: 6 engines use 6 different isolation key names:
//   - Cognee:      tenant_id
//   - Graphiti:    group_id
//   - Memobase:    project_id
//   - OpenViking:  account_id
//   - Zep:         project_uuid
//   - Supermemory: org_id
//
// Solution: Gateway resolves the canonical tenant_id from JWT/APIKey once,
// then sets both x-tenant-id and engine-specific aliases in gRPC metadata.
// Each engine reads from TenantContext instead of raw metadata.
//
// See: specs/technical/TECH-002-unify-tenant-isolation-keys.md
package tenant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// Engine identifies a VNP Memory engine.
type Engine string

const (
	EngineCognee      Engine = "cognee"
	EngineGraphiti    Engine = "graphiti"
	EngineMemobase    Engine = "memobase"
	EngineOpenViking  Engine = "openviking"
	EngineZep         Engine = "zep"
	EngineSupermemory Engine = "supermemory"
	EnginePlatform    Engine = "platform"
)

// engineKeyMap maps each engine to its legacy isolation key name.
var engineKeyMap = map[Engine]string{
	EngineCognee:      "tenant_id",
	EngineGraphiti:    "group_id",
	EngineMemobase:    "project_id",
	EngineOpenViking:  "account_id",
	EngineZep:         "project_uuid",
	EngineSupermemory: "org_id",
	EnginePlatform:    "tenant_id",
}

// TenantContext carries the resolved tenant identity across service boundaries.
type TenantContext struct {
	// TenantID is the canonical, unified tenant identifier.
	TenantID uuid.UUID `json:"tenant_id"`

	// Aliases maps engine names to their legacy key values.
	// Gateway populates this from the tenant registry.
	Aliases map[Engine]string `json:"aliases,omitempty"`
}

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey struct{}

var tenantCtxKey = contextKey{}

// MetadataKey is the canonical gRPC metadata key for tenant ID.
const MetadataKey = "x-tenant-id"

// WithTenantContext injects a TenantContext into a Go context.
func WithTenantContext(ctx context.Context, tc *TenantContext) context.Context {
	return context.WithValue(ctx, tenantCtxKey, tc)
}

// FromContext extracts the TenantContext from a Go context.
// Returns an error if no tenant context is set.
func FromContext(ctx context.Context) (*TenantContext, error) {
	tc, ok := ctx.Value(tenantCtxKey).(*TenantContext)
	if !ok || tc == nil {
		return nil, fmt.Errorf("tenant: no TenantContext in context")
	}
	return tc, nil
}

// FromGRPCMetadata extracts the tenant ID from gRPC metadata.
// Used by engine-side interceptors to build TenantContext.
func FromGRPCMetadata(ctx context.Context) (*TenantContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant: no gRPC metadata in context")
	}

	vals := md.Get(MetadataKey)
	if len(vals) == 0 {
		return nil, fmt.Errorf("tenant: missing %s in gRPC metadata", MetadataKey)
	}

	tenantID, err := uuid.Parse(vals[0])
	if err != nil {
		return nil, fmt.Errorf("tenant: invalid %s value %q: %w", MetadataKey, vals[0], err)
	}

	tc := &TenantContext{
		TenantID: tenantID,
		Aliases:  make(map[Engine]string),
	}

	// Read engine-specific aliases if present
	for engine, key := range engineKeyMap {
		if aliasVals := md.Get("x-" + key); len(aliasVals) > 0 {
			tc.Aliases[engine] = aliasVals[0]
		}
	}

	return tc, nil
}

// EngineKey returns the legacy isolation key value for a specific engine.
// Falls back to the canonical TenantID string if no alias is configured.
func (tc *TenantContext) EngineKey(engine Engine) string {
	if alias, ok := tc.Aliases[engine]; ok && alias != "" {
		return alias
	}
	return tc.TenantID.String()
}

// ToGRPCMetadata converts TenantContext into outgoing gRPC metadata.
// Used by gateway to propagate tenant identity to downstream services.
func (tc *TenantContext) ToGRPCMetadata() metadata.MD {
	md := metadata.New(map[string]string{
		MetadataKey: tc.TenantID.String(),
	})
	for engine, alias := range tc.Aliases {
		key, ok := engineKeyMap[engine]
		if ok && alias != "" {
			md.Set("x-"+key, alias)
		}
	}
	return md
}
