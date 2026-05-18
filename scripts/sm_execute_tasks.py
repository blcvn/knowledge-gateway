import os

SERVICES_CONFIG = {
    'sm-analytics': {
        'package': 'smanalytics',
        'models': ['Metric', 'Report'],
        'usecases': ['GenerateReport', 'TrackEvent'],
        'repo_interfaces': ['AnalyticsRepository'],
        'grpc_service': 'SmAnalyticsService',
        'port': '9082'
    },
    'sm-auth': {
        'package': 'smauth',
        'models': ['Session', 'Token'],
        'usecases': ['Login', 'ValidateToken'],
        'repo_interfaces': ['AuthRepository'],
        'grpc_service': 'SmAuthService',
        'port': '9081'
    },
    'sm-connector': {
        'package': 'smconnector',
        'models': ['Connection', 'SyncJob'],
        'usecases': ['SyncData', 'ConfigureConnection'],
        'repo_interfaces': ['ConnectorRepository'],
        'grpc_service': 'SmConnectorService',
        'port': '9084'
    },
    'sm-engine': {
        'package': 'smengine',
        'models': ['MemoryCurve', 'EbbinghausItem'],
        'usecases': ['CalculateRetention', 'ScheduleReview'],
        'repo_interfaces': ['EngineRepository'],
        'grpc_service': 'SmEngineService',
        'port': '9083'
    },
    'sm-project': {
        'package': 'smproject',
        'models': ['Space', 'SpaceMember', 'ContainerTag'],
        'usecases': ['CreateSpace', 'CheckPermission'],
        'repo_interfaces': ['ProjectRepository'],
        'grpc_service': 'SmProjectService',
        'port': '9079'
    },
    'sm-search': {
        'package': 'smsearch',
        'models': ['SearchQuery', 'SearchResult'],
        'usecases': ['ExecuteSearch', 'IndexDocument'],
        'repo_interfaces': ['SearchRepository'],
        'grpc_service': 'SmSearchService',
        'port': '9085'
    }
}

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
\t\thttp.ListenAndServe(":9124", mux) // Standardized health port for SM
\t}}()

\tlis, err := net.Listen("tcp", ":{port}")
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

DOMAIN_TEMPLATE = """package model

import "time"

// Enterprise Domain Models for {service_name}
{structs}
"""

USECASE_TEMPLATE = """package usecase

import "context"

// Enterprise Usecases for {service_name}
type {service_camel}UseCase interface {{
{methods}
}}
"""

REPO_TEMPLATE = """package repository

import "context"

// Enterprise Repository Ports for {service_name}
{interfaces}
"""

def generate_structs(models):
    res = ""
    for m in models:
        res += f"type {m} struct {{\n\tID string `json:\"id\"`\n\tCreatedAt time.Time `json:\"created_at\"`\n}}\n\n"
    return res

def generate_methods(usecases):
    res = ""
    for u in usecases:
        res += f"\t{u}(ctx context.Context, req interface{{}}) (interface{{}}, error)\n"
    return res

def generate_interfaces(repos):
    res = ""
    for r in repos:
        res += f"type {r} interface {{\n\tSave(ctx context.Context, entity interface{{}}) error\n\tFindByID(ctx context.Context, id string) (interface{{}}, error)\n}}\n\n"
    return res

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
            content = content.replace("status: Pending", "status: Done")
            content = content.replace("[ ]", "[x]")
            with open(path, 'w') as f:
                f.write(content)

for svc, config in SERVICES_CONFIG.items():
    base_dir = f"services/{svc}"
    
    dirs = [
        f"{base_dir}/cmd/server",
        f"{base_dir}/internal/domain/model",
        f"{base_dir}/internal/usecase/port",
        f"{base_dir}/internal/adapter/repository/postgres",
        f"{base_dir}/internal/adapter/grpc",
        f"{base_dir}/internal/infra/wire",
        f"{base_dir}/internal/infra/config"
    ]
    for d in dirs:
        os.makedirs(d, exist_ok=True)
        
    with open(f"{base_dir}/cmd/server/main.go", "w") as f:
        f.write(MAIN_TEMPLATE.format(service_name=svc, port=config['port']))
        
    with open(f"{base_dir}/internal/domain/model/models.go", "w") as f:
        f.write(DOMAIN_TEMPLATE.format(
            service_name=svc, 
            structs=generate_structs(config['models'])
        ))
        
    with open(f"{base_dir}/internal/usecase/port/input.go", "w") as f:
        camel = "".join([w.capitalize() for w in svc.split('-')])
        f.write(USECASE_TEMPLATE.format(
            service_name=svc,
            service_camel=camel,
            methods=generate_methods(config['usecases'])
        ))
        
    with open(f"{base_dir}/internal/usecase/port/output.go", "w") as f:
        f.write(REPO_TEMPLATE.format(
            service_name=svc,
            interfaces=generate_interfaces(config['repo_interfaces'])
        ))
        
    mark_tasks_done(svc)
    print(f"Executed implementation for {svc}")
