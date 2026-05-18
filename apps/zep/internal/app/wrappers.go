package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/pkg/telemetry"
	"vnp-memory/pkg/tenant"
)

// ZepServiceWrapper wraps a Zep microservice execution in the monolith.
type ZepServiceWrapper struct {
	name string
	port string
}

func NewZepServiceWrapper(name string, port string) *ZepServiceWrapper {
	return &ZepServiceWrapper{
		name: name,
		port: port,
	}
}

func (w *ZepServiceWrapper) Name() string {
	return w.name
}

func (w *ZepServiceWrapper) Start(ctx context.Context) error {
	slog.Info("Starting embedded Zep service...", "service", w.name, "port", w.port)

	// In a complete zero-modification constraint, this would inject env vars and call
	// the actual unexported logic or use generic bootstrapping. For now, we simulate
	// the exact gRPC setup the service would normally do.

	// 1. OTel Init
	shutdownTracer, err := telemetry.InitProvider(ctx, w.name)
	if err != nil {
		slog.Warn("OTel init failed, continuing without tracing", "service", w.name)
	} else {
		go func() {
			<-ctx.Done()
			_ = shutdownTracer(context.Background())
		}()
	}

	// 2. gRPC Server Setup
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 3. Bind to port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", w.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s for %s: %w", w.port, w.name, err)
	}
	healthCheck.SetServingStatus(w.name, grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "service", w.name, "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("Stopping embedded gRPC service", "service", w.name)
		grpcServer.GracefulStop()
	}()

	return nil
}

func (w *ZepServiceWrapper) Stop(ctx context.Context) error {
	// The Stop logic is handled gracefully via the context cancellation
	// inside the Start method's goroutines.
	return nil
}

// GatewayWrapper wraps the API gateway.
type GatewayWrapper struct {
	port string
}

func NewGatewayWrapper(port string) *GatewayWrapper {
	return &GatewayWrapper{
		port: port,
	}
}

func (w *GatewayWrapper) Name() string {
	return "zep-gateway"
}

func (w *GatewayWrapper) Start(ctx context.Context) error {
	slog.Info("Starting Gateway REST proxy...", "port", w.port)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"status": "gateway_ok"}`))
	})
	
	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotImplemented)
		rw.Write([]byte(`{"error": "route pending grpc transcoding via internal registry"}`))
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", w.port),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gateway HTTP server failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("Stopping gateway HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	return nil
}

func (w *GatewayWrapper) Stop(ctx context.Context) error {
	return nil
}
