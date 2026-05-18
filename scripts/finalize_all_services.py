import os

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'

# Exclude already processed or non-services
EXCLUDE = ['Makefile', 'zep-go']

def get_services():
    services = []
    for item in os.listdir(BASE_DIR):
        path = os.path.join(BASE_DIR, item)
        if os.path.isdir(path) and item not in EXCLUDE:
            services.append(item)
    return services

GOMOD_TEMPLATE = """module vnp-memory/services/{service}

go 1.23.0

require (
\tgoogle.golang.org/grpc v1.65.0
\tgoogle.golang.org/protobuf v1.34.2
\tgo.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
\tvnp-memory/pkg/telemetry v0.0.0
\tvnp-memory/pkg/tenant v0.0.0
)

replace vnp-memory/pkg/telemetry => ../../pkg/telemetry
replace vnp-memory/pkg/tenant => ../../pkg/tenant
"""

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
\t"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
\t
\t"vnp-memory/pkg/telemetry"
\t"vnp-memory/pkg/tenant"
)

// Enterprise-grade bootstrap for {service_name}
func main() {{
\tctx, cancel := context.WithCancel(context.Background())
\tdefer cancel()

\t// 1. Init Structured Logging
\ttelemetry.InitLogger("info")
\tslog.Info("Initializing {service_name} at enterprise-grade...")

\t// 2. Init OpenTelemetry Provider
\tshutdownOtel, err := telemetry.InitProvider(ctx, "{service_name}")
\tif err != nil {{
\t\tslog.Error("Failed to init OpenTelemetry", slog.String("error", err.Error()))
\t\tos.Exit(1)
\t}}
\tdefer func() {{
\t\tif err := shutdownOtel(context.Background()); err != nil {{
\t\t\tslog.Error("OTel shutdown error", slog.String("error", err.Error()))
\t\t}}
\t}}()

\t// 3. Setup gRPC Server with Interceptors (OTel Tracing + Multi-Tenant Auth)
\tgrpcServer := grpc.NewServer(
\t\tgrpc.StatsHandler(otelgrpc.NewServerHandler()),
\t\tgrpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
\t)

\t// 4. Health Checks
\thealthCheck := health.NewServer()
\tgrpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

\tgo func() {{
\t\tmux := http.NewServeMux()
\t\tmux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {{
\t\t\tw.WriteHeader(http.StatusOK)
\t\t\tw.Write([]byte("OK"))
\t\t}})
\t\thttp.ListenAndServe(":9199", mux)
\t}}()

\tlis, err := net.Listen("tcp", ":9090")
\tif err != nil {{
\t\tslog.Error("failed to listen", slog.String("error", err.Error()))
\t\tos.Exit(1)
\t}}
\thealthCheck.SetServingStatus("{service_name}", grpc_health_v1.HealthCheckResponse_SERVING)
\t
\tgo func() {{
\t\tif err := grpcServer.Serve(lis); err != nil {{
\t\t\tslog.Error("failed to serve gRPC", slog.String("error", err.Error()))
\t\t}}
\t}}()

\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit
\tslog.Info("Shutting down gracefully...")
\tgrpcServer.GracefulStop()
}}
"""

DOCKERFILE_TEMPLATE = """# Build Stage
FROM golang:1.23.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git tzdata ca-certificates
# Copy global packages
COPY pkg/ ./pkg/
# Copy service
COPY services/{service}/ ./services/{service}/
WORKDIR /app/services/{service}
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/{service} ./cmd/server

# Final Stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/{service} .
ENV ENV=production TZ=UTC
EXPOSE 9090 9199
USER nobody:nobody
CMD ["./{service}"]
"""

PROTO_TEMPLATE = """syntax = "proto3";
package vnp.memory.{pkg}.v1;
option go_package = "vnp-memory/services/{service}/api/proto/v1;{pkg}v1";

service {camel}Service {{
  rpc Ping(PingRequest) returns (PingResponse);
}}

message PingRequest {{}}
message PingResponse {{
  string status = 1;
}}
"""

services = get_services()
for svc in services:
    svc_dir = os.path.join(BASE_DIR, svc)
    
    # 1. Main.go
    main_path = os.path.join(svc_dir, 'cmd', 'server', 'main.go')
    with open(main_path, 'w') as f:
        f.write(MAIN_TEMPLATE.format(service_name=svc))

    # 2. Go Mod
    go_mod_path = os.path.join(svc_dir, 'go.mod')
    with open(go_mod_path, 'w') as f:
        f.write(GOMOD_TEMPLATE.format(service=svc))
        
    # 3. Dockerfile
    dockerfile_path = os.path.join(svc_dir, 'Dockerfile')
    with open(dockerfile_path, 'w') as f:
        f.write(DOCKERFILE_TEMPLATE.format(service=svc))

print("All services updated with enterprise interceptors (telemetry & tenant).")
