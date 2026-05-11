import os

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'
EXCLUDE = ['Makefile', 'zep-go']

def get_services():
    services = []
    for item in os.listdir(BASE_DIR):
        path = os.path.join(BASE_DIR, item)
        if os.path.isdir(path) and item not in EXCLUDE:
            services.append(item)
    return services

MAIN_TEMPLATE = """package main

import (
\t"context"
\t"log/slog"
\t"net"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"

\t"google.golang.org/grpc"
\t"google.golang.org/grpc/health"
\t"google.golang.org/grpc/health/grpc_health_v1"

\t"vnp-memory/pkg/telemetry"
\t"vnp-memory/pkg/tenant"
)

// Enterprise-grade bootstrap for {service_name}
func main() {{
\tctx, cancel := context.WithCancel(context.Background())
\tdefer cancel()

\t// 1. Initialize Structured Logging
\ttelemetry.InitLogger("info")
\tslog.Info("Initializing {service_name} at enterprise-grade...")

\t// 2. Initialize OpenTelemetry Distributed Tracing
\tshutdownTracer, err := telemetry.InitProvider(ctx, "{service_name}")
\tif err != nil {{
\t\tslog.Error("failed to initialize OTel provider", slog.String("error", err.Error()))
\t\tos.Exit(1)
\t}}
\tdefer func() {{
\t\tif err := shutdownTracer(context.Background()); err != nil {{
\t\t\tslog.Error("failed to shutdown OTel provider gracefully", slog.String("error", err.Error()))
\t\t}}
\t}}()

\t// 3. Setup gRPC Server with Interceptors (Tenant isolation & OTel traces)
\tgrpcServer := grpc.NewServer(
\t\tgrpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
\t)

\t// 4. Setup Health Probes
\thealthCheck := health.NewServer()
\tgrpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

\t// 5. Start HTTP Health/Metrics Server
\tgo func() {{
\t\tmux := http.NewServeMux()
\t\tmux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {{
\t\t\tw.WriteHeader(http.StatusOK)
\t\t\tw.Write([]byte("OK"))
\t\t}})
\t\tslog.Info("Starting HTTP probe server on :9199")
\t\tif err := http.ListenAndServe(":9199", mux); err != nil {{
\t\t\tslog.Error("HTTP probe server failed", slog.String("error", err.Error()))
\t\t}}
\t}}()

\t// 6. Start gRPC Server
\tlis, err := net.Listen("tcp", ":9090")
\tif err != nil {{
\t\tslog.Error("failed to listen", slog.String("error", err.Error()))
\t\tos.Exit(1)
\t}}
\thealthCheck.SetServingStatus("{service_name}", grpc_health_v1.HealthCheckResponse_SERVING)
\t
\tgo func() {{
\t\tslog.Info("Starting gRPC server on :9090")
\t\tif err := grpcServer.Serve(lis); err != nil {{
\t\t\tslog.Error("failed to serve gRPC", slog.String("error", err.Error()))
\t\t}}
\t}}()

\t// 7. Graceful Shutdown Management
\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit
\tslog.Info("Shutting down gracefully...")
\tgrpcServer.GracefulStop()
\tslog.Info("Server exited properly")
}}
"""

services = get_services()
for svc in services:
    main_path = os.path.join(BASE_DIR, svc, 'cmd', 'server', 'main.go')
    # Overwrite all main.go files with the latest enterprise template
    if os.path.exists(main_path):
        with open(main_path, 'w') as f:
            f.write(MAIN_TEMPLATE.format(service_name=svc))

print("Applied enterprise bootstrap to all service main.go files.")
