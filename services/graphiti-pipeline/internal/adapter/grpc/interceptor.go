package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TenantExtractorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			tenantID := md.Get("x-tenant-id")
			if len(tenantID) > 0 {
				ctx = context.WithValue(ctx, "tenant_id", tenantID[0])
			}
		}
		return handler(ctx, req)
	}
}
