# VNP Memory — API Gateway Technical Design

> **Service**: `vnp-gateway`
> **Module**: `github.com/vnp-community/vnp-memory/gateway`
> **Port**: 8080 (REST) | 8081 (gRPC) | 8082 (MCP) | 8083 (Health/Metrics)
> **Updated**: 2026-09-03 — synced from `gateway/adapter/handler/router.go`

---

## 1. Responsibility

Single entry point cho ALL VNP Memory APIs — routes tới 35 domain services. Xử lý auth, rate limiting, protocol translation, MCP server, WebDAV proxy, và circuit breaking.

**Key files:**
- `gateway/adapter/handler/router.go` — tất cả HTTP routes
- `gateway/adapter/client/registry.go` — `GRPCRegistry` (35 downstream services)
- `gateway/adapter/mcp/server.go` — MCP SSE + HTTP Streamable server (22 tools)
- `gateway/internal/infra/middleware/auth.go` — JWT + API Key authentication
- `gateway/internal/infra/config/config.go` — Viper config struct

---

## 2. Auth Middleware

Auth middleware kiểm tra theo thứ tự:
1. **Public paths** skip: `/healthz`, `/metrics`, `/`
2. **OPTIONS** skip (CORS preflight)
3. **X-API-Key** header → `authUC.AuthenticateAPIKey()`
4. **Authorization: Bearer \<token\>** → `authUC.AuthenticateJWT()`
5. Dev mode: JWT token có thể empty

AuthContext inject vào `context.Context`:
```go
type AuthContext struct {
    TenantID string   // mandatory for all DB queries
    UserID   string
    Roles    []string
    Scopes   []string
    RateTier string   // "free" | "pro" | "enterprise"
}
```

---

## 3. Config Structure

```go
type Config struct {
    Server    ServerConfig              // REST_PORT, GRPC_PORT, MCP_PORT, HEALTH_PORT, SHUTDOWN_TIMEOUT
    Auth      AuthConfig                // JWT_PUBLIC_KEY, JWT_ISSUER, JWT_AUDIENCE, API_KEY_PREFIX, DEV_MODE
    Postgres  PostgresConfig            // DSN, MAX_CONNS, MIN_CONNS
    Redis     RedisConfig               // ADDR, PASSWORD, DB
    RateLimit RateLimitConfig           // ENABLED, FREE_RPM, PRO_RPM, ENTERPRISE_RPM
    Circuit   CircuitConfig             // CB_MAX_FAILURES, CB_TIMEOUT, CB_MAX_REQUESTS
    Timeout   TimeoutConfig             // DEFAULT, INGESTION, SEARCH, MCP
    CORS      CORSConfig
    OTEL      OTELConfig
    NATS      NATSConfig
    Services  map[string]string         // service-name → grpc-address
}
```

---

## 4. REST API Routes

> Source of truth: `gateway/adapter/handler/router.go`

### 4.1 Auth API (`/v1/auth/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/auth/login` | `auth.Login` | Login username/password |
| POST | `/v1/auth/logout` | `auth.Logout` | Logout + invalidate token |
| POST | `/v1/auth/refresh` | `auth.Refresh` | Refresh JWT token |
| POST | `/v1/auth/sso/google` | `auth.LoginWithGoogle` | Google SSO |
| GET | `/v1/auth/me` | `auth.Me` | Get current user info |

### 4.2 Unified Memory API (`/v1/memory/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/memory/store` | `memory.Store` | Store (auto-classified, routed) |
| POST | `/v1/memory/recall` | `memory.Recall` | Cross-engine recall |
| POST | `/v1/memory/forget` | `memory.Forget` | Cascading GDPR delete |
| GET | `/v1/memory/timeline` | `memory.Timeline` | Temporal event query |

### 4.3 AgentMemory — Observe (`/v1/observe/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/observe/sessions` | `agentmemH.StartSession` | Start agent session |
| POST | `/v1/observe/sessions/{id}/observe` | `agentmemH.Observe` | Submit hook observation |
| POST | `/v1/observe/sessions/{id}/end` | `agentmemH.EndSession` | End session + trigger consolidation |
| GET | `/v1/observe/sessions/{id}` | `agentmemH.GetSession` | Get session details |
| GET | `/v1/observe/sessions` | `agentmemH.ListSessions` | List sessions |
| DELETE | `/v1/observe/sessions/{id}` | `agentmemH.DeleteSession` | Delete session |
| GET | `/v1/observe/sessions/{id}/observations` | `agentmemH.GetObservations` | List raw observations |
| GET | `/v1/observe/stream` | `agentmemH.StreamEvents` | SSE live stream |

### 4.4 AgentMemory — Agent Memory (`/v1/memory/agent/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/memory/agent/remember` | `agentmemH.RememberAgent` | Store agent memory |
| GET | `/v1/memory/agent/list` | `agentmemH.ListAgentMemories` | List agent memories |
| GET | `/v1/memory/agent/{id}` | `agentmemH.GetAgentMemory` | Get specific memory |
| DELETE | `/v1/memory/agent/{id}` | `agentmemH.DeleteAgentMemory` | Delete memory |
| GET | `/v1/memory/agent/{id}/retention` | `agentmemH.GetRetentionScore` | Salience/retention score |
| POST | `/v1/memory/agent/evict` | `agentmemH.EvictMemories` | Manual eviction |
| POST | `/v1/memory/agent/auto-forget` | `agentmemH.AutoForgetSweep` | Auto-forget sweep |

### 4.5 AgentMemory — Memory Slots (`/v1/memory/slots/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/v1/memory/slots` | `agentmemH.ListSlots` | List all slots |
| GET | `/v1/memory/slots/{scope}/{label}` | `agentmemH.GetSlot` | Get slot by scope+label |
| POST | `/v1/memory/slots/{scope}/{label}` | `agentmemH.WriteSlot` | Write slot value |
| DELETE | `/v1/memory/slots/{scope}/{label}` | `agentmemH.DeleteSlot` | Delete slot |

### 4.6 Cognee API (`/v1/cognee/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/cognee/datasets` | `cognee.CreateDataset` | Create dataset |
| POST | `/v1/cognee/datasets/{id}/data` | `cognee.UploadData` | Upload data |
| POST | `/v1/cognee/datasets/{id}/cognify` | `cognee.Cognify` | Trigger KG pipeline |
| POST | `/v1/cognee/search` | `cognee.Search` | Search knowledge |
| POST | `/v1/cognee/datasets/{id}/memify` | `cognee.Memify` | Memify enrichment |
| GET | `/v1/cognee/datasets/{id}/memify/status` | `cognee.GetMemifyStatus` | Memify status |
| POST | `/v1/cognee/datasets/{id}/datapoints` | `cognee.AddDataPoints` | Add datapoints |

### 4.7 Graphiti API (`/v1/graphiti/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/graphiti/episodes` | `graphiti.IngestEpisode` | Ingest episode |
| POST | `/v1/graphiti/search` | `graphiti.Search` | Hybrid search |
| GET | `/v1/graphiti/nodes/{id}` | `graphiti.GetNode` | Get graph node |
| GET | `/v1/graphiti/edges/{id}` | `graphiti.GetEdge` | Get graph edge |

### 4.8 Memobase API (`/v1/memobase/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/memobase/users/{uid}/blobs` | `memobase.InsertBlob` | Insert conversation blob |
| POST | `/v1/memobase/users/{uid}/flush` | `memobase.Flush` | Force buffer flush → YOLO |
| GET | `/v1/memobase/users/{uid}/context` | `memobase.GetContext` | Get context string (< 100ms) |
| GET | `/v1/memobase/users/{uid}/profiles` | `memobase.GetProfiles` | Get profile categories |
| GET | `/v1/memobase/users/{uid}/events` | `memobase.GetEvents` | Get user events |

### 4.9 OpenViking API (`/v1/ov/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/v1/ov/files/{path...}` | `ov.ReadFile` | Read file |
| PUT | `/v1/ov/files/{path...}` | `ov.WriteFile` | Write file |
| DELETE | `/v1/ov/files/{path...}` | `ov.DeleteFile` | Delete file |
| GET | `/v1/ov/tree/{path...}` | `ov.Tree` | Directory tree |
| POST | `/v1/ov/grep` | `ov.Grep` | Content grep |
| POST | `/v1/ov/search` | `ov.Search` | Semantic search |
| POST | `/v1/ov/sessions` | `ov.CreateSession` | Create edit session |
| POST | `/v1/ov/sessions/{id}/messages` | `ov.AddMessage` | Add message to session |
| POST | `/v1/ov/sessions/{id}/commit` | `ov.CommitSession` | 2-phase commit |
| POST | `/v1/ov/resources/ingest` | `ov.Ingest` | Ingest resource |

### 4.10 Zep API (`/v1/zep/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/zep/users` | `zep.CreateUser` | Create Zep user |
| GET | `/v1/zep/users/{id}` | `zep.GetUser` | Get user |
| PATCH | `/v1/zep/users/{id}` | `zep.UpdateUser` | Update user |
| POST | `/v1/zep/sessions/{id}/memory` | `zep.PutMemory` | Put memory (messages) |
| GET | `/v1/zep/sessions/{id}/memory` | `zep.GetMemory` | Get context assembly |
| POST | `/v1/zep/graph/search` | `zep.GraphSearch` | Semantic graph search |
| POST | `/v1/zep/sessions/{id}/search` | `zep.SessionSearch` | Session search |
| POST | `/v1/zep/graph/facts` | `zep.AddFact` | Add graph fact |
| POST | `/v1/zep/graph/ontology` | `zep.SetOntology` | Set custom ontology |

### 4.11 Supermemory API (`/v1/sm/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/sm/documents` | `sm.CreateDocument` | Create document |
| GET | `/v1/sm/documents/{id}` | `sm.GetDocument` | Get document |
| POST | `/v1/sm/memories` | `sm.CreateMemory` | Store adaptive memory |
| POST | `/v1/sm/search` | `sm.Search` | Search memories |
| POST | `/v1/sm/rag` | `sm.RAG` | RAG query |
| GET | `/v1/sm/profiles/{uid}` | `sm.GetProfile` | Get user profile |
| POST | `/v1/sm/connections` | `sm.CreateConnection` | Create connector |
| POST | `/v1/sm/connections/{id}/sync` | `sm.SyncConnection` | Sync connector |
| POST | `/v1/sm/projects/spaces` | `sm.CreateSpace` | Create project space |

### 4.12 Admin API (`/v1/admin/*`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/v1/admin/tenants` | `admin.CreateTenant` | Create tenant |
| POST | `/v1/admin/tenants/{id}/keys` | `admin.IssueAPIKey` | Issue API key |
| GET | `/v1/admin/health` | `admin.Health` | Aggregated health |
| GET | `/v1/admin/metrics` | `admin.Metrics` | System metrics |

### 4.13 Console API (`/v1/console/*`)

| Group | Method | Path | Description |
|-------|--------|------|-------------|
| Dashboard | GET | `/v1/console/dashboard/health` | Service health |
| Dashboard | GET | `/v1/console/dashboard/metrics` | System metrics |
| Dashboard | GET | `/v1/console/dashboard/throughput` | Request throughput |
| Dashboard | GET | `/v1/console/dashboard/heatmap` | Memory heatmap |
| Memory | POST | `/v1/console/memory/search` | Cross-engine search |
| Memory | GET | `/v1/console/memory/{id}` | Get memory |
| Memory | GET | `/v1/console/memory/{id}/neighbors` | Get graph neighbors |
| Memory | GET | `/v1/console/memory/{id}/versions` | Get version chain |
| Graph | POST | `/v1/console/graph/subgraph` | Subgraph query |
| Graph | GET | `/v1/console/graph/entity/{id}` | Get entity |
| Graph | POST | `/v1/console/graph/timeline` | Timeline query |
| Graph | GET/PUT | `/v1/console/graph/ontology` | Get/Update ontology |
| Graph | POST | `/v1/console/graph/query` | Graph query |
| Profiles | GET | `/v1/console/profiles` | List profiles |
| Profiles | GET/PUT | `/v1/console/profiles/config` | Profile config |
| Profiles | GET | `/v1/console/profiles/{user_id}` | User profile |
| Profiles | GET | `/v1/console/profiles/{user_id}/events` | User events |
| Profiles | GET | `/v1/console/profiles/{user_id}/context` | Context string |
| Profiles | GET | `/v1/console/profiles/{user_id}/buffers` | Blob buffers |
| Adaptive | GET | `/v1/console/adaptive/memories` | Adaptive memories |
| Adaptive | GET | `/v1/console/adaptive/memories/{id}/versions` | Version chain |
| Adaptive | GET/POST | `/v1/console/adaptive/connectors` | Connectors |
| Adaptive | POST | `/v1/console/adaptive/connectors/{id}/sync` | Sync connector |
| Adaptive | GET | `/v1/console/adaptive/analytics` | Analytics |
| Adaptive | GET/PUT | `/v1/console/adaptive/forget-rules` | Forget rules |
| Debugger | POST | `/v1/console/debugger/trace` | Create context trace |
| Debugger | GET | `/v1/console/debugger/traces/{id}` | Get trace |
| Debugger | GET | `/v1/console/debugger/traces` | List traces |
| Sessions | GET | `/v1/console/sessions` | List sessions |
| Sessions | GET | `/v1/console/sessions/live` | Live sessions |
| Sessions | GET | `/v1/console/sessions/{id}` | Session detail |
| Sessions | GET | `/v1/console/sessions/{id}/timeline` | Session timeline |
| Sessions | GET | `/v1/console/sessions/{id}/diff` | Session diff |
| Sessions | GET | `/v1/console/sessions/{id}/working-memory` | Working memory |
| Sessions | GET | `/v1/console/sessions/{id}/user-summary` | User summary |
| Governance | GET/POST | `/v1/console/governance/tenants` | Tenants CRUD |
| Governance | PUT | `/v1/console/governance/tenants/{id}` | Update tenant |
| Governance | GET/POST | `/v1/console/governance/policies` | Policies CRUD |
| Governance | PUT | `/v1/console/governance/policies/{id}` | Update policy |
| Governance | GET | `/v1/console/governance/audit` | Search audit log |
| Governance | POST | `/v1/console/governance/gdpr/forget` | GDPR forget |
| Governance | POST | `/v1/console/governance/gdpr/forget/preview` | Preview forget scope |
| Pipelines | GET | `/v1/console/pipelines/status` | Pipeline status |
| Pipelines | GET | `/v1/console/pipelines/queues` | Queue depths |
| Pipelines | GET | `/v1/console/pipelines/workers` | Worker status |
| Pipelines | GET | `/v1/console/pipelines/templates` | Pipeline templates |
| Pipelines | GET | `/v1/console/pipelines/{engine}` | Engine pipeline |
| Pipelines | GET | `/v1/console/pipelines/{engine}/jobs` | Jobs list |
| Pipelines | GET | `/v1/console/pipelines/{engine}/jobs/{id}` | Job detail |
| Infra | GET | `/v1/console/infra/topology` | Service topology |
| Infra | GET | `/v1/console/infra/services` | List services |
| Infra | GET | `/v1/console/infra/services/{name}` | Service detail |
| Infra | GET | `/v1/console/infra/databases` | DB status |
| Infra | GET | `/v1/console/infra/resources` | Resource usage |
| Infra | GET | `/v1/console/infra/deployments` | Deployment status |
| Observability | GET | `/v1/console/observability/metrics` | Metrics |
| Observability | GET | `/v1/console/observability/traces` | Traces |
| Observability | GET | `/v1/console/observability/traces/{id}` | Trace detail |
| Observability | GET | `/v1/console/observability/errors` | Error aggregation |
| Observability | GET | `/v1/console/observability/costs` | LLM cost report |
| Org | GET/PUT | `/v1/console/org/settings` | Org settings |
| Org | GET | `/v1/console/org/members` | Members |
| Org | GET | `/v1/console/org/roles` | Roles |
| SDK | GET/POST | `/v1/console/sdk/keys` | API keys |
| SDK | DELETE | `/v1/console/sdk/keys/{id}` | Revoke key |
| SDK | GET | `/v1/console/sdk/rate-limits` | Rate limits |
| SDK | GET/POST | `/v1/console/sdk/webhooks` | Webhooks |
| SDK | DELETE | `/v1/console/sdk/webhooks/{id}` | Delete webhook |
| WS | GET | `/v1/console/ws` | WebSocket stream |

---

## 5. MCP Server

**Transport:** SSE (`GET /sse`) và HTTP Streamable (`POST /message`)

**22 tools registered** (từ `gateway/adapter/mcp/server.go`):

| Tool | Description | Backend |
|------|-------------|---------|
| `memory_store` | Store with auto-classification | cognee-ingestion |
| `memory_recall` | Cross-engine semantic recall | vnp-search-hub |
| `memory_search` | Search knowledge graph | cognee-search |
| `memory_timeline` | Temporal event query | vnp-event |
| `memory_profile` | Get user profile context | memobase-context |
| `memory_forget` | Cascading delete across engines | vnp-event |
| `graph_query` | Query knowledge graph | graphiti-store |
| `ov_read_file` | Read file from context DB | ov-fs |
| `ov_write_file` | Write file to context DB | ov-fs |
| `ov_search` | Hierarchical semantic search | ov-search |
| `ov_list_dir` | List directory contents | ov-fs |
| `ov_grep` | Regex search file contents | ov-fs |
| `ov_tree` | Directory tree structure | ov-fs |
| `ov_session_commit` | Commit editing session | ov-session |
| `ov_ingest` | Ingest resource into context DB | ov-resource |
| `ov_delete` | Delete file/resource | ov-fs |
| `cognify` | Transform data into KG | kg-service |
| `search` | Query KG multi-strategy | kg-service |
| `save_interaction` | Log user-agent interaction | kg-service |
| `list_data` | List datasets | kg-service |
| `delete_dataset` | Delete dataset + all data | kg-service |
| `cognify_status` | Check pipeline status | kg-service |

---

## 6. Domain Types

```go
// gateway/domain/entity.go

type AuthContext struct {
    TenantID string; UserID string; Roles []string
    Scopes []string; RateTier string // "free"|"pro"|"enterprise"
}

type StoreRequest struct {
    Type     string            // "auto"|"semantic"|"episodic"|"conversational"|"profile"|"procedural"
    Content  string
    Metadata map[string]string
    SourceID string; UserID string
}

type RouteTarget struct {
    Service string; Address string
    Timeout time.Duration; Method string
}

// MemoryType constants:
// MemoryTypeSemantic, MemoryTypeEpisodic, MemoryTypeConversational
// MemoryTypeProfile, MemoryTypeProcedural, MemoryTypeAuto
```

---

## 7. gRPC Registry

`GRPCRegistry` trong `gateway/adapter/client/registry.go`:
- Manages gRPC connections tới 35 downstream services
- `services map[string]string` từ config (`gateway.services.*` keys)
- Max recv msg size: **16MB**
- Keepalive: 10s ping, 3s timeout
- Credentials: insecure (internal network only)
