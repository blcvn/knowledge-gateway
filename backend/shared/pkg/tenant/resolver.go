package tenant

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Define context keys to avoid collisions
type contextKey string

const (
	// TenantIDKey is the key used to store the Tenant ID in the context
	TenantIDKey contextKey = "x-tenant-id"
	// ProjectIDKey is the key used to store the Project UUID in the context
	ProjectIDKey contextKey = "x-project-id"
)

// ExtractFromMetadata extracts tenant headers from gRPC metadata.
func ExtractFromMetadata(ctx context.Context) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	// Extract Tenant ID (can be Org ID or Account ID)
	tenantVals := md.Get(string(TenantIDKey))
	if len(tenantVals) == 0 || strings.TrimSpace(tenantVals[0]) == "" {
		return "", "", status.Errorf(codes.Unauthenticated, "missing %s header", TenantIDKey)
	}
	tenantID := tenantVals[0]

	// Extract Project ID (optional in some contexts, but useful for Zep/Supermemory)
	var projectID string
	projectVals := md.Get(string(ProjectIDKey))
	if len(projectVals) > 0 {
		projectID = strings.TrimSpace(projectVals[0])
	}

	return tenantID, projectID, nil
}

// InjectIntoContext places the tenant identifiers strongly into the Go Context.
func InjectIntoContext(ctx context.Context, tenantID, projectID string) context.Context {
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	if projectID != "" {
		ctx = context.WithValue(ctx, ProjectIDKey, projectID)
	}
	return ctx
}

// UnaryServerInterceptor creates a gRPC interceptor that enforces multi-tenancy.
// It extracts the tenant info from metadata and injects it into the request context.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		
		// 1. Extract
		tenantID, projectID, err := ExtractFromMetadata(ctx)
		if err != nil {
			// For health checks, bypass tenant validation
			if isHealthCheck(info.FullMethod) {
				return handler(ctx, req)
			}
			return nil, err
		}

		// 2. Inject
		newCtx := InjectIntoContext(ctx, tenantID, projectID)

		// 3. Process
		return handler(newCtx, req)
	}
}

// FromContext safely retrieves the TenantID from the context, ensuring isolation.
// This should be called by the Repositories/Usecases to scope DB queries.
func FromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(TenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", fmt.Errorf("tenant isolation violation: context missing tenant_id")
	}
	return tenantID, nil
}

func isHealthCheck(fullMethod string) bool {
	return strings.Contains(fullMethod, "/grpc.health.v1.Health/Check") || 
	       strings.Contains(fullMethod, "/grpc.health.v1.Health/Watch")
}
