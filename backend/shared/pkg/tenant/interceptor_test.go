package tenant_test

import (
	"context"
	"testing"

	"vnp-memory/shared/pkg/tenant"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptor_ValidTenant(t *testing.T) {
	interceptor := tenant.UnaryServerInterceptor()
	expectedTenantID := "tenant-valid-999"

	// Setup incoming context with metadata
	md := metadata.Pairs(string(tenant.TenantIDKey), expectedTenantID)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Mock handler
	invoked := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		invoked = true
		// Verify the tenant ID was injected into the context passed to the handler
		actualTenantID, err := tenant.FromContext(ctx)
		if err != nil {
			t.Errorf("Expected tenant ID in context, got error: %v", err)
		}
		if actualTenantID != expectedTenantID {
			t.Errorf("Expected injected tenant ID %s, got %s", expectedTenantID, actualTenantID)
		}
		return "success", nil
	}

	// Execute interceptor
	res, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if res != "success" {
		t.Errorf("Expected 'success', got %v", res)
	}
	if !invoked {
		t.Errorf("Handler was never invoked")
	}
}

func TestUnaryServerInterceptor_MissingTenant(t *testing.T) {
	interceptor := tenant.UnaryServerInterceptor()

	// Setup incoming context WITHOUT the metadata header
	ctx := context.Background()

	// Mock handler (should not be called)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Errorf("Handler should not be invoked when tenant is missing")
		return nil, nil
	}

	// Execute interceptor
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	// Assertions
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	
	if st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated code, got %v", st.Code())
	}
}
