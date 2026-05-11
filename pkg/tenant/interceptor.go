package tenant

import (
\t"context"

\t"google.golang.org/grpc"
\t"google.golang.org/grpc/codes"
\t"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC interceptor that enforces multi-tenancy.
// It extracts the tenant ID from the incoming gRPC metadata and injects it into the context.
// If the tenant ID is missing, it rejects the request.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
\treturn func(
\t\tctx context.Context,
\t\treq interface{},
\t\tinfo *grpc.UnaryServerInfo,
\t\thandler grpc.UnaryHandler,
\t) (interface{}, error) {
\t\t
\t\t// Extract tenant ID from metadata
\t\ttenantID, err := ExtractFromMetadata(ctx)
\t\tif err != nil {
\t\t\t// Reject the request immediately if no tenant context is provided
\t\t\t// This ensures no cross-tenant data leakage can happen accidentally.
\t\t\treturn nil, status.Errorf(codes.Unauthenticated, "multi-tenant policy violation: %v", err)
\t\t}

\t\t// Inject the tenant ID down into the context for Usecases and Repositories to use
\t\tnewCtx := WithTenant(ctx, tenantID)

\t\t// Call the actual handler
\t\treturn handler(newCtx, req)
\t}
}
