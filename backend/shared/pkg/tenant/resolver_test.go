package tenant_test

import (
	"context"
	"testing"

	"vnp-memory/shared/pkg/tenant"

	"google.golang.org/grpc/metadata"
)

func TestInjectAndFromContext(t *testing.T) {
	expectedTenantID := "tenant-12345"

	// 1. Create context with tenant
	ctx := tenant.InjectIntoContext(context.Background(), expectedTenantID, "")

	// 2. Extract tenant
	actualTenantID, err := tenant.FromContext(ctx)

	// 3. Assertions
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if actualTenantID != expectedTenantID {
		t.Errorf("Expected tenant ID %s, got: %s", expectedTenantID, actualTenantID)
	}
}

func TestFromContext_MissingTenant(t *testing.T) {
	// Empty context
	ctx := context.Background()

	_, err := tenant.FromContext(ctx)
	if err == nil {
		t.Errorf("Expected error for missing tenant, got nil")
	}
}

func TestExtractFromMetadata(t *testing.T) {
	expectedTenantID := "tenant-grpc-888"

	// 1. Create a dummy metadata containing the x-tenant-id header
	md := metadata.Pairs(string(tenant.TenantIDKey), expectedTenantID)

	// 2. Attach metadata to incoming context (as gRPC would do)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// 3. Extract it
	actualTenantID, _, err := tenant.ExtractFromMetadata(ctx)

	// 4. Assertions
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if actualTenantID != expectedTenantID {
		t.Errorf("Expected %s, got %s", expectedTenantID, actualTenantID)
	}
}

func TestExtractFromMetadata_MissingHeader(t *testing.T) {
	// Metadata with wrong or missing header
	md := metadata.Pairs("some-other-header", "123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, _, err := tenant.ExtractFromMetadata(ctx)
	if err == nil {
		t.Errorf("Expected error for missing tenant header, got nil")
	}
}
