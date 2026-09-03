package server

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor creates a UnaryServerInterceptor for request logging and tracing extraction.
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		
		// Extract tenant-id and trace-id
		tenantID := "unknown"
		traceID := "unknown"
		
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if t := md.Get("x-tenant-id"); len(t) > 0 {
				tenantID = t[0]
			}
			if tr := md.Get("x-trace-id"); len(tr) > 0 {
				traceID = tr[0]
			}
		}

		reqLogger := logger.With(
			"method", info.FullMethod,
			"tenant_id", tenantID,
			"trace_id", traceID,
		)

		reqLogger.Info("gRPC request started")

		// Pass the enriched logger/context down if context allowed it (simplification for example)
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		if err != nil {
			reqLogger.Error("gRPC request failed",
				"duration_ms", duration.Milliseconds(),
				"error", err,
			)
		} else {
			reqLogger.Info("gRPC request completed",
				"duration_ms", duration.Milliseconds(),
			)
		}

		return resp, err
	}
}

// RecoveryInterceptor creates a UnaryServerInterceptor that catches panics and returns an Internal error.
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "Internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
