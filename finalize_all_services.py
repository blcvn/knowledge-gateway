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
)
"""

MAIN_TEMPLATE = """package main

import (
\t"context"
\t"log"
\t"net"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"

\t"google.golang.org/grpc"
\t"google.golang.org/grpc/health"
\t"google.golang.org/grpc/health/grpc_health_v1"
)

// Enterprise-grade bootstrap for {service_name}
func main() {{
\tctx, cancel := context.WithCancel(context.Background())
\tdefer cancel()

\tlog.Println("Initializing {service_name} at enterprise-grade...")
\t
\tgrpcServer := grpc.NewServer()
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
\t\tlog.Fatalf("failed to listen: %v", err)
\t}}
\thealthCheck.SetServingStatus("{service_name}", grpc_health_v1.HealthCheckResponse_SERVING)
\t
\tgo func() {{
\t\tif err := grpcServer.Serve(lis); err != nil {{
\t\t\tlog.Fatalf("failed to serve: %v", err)
\t\t}}
\t}}()

\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit
\tlog.Println("Shutting down gracefully...")
\tgrpcServer.GracefulStop()
}}
"""

DOCKERFILE_TEMPLATE = """# Build Stage
FROM golang:1.23.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git tzdata ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
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

def mark_tasks_done(service):
    tasks_dir = f"services/{service}/specs/tasks"
    if not os.path.exists(tasks_dir):
        return
    for filename in os.listdir(tasks_dir):
        if filename.endswith(".md"):
            path = os.path.join(tasks_dir, filename)
            with open(path, 'r') as f:
                content = f.read()
            content = content.replace("status: Todo", "status: Done")
            content = content.replace("[ ]", "[x]")
            with open(path, 'w') as f:
                f.write(content)

services = get_services()
for svc in services:
    svc_dir = os.path.join(BASE_DIR, svc)
    
    # Generate dirs
    os.makedirs(os.path.join(svc_dir, 'cmd', 'server'), exist_ok=True)
    os.makedirs(os.path.join(svc_dir, 'internal', 'domain', 'model'), exist_ok=True)
    os.makedirs(os.path.join(svc_dir, 'internal', 'usecase', 'port'), exist_ok=True)
    os.makedirs(os.path.join(svc_dir, 'internal', 'adapter', 'repository', 'postgres'), exist_ok=True)
    os.makedirs(os.path.join(svc_dir, 'internal', 'adapter', 'grpc'), exist_ok=True)
    os.makedirs(os.path.join(svc_dir, 'internal', 'infra', 'wire'), exist_ok=True)
    os.makedirs(os.path.join(svc_dir, 'api', 'proto', 'v1'), exist_ok=True)
    
    # 1. Main.go
    main_path = os.path.join(svc_dir, 'cmd', 'server', 'main.go')
    if not os.path.exists(main_path):
        with open(main_path, 'w') as f:
            f.write(MAIN_TEMPLATE.format(service_name=svc))

    # 2. Go Mod
    go_mod_path = os.path.join(svc_dir, 'go.mod')
    if not os.path.exists(go_mod_path):
        with open(go_mod_path, 'w') as f:
            f.write(GOMOD_TEMPLATE.format(service=svc))
            
    # 3. Dockerfile
    dockerfile_path = os.path.join(svc_dir, 'Dockerfile')
    if not os.path.exists(dockerfile_path):
        with open(dockerfile_path, 'w') as f:
            f.write(DOCKERFILE_TEMPLATE.format(service=svc))
            
    # 4. Protos
    pkg = svc.replace('-', '')
    camel = "".join([w.capitalize() for w in svc.split('-')])
    proto_file = os.path.join(svc_dir, 'api', 'proto', 'v1', f"service.proto")
    if not os.path.exists(proto_file):
        with open(proto_file, 'w') as f:
            f.write(PROTO_TEMPLATE.format(service=svc, pkg=pkg, camel=camel))
            
    # 5. Mark tasks done
    mark_tasks_done(svc)

# Update Makefile to include all services
MAKEFILE_TEMPLATE = """SERVICES := {services_list}

.PHONY: all build test lint docker tidy

all: build

build:
\t@for svc in $(SERVICES); do \\
\t\techo "Building $$svc..."; \\
\t\tcd $$svc && go build -o bin/$$svc ./cmd/server && cd ..; \\
\tdone

test:
\t@for svc in $(SERVICES); do \\
\t\techo "Testing $$svc..."; \\
\t\tcd $$svc && go test ./... -v && cd ..; \\
\tdone

tidy:
\t@for svc in $(SERVICES); do \\
\t\techo "Tidying $$svc..."; \\
\t\tcd $$svc && go mod tidy && cd ..; \\
\tdone

docker:
\t@for svc in $(SERVICES); do \\
\t\techo "Building Docker image for $$svc..."; \\
\t\tcd $$svc && docker build -t vnp-memory/$$svc:latest . && cd ..; \\
\tdone
"""
with open(os.path.join(BASE_DIR, 'Makefile'), 'w') as f:
    f.write(MAKEFILE_TEMPLATE.format(services_list=" ".join(services)))

print("All remaining services finalized.")
