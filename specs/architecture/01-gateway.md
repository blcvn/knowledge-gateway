# VNP Memory — Unified API Gateway

> **Service**: `vnp-gateway` | **Port**: 8080(REST) 8081(gRPC) 8082(MCP)

---

## 1. Responsibility

Single entry point for ALL VNP Memory APIs — routes to 20 domain services. Handles auth, rate limiting, protocol translation, MCP server, WebDAV proxy, and circuit breaking.

---

## 2. REST API Routes

### 2.1 Unified Memory API (`/v1/memory/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/memory/store` | Auto-route (see §3) | Store memory (auto-classified) |
| POST | `/v1/memory/recall` | vnp-search-hub | Cross-engine recall |
| POST | `/v1/memory/forget` | Fan-out → All | Cascading delete |
| GET | `/v1/memory/timeline` | vnp-event | Temporal event query |

### 2.2 Cognee API (`/v1/cognee/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/cognee/datasets` | cognee-ingestion | Create dataset |
| POST | `/v1/cognee/datasets/{id}/data` | cognee-ingestion | Upload data |
| POST | `/v1/cognee/datasets/{id}/cognify` | cognee-cognify | Trigger KG pipeline |
| POST | `/v1/cognee/search` | cognee-search | Search knowledge |

### 2.3 Graphiti API (`/v1/graphiti/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/graphiti/episodes` | graphiti-ingestion | Ingest episode |
| POST | `/v1/graphiti/search` | graphiti-search | Hybrid search |
| GET | `/v1/graphiti/nodes/{id}` | graphiti-store | Get node |
| GET | `/v1/graphiti/edges/{id}` | graphiti-store | Get edge |

### 2.4 Memobase API (`/v1/memobase/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/memobase/users/{uid}/blobs` | memobase-ingestion | Insert blob |
| POST | `/v1/memobase/users/{uid}/flush` | memobase-ingestion | Force flush buffer |
| GET | `/v1/memobase/users/{uid}/context` | memobase-context | Get context string |
| GET | `/v1/memobase/users/{uid}/profiles` | memobase-context | Get profiles |
| GET | `/v1/memobase/users/{uid}/events` | vnp-event | Get user events |

### 2.5 OpenViking API (`/v1/ov/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| GET | `/v1/ov/files/{path}` | ov-fs | Read file |
| PUT | `/v1/ov/files/{path}` | ov-fs | Write file |
| DELETE | `/v1/ov/files/{path}` | ov-fs | Delete file |
| GET | `/v1/ov/tree/{path}` | ov-fs | Directory tree |
| POST | `/v1/ov/grep` | ov-fs | Content search |
| POST | `/v1/ov/search` | ov-search | Hierarchical retrieval |
| POST | `/v1/ov/sessions` | ov-session | Create session |
| POST | `/v1/ov/sessions/{id}/messages` | ov-session | Add message |
| POST | `/v1/ov/sessions/{id}/commit` | ov-session | Commit (2-phase) |
| POST | `/v1/ov/resources/ingest` | ov-resource | Ingest resource |
| WebDAV | `/webdav/*` | ov-fs | WebDAV file access |

### 2.6 Zep API (`/v1/zep/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/zep/users` | zep-user | Create user |
| GET | `/v1/zep/users/{id}` | zep-user | Get user |
| POST | `/v1/zep/sessions/{id}/memory` | zep-memory | PutMemory (ingest) |
| GET | `/v1/zep/sessions/{id}/memory` | zep-memory | GetMemory (retrieve) |
| POST | `/v1/zep/graph/search` | zep-search | Graph search |
| POST | `/v1/zep/sessions/{id}/search` | zep-search | Session search |
| POST | `/v1/zep/graph/facts` | zep-graph | Add fact |
| POST | `/v1/zep/graph/ontology` | zep-graph | Set ontology |

### 2.7 Supermemory API (`/v1/sm/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/sm/documents` | sm-document | Create document |
| GET | `/v1/sm/documents/{id}` | sm-document | Get document |
| POST | `/v1/sm/memories` | sm-memory | Create memory |
| POST | `/v1/sm/search` | sm-search | Hybrid search |
| POST | `/v1/sm/rag` | sm-search | RAG completion |
| GET | `/v1/sm/profiles/{uid}` | sm-profile | Get profile |
| POST | `/v1/sm/connections` | sm-connector | Create connection |
| POST | `/v1/sm/connections/{id}/sync` | sm-connector | Trigger sync |
| POST | `/v1/sm/projects/spaces` | sm-project | Create space |

### 2.8 Admin API (`/v1/admin/*`)

| Method | Path | Target Service | Description |
|--------|------|---------------|-------------|
| POST | `/v1/admin/tenants` | vnp-admin | Create tenant |
| POST | `/v1/admin/tenants/{id}/keys` | vnp-admin | Issue API key |
| GET | `/v1/admin/health` | vnp-admin | Aggregated health |

---

## 3. Auto-Routing Logic (memory.store)

```go
func (g *Gateway) routeStore(req *StoreRequest) Service {
    switch req.Type {
    case "semantic":
        return g.cogneeIngestion
    case "episodic":
        return g.graphitiIngestion
    case "conversational", "profile":
        return g.memobaseIngestion
    case "procedural":
        return g.ovResource
    case "auto":
        classified := g.classify(req.Data)
        return g.routeStore(&StoreRequest{Type: classified, Data: req.Data})
    }
}
```

---

## 4. MCP Server (Port 8082)

Transport: stdio, SSE, HTTP Streamable

### Tools

| Tool | Description | Target |
|------|-------------|--------|
| `memory_store` | Store memory (auto-route) | Auto-route |
| `memory_recall` | Cross-engine recall | vnp-search-hub |
| `memory_search` | Semantic search | cognee-search |
| `memory_timeline` | Temporal events | vnp-event |
| `memory_profile` | Get user profile | memobase-context |
| `memory_forget` | Delete memory | Fan-out |
| `graph_query` | KG query | graphiti-store |
| `ov_read_file` | Read file | ov-fs |
| `ov_write_file` | Write file | ov-fs |
| `ov_search` | Hierarchical search | ov-search |
| `ov_list_dir` | List directory | ov-fs |
| `ov_grep` | Content grep | ov-fs |
| `ov_tree` | Directory tree | ov-fs |
| `ov_session_commit` | Commit session | ov-session |
| `ov_ingest` | Ingest resource | ov-resource |
| `ov_delete` | Delete file | ov-fs |

---

## 5. Clean Architecture

```
internal/
├── domain/
│   ├── entity.go          # RouteTarget, ProtocolType, AuthContext
│   ├── errors.go          # GatewayError types
│   └── event.go           # RequestReceived, RequestRouted
├── usecase/
│   ├── route.go           # RouteUseCase — classify + route
│   ├── auth.go            # AuthenticateUseCase — JWT/APIKey
│   ├── mcp.go             # MCPServerUseCase — tool dispatch
│   └── port/
│       ├── input.go       # Router, Authenticator, MCPHandler
│       └── output.go      # ServiceRegistry, TenantStore, KeyStore
├── adapter/
│   ├── http/              # chi/v5 REST handlers
│   │   ├── router.go
│   │   ├── memory_handler.go
│   │   ├── cognee_handler.go
│   │   ├── graphiti_handler.go
│   │   ├── memobase_handler.go
│   │   ├── openviking_handler.go
│   │   └── admin_handler.go
│   ├── grpc/              # gRPC-Web proxy
│   ├── mcp/               # MCP SSE/HTTP handler
│   ├── webdav/            # WebDAV proxy → ov-fs
│   ├── ws/                # WebSocket handler
│   └── client/            # gRPC clients to all 20 services
└── infra/
    ├── config/config.go
    ├── server/http.go
    ├── middleware/         # auth, cors, ratelimit, circuit_breaker
    └── wire/wire.go
```

---

## 6. Cross-Cutting

| Feature | Implementation |
|---------|---------------|
| Auth | JWT RS256 + API Key → tenant_id extraction |
| Rate Limit | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | sony/gobreaker per downstream service |
| CORS | Configurable origins, credentials |
| Request ID | UUID v7 generation, X-Request-ID header |
| Timeout | 30s default, 120s for ingestion, 10s for search |
| Metrics | Request count, latency histogram, error rate per route |
