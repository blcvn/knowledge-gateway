package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func TestTenantContextRoundTrip(t *testing.T) {
	tenantID := uuid.New()
	tc := &TenantContext{
		TenantID: tenantID,
		Aliases: map[Engine]string{
			EngineGraphiti:   "custom-group-1",
			EngineOpenViking: "custom-account-1",
		},
	}

	// Test context injection/extraction
	ctx := WithTenantContext(context.Background(), tc)
	got, err := FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext: %v", err)
	}
	if got.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", got.TenantID, tenantID)
	}

	// Test engine key resolution
	if key := got.EngineKey(EngineGraphiti); key != "custom-group-1" {
		t.Errorf("EngineKey(Graphiti) = %q, want %q", key, "custom-group-1")
	}
	if key := got.EngineKey(EngineCognee); key != tenantID.String() {
		t.Errorf("EngineKey(Cognee) = %q, want %q (fallback)", key, tenantID.String())
	}

	// Test gRPC metadata round-trip
	md := tc.ToGRPCMetadata()
	if v := md.Get(MetadataKey); len(v) == 0 || v[0] != tenantID.String() {
		t.Errorf("metadata %s = %v, want %s", MetadataKey, v, tenantID.String())
	}
	if v := md.Get("x-group_id"); len(v) == 0 || v[0] != "custom-group-1" {
		t.Errorf("metadata x-group_id = %v, want %q", v, "custom-group-1")
	}
}

func TestFromGRPCMetadata(t *testing.T) {
	tenantID := uuid.New()
	md := metadata.New(map[string]string{
		MetadataKey:    tenantID.String(),
		"x-group_id":  "grp-123",
		"x-account_id": "acc-456",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	tc, err := FromGRPCMetadata(ctx)
	if err != nil {
		t.Fatalf("FromGRPCMetadata: %v", err)
	}
	if tc.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", tc.TenantID, tenantID)
	}
	if tc.Aliases[EngineGraphiti] != "grp-123" {
		t.Errorf("Graphiti alias = %q, want %q", tc.Aliases[EngineGraphiti], "grp-123")
	}
	if tc.Aliases[EngineOpenViking] != "acc-456" {
		t.Errorf("OpenViking alias = %q, want %q", tc.Aliases[EngineOpenViking], "acc-456")
	}
}

func TestFromContext_NoTenant(t *testing.T) {
	_, err := FromContext(context.Background())
	if err == nil {
		t.Error("FromContext should fail when no TenantContext is set")
	}
}

func TestFromGRPCMetadata_Missing(t *testing.T) {
	// No metadata at all
	_, err := FromGRPCMetadata(context.Background())
	if err == nil {
		t.Error("should fail without gRPC metadata")
	}

	// Metadata but no tenant key
	md := metadata.New(map[string]string{"x-other": "value"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err = FromGRPCMetadata(ctx)
	if err == nil {
		t.Error("should fail without x-tenant-id")
	}
}
