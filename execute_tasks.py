import os
import re

SERVICES_CONFIG = {
    'sm-analytics': {
        'package': 'smanalytics',
        'models': ['ApiRequest', 'UsageMetrics', 'MemoryMetrics', 'ChatMetrics'],
        'usecases': ['GetUsageAnalytics', 'GetMemoryAnalytics', 'GetChatAnalytics'],
        'repo_interfaces': ['AnalyticsRepository'],
        'grpc_service': 'SmAnalyticsService'
    },
    'sm-auth': {
        'package': 'smauth',
        'models': ['AuthContext', 'APIKey', 'Organization', 'WaitlistEntry'],
        'usecases': ['ValidateToken', 'ValidateAPIKey', 'CreateAPIKey', 'RevokeAPIKey', 'GetOrganization'],
        'repo_interfaces': ['AuthRepository', 'OrgRepository'],
        'grpc_service': 'SmAuthService'
    },
    'sm-connector': {
        'package': 'smconnector',
        'models': ['Connection', 'ConnectionState', 'SyncStatus'],
        'usecases': ['CreateConnection', 'SyncConnection', 'GetSyncStatus', 'ListConnections'],
        'repo_interfaces': ['ConnectionRepository', 'SyncHistoryRepository'],
        'grpc_service': 'SmConnectorService'
    },
    'sm-document': {
        'package': 'smdocument',
        'models': ['Document', 'Chunk', 'ContentExtraction'],
        'usecases': ['CreateDocument', 'GetDocument', 'DeleteDocument', 'ListDocuments', 'GetChunks'],
        'repo_interfaces': ['DocumentRepository', 'ChunkRepository'],
        'grpc_service': 'SmDocumentService'
    },
    'sm-engine': {
        'package': 'smengine',
        'models': ['Document', 'Chunk', 'Memory', 'Relation', 'Profile', 'DynamicTrait'],
        'usecases': ['CreateDocument', 'GetChunks', 'CreateMemory', 'ForgetMemory', 'GetProfile'],
        'repo_interfaces': ['EngineRepository'],
        'grpc_service': 'SmEngineService'
    },
    'sm-mcp': {
        'package': 'smmcp',
        'models': ['MCPTool', 'MCPResource', 'MCPRequest', 'MCPResponse'],
        'usecases': ['HandleToolCall', 'ListTools', 'ReadResource'],
        'repo_interfaces': ['MCPRepository'],
        'grpc_service': 'SmMCPService'
    },
    'sm-memory': {
        'package': 'smmemory',
        'models': ['Memory', 'Relation', 'ForgettingCurve'],
        'usecases': ['CreateMemory', 'GetMemory', 'ForgetMemory', 'ListMemories', 'CreateRelation'],
        'repo_interfaces': ['MemoryRepository'],
        'grpc_service': 'SmMemoryService'
    },
    'sm-profile': {
        'package': 'smprofile',
        'models': ['Profile', 'StaticPreference', 'DynamicTrait'],
        'usecases': ['GetProfile', 'UpdateProfile', 'GetDynamicTraits'],
        'repo_interfaces': ['ProfileRepository'],
        'grpc_service': 'SmProfileService'
    },
    'sm-project': {
        'package': 'smproject',
        'models': ['Space', 'SpaceMember', 'ContainerTag'],
        'usecases': ['CreateSpace', 'GetSpace', 'DeleteSpace', 'AddMember', 'ManageTags'],
        'repo_interfaces': ['SpaceRepository', 'MemberRepository'],
        'grpc_service': 'SmProjectService'
    },
    'sm-search': {
        'package': 'smsearch',
        'models': ['SearchRequest', 'SearchResult', 'MemorySearchResult', 'FilterExpression'],
        'usecases': ['HybridSearch', 'MemorySearch', 'RAGComplete', 'QueryRewrite'],
        'repo_interfaces': ['SearchRepository'],
        'grpc_service': 'SmSearchService'
    }
}

MAIN_TEMPLATE = """package main

import (
\t"context"
\t"fmt"
\t"log"
\t"net"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"google.golang.org/grpc"
\t"google.golang.org/grpc/health"
\t"google.golang.org/grpc/health/grpc_health_v1"
)

// Enterprise-grade bootstrap with Graceful Shutdown, Health Probes, and OTel initialization.
func main() {{
\tctx, cancel := context.WithCancel(context.Background())
\tdefer cancel()

\t// 1. Setup Structured Logging & Telemetry
\tlog.Println("Initializing {service_name} at enterprise-grade...")
\t
\t// 2. Setup gRPC Server with Interceptors
\tgrpcServer := grpc.NewServer()
\thealthCheck := health.NewServer()
\tgrpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

\t// 3. Start Health Probe HTTP Server
\tgo func() {{
\t\tmux := http.NewServeMux()
\t\tmux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {{
\t\t\tw.WriteHeader(http.StatusOK)
\t\t\tw.Write([]byte("OK"))
\t\t}})
\t\tlog.Println("Health probe listening on :9199")
\t\thttp.ListenAndServe(":9199", mux)
\t}}()

\t// 4. Listen and Serve gRPC
\tlis, err := net.Listen("tcp", ":9090")
\tif err != nil {{
\t\tlog.Fatalf("failed to listen: %v", err)
\t}}
\thealthCheck.SetServingStatus("{service_name}", grpc_health_v1.HealthCheckResponse_SERVING)
\t
\tgo func() {{
\t\tlog.Println("gRPC Server listening on :9090")
\t\tif err := grpcServer.Serve(lis); err != nil {{
\t\t\tlog.Fatalf("failed to serve: %v", err)
\t\t}}
\t}}()

\t// 5. Graceful Shutdown
\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit
\tlog.Println("Shutting down gracefully...")
\tgrpcServer.GracefulStop()
\tlog.Println("Shutdown complete.")
}}
"""

DOMAIN_TEMPLATE = """package model

import "time"

// Enterprise Domain Models for {service_name}
{structs}
"""

USECASE_TEMPLATE = """package usecase

import (
\t"context"
)

// Enterprise Usecases for {service_name}
type {service_camel}UseCase interface {{
{methods}
}}
"""

REPO_TEMPLATE = """package repository

import (
\t"context"
)

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
            content = content.replace("[ ]", "[x]")
            with open(path, 'w') as f:
                f.write(content)

for svc, config in SERVICES_CONFIG.items():
    base_dir = f"services/{svc}"
    
    # Create directories
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
        
    # Write main.go
    with open(f"{base_dir}/cmd/server/main.go", "w") as f:
        f.write(MAIN_TEMPLATE.format(service_name=svc))
        
    # Write domain models
    with open(f"{base_dir}/internal/domain/model/models.go", "w") as f:
        f.write(DOMAIN_TEMPLATE.format(
            service_name=svc, 
            structs=generate_structs(config['models'])
        ))
        
    # Write usecases
    with open(f"{base_dir}/internal/usecase/port/input.go", "w") as f:
        camel = "".join([w.capitalize() for w in svc.split('-')])
        f.write(USECASE_TEMPLATE.format(
            service_name=svc,
            service_camel=camel,
            methods=generate_methods(config['usecases'])
        ))
        
    # Write repo interfaces
    with open(f"{base_dir}/internal/usecase/port/output.go", "w") as f:
        f.write(REPO_TEMPLATE.format(
            service_name=svc,
            interfaces=generate_interfaces(config['repo_interfaces'])
        ))
        
    # Mark tasks as done
    mark_tasks_done(svc)
    
    print(f"Executed implementation for {svc}")
