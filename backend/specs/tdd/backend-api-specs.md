# VNP Memory — Backend API Specifications

> **Version**: 1.1 — synced from `gateway/adapter/handler/router.go` as of 2026-09-03  
> **Source of truth**: `gateway/adapter/handler/router.go` + handler files  
> **Protocols**: REST/HTTP (8080), gRPC (8081), MCP (8082), Health (8083)

---

## 1. Authentication

All requests require **one** of:
- `Authorization: Bearer <jwt>` — RS256 signed JWT containing `tenant_id`, `user_id`, `roles`
- `X-API-Key: vnp_<key>` — resolved to tenant via `vnp-admin`

### Auto-Injected Headers

| Header | Source | Description |
|--------|--------|-------------|
| `X-Tenant-ID` | Extracted from token/key | Propagated to all downstream services |
| `X-Request-ID` | UUID v7, generated if absent | Correlation ID for tracing |

> **Implemented**: `/v1/auth/*` routes are **fully implemented** in `gateway/adapter/handler/auth.go`.
> Auth handler implements: `Login`, `Logout`, `Refresh`, `LoginWithGoogle`, `Me`.

---

## 2. Unified Memory API — `/v1/memory/*`

### Core Operations
| Method | Path | Handler | Backend Service | Description |
|--------|------|---------|----------------|-------------|
| `POST` | `/v1/memory/store` | `MemoryHandler.Store` | `memory-service` | Store memory (auto-routed by type) |
| `POST` | `/v1/memory/recall` | `MemoryHandler.Recall` | `memory-service` | Cross-engine recall |
| `POST` | `/v1/memory/forget` | `MemoryHandler.Forget` | `memory-service` | Cascading delete |
| `GET` | `/v1/memory/timeline` | `MemoryHandler.Timeline` | `memory-service` | Temporal event query |

**Auto-Routing Rules for `POST /v1/memory/store`:**
| `type` field | Target Service | Memory Engine |
|---|---|---|
| `semantic` | `cognee-ingestion` | Cognee KG |
| `episodic` | `graphiti-ingestion` | Graphiti temporal KG |
| `conversational` | `zep-memory` | Zep context engine |
| `profile` | `memobase-ingestion` | Memobase YOLO profile |
| `procedural` | `ov-resource` | OpenViking context DB |
| `auto` | LLM classifier → re-route | Bifrost-based classification |

### Agent Memory Management — `/v1/memory/agent/*`
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/v1/memory/agent/remember` | `AgentMemoryHandler.RememberAgent` | Store agent memory |
| `GET` | `/v1/memory/agent/list` | `AgentMemoryHandler.ListAgentMemories` | List agent memories |
| `GET` | `/v1/memory/agent/{id}` | `AgentMemoryHandler.GetAgentMemory` | Get agent memory |
| `DELETE` | `/v1/memory/agent/{id}` | `AgentMemoryHandler.DeleteAgentMemory` | Delete agent memory |
| `GET` | `/v1/memory/agent/{id}/retention` | `AgentMemoryHandler.GetRetentionScore` | Salience/retention score |
| `POST` | `/v1/memory/agent/evict` | `AgentMemoryHandler.EvictMemories` | Manual eviction |
| `POST` | `/v1/memory/agent/auto-forget` | `AgentMemoryHandler.AutoForgetSweep` | Auto-forget sweep |

### Memory Slots — `/v1/memory/slots/*`
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/v1/memory/slots` | `AgentMemoryHandler.ListSlots` | List all slots |
| `GET` | `/v1/memory/slots/{scope}/{label}` | `AgentMemoryHandler.GetSlot` | Get slot by scope+label |
| `POST` | `/v1/memory/slots/{scope}/{label}` | `AgentMemoryHandler.WriteSlot` | Write slot value |
| `DELETE` | `/v1/memory/slots/{scope}/{label}` | `AgentMemoryHandler.DeleteSlot` | Delete slot |

### Observe API — `/v1/observe/*` [AgentMemory Layer]
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/v1/observe/sessions` | `AgentMemoryHandler.StartSession` | Start agent session |
| `POST` | `/v1/observe/sessions/{id}/observe` | `AgentMemoryHandler.Observe` | Submit hook event |
| `POST` | `/v1/observe/sessions/{id}/end` | `AgentMemoryHandler.EndSession` | End session |
| `GET` | `/v1/observe/sessions/{id}` | `AgentMemoryHandler.GetSession` | Get session detail |
| `GET` | `/v1/observe/sessions` | `AgentMemoryHandler.ListSessions` | List sessions |
| `DELETE` | `/v1/observe/sessions/{id}` | `AgentMemoryHandler.DeleteSession` | Delete session |
| `GET` | `/v1/observe/sessions/{id}/observations` | `AgentMemoryHandler.GetObservations` | List observations |
| `GET` | `/v1/observe/stream` | `AgentMemoryHandler.StreamEvents` | SSE live stream |

---

## 3. Cognee API — `/v1/cognee/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/cognee/datasets` | `cognee-ingestion` | Create dataset |
| `POST` | `/v1/cognee/datasets/{id}/data` | `cognee-ingestion` | Upload data to dataset |
| `POST` | `/v1/cognee/datasets/{id}/cognify` | `cognee-cognify` | Trigger KG construction pipeline |
| `POST` | `/v1/cognee/search` | `cognee-search` | Search knowledge (15 retrieval strategies) |

---

## 4. Graphiti API — `/v1/graphiti/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/graphiti/episodes` | `graphiti-ingestion` | Ingest episode |
| `POST` | `/v1/graphiti/search` | `graphiti-search` | Hybrid search with temporal filtering |
| `GET` | `/v1/graphiti/nodes/{id}` | `graphiti-store` | Get node by ID |
| `GET` | `/v1/graphiti/edges/{id}` | `graphiti-store` | Get edge by ID |

---

## 5. Memobase API — `/v1/memobase/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/memobase/users/{uid}/blobs` | `memobase-ingestion` | Insert chat/doc/event blob |
| `POST` | `/v1/memobase/users/{uid}/flush` | `memobase-ingestion` | Force flush buffer to engine |
| `GET` | `/v1/memobase/users/{uid}/context` | `memobase-context` | Get assembled context string |
| `GET` | `/v1/memobase/users/{uid}/profiles` | `memobase-context` | Get user profiles |
| `GET` | `/v1/memobase/users/{uid}/events` | `vnp-event` | Get user event timeline |

---

## 6. OpenViking API — `/v1/ov/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/ov/files/{path...}` | `ov-fs` | Read file content |
| `PUT` | `/v1/ov/files/{path...}` | `ov-fs` | Write/update file |
| `DELETE` | `/v1/ov/files/{path...}` | `ov-fs` | Delete file |
| `GET` | `/v1/ov/tree/{path...}` | `ov-fs` | Directory tree listing |
| `POST` | `/v1/ov/grep` | `ov-fs` | Content search (grep) |
| `POST` | `/v1/ov/search` | `ov-search` | Hierarchical retrieval (L0/L1/L2) |
| `POST` | `/v1/ov/sessions` | `ov-session` | Create session |
| `POST` | `/v1/ov/sessions/{id}/messages` | `ov-session` | Add message to session |
| `POST` | `/v1/ov/sessions/{id}/commit` | `ov-session` | 2-phase commit session |
| `POST` | `/v1/ov/resources/ingest` | `ov-resource` | Ingest resource |
| `WebDAV` | `/webdav/*` | — | Full WebDAV protocol (proxy) |

---

## 7. Zep API — `/v1/zep/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/zep/users` | `zep-user` | Create user |
| `GET` | `/v1/zep/users/{id}` | `zep-user` | Get user |
| `PATCH` | `/v1/zep/users/{id}` | `zep-user` | Update user metadata (merge-patch) |
| `POST` | `/v1/zep/sessions/{id}/memory` | `zep-memory` | PutMemory — ingest messages |
| `GET` | `/v1/zep/sessions/{id}/memory` | `zep-memory` | GetMemory — retrieve context |
| `POST` | `/v1/zep/graph/search` | `zep-search` | Graph semantic search |
| `POST` | `/v1/zep/sessions/{id}/search` | `zep-search` | Session-scoped search |
| `POST` | `/v1/zep/graph/facts` | `zep-graph` | Add graph fact |
| `POST` | `/v1/zep/graph/ontology` | `zep-graph` | Set graph ontology |

---

## 8. Supermemory API — `/v1/sm/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/sm/documents` | `sm-document` | Create document |
| `GET` | `/v1/sm/documents/{id}` | `sm-document` | Get document |
| `POST` | `/v1/sm/memories` | `sm-memory` | Create memory (fact extraction) |
| `POST` | `/v1/sm/search` | `sm-search` | Hybrid search (vector + fulltext) |
| `POST` | `/v1/sm/rag` | `sm-search` | RAG completion |
| `GET` | `/v1/sm/profiles/{uid}` | `sm-profile` | Get user profile |
| `POST` | `/v1/sm/connections` | `sm-connector` | Create external connection |
| `POST` | `/v1/sm/connections/{id}/sync` | `sm-connector` | Trigger connection sync |
| `POST` | `/v1/sm/projects/spaces` | `sm-project` | Create space |

---

## 9. Admin & Health APIs — `/v1/admin/*`, `/v1/health`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/admin/tenants` | `vnp-admin` | Create tenant |
| `POST` | `/v1/admin/tenants/{id}/keys` | `vnp-admin` | Issue API key |
| `GET` | `/v1/admin/health` | `vnp-admin` | Aggregated health (all services) |
| `GET` | `/v1/admin/metrics` | `vnp-admin` | System metrics |
| `GET` | `/v1/health` | `memory-platform` | Health snapshot |
| `GET` | `/v1/admin/doctor` | `memory-platform` | Doctor health check |
| `POST` | `/v1/admin/snapshot` | `memory-platform` | Create snapshot |
| `GET` | `/v1/admin/snapshots` | `memory-platform` | List snapshots |
| `GET` | `/v1/admin/plugin/claude-code` | `memory-platform` | Get Claude Code plugin config |
| `GET` | `/v1/admin/plugin/codex` | `memory-platform` | Get Codex plugin config |
| `GET` | `/v1/admin/plugin/opencode` | `memory-platform` | Get OpenCode plugin config |
| `POST` | `/v1/admin/plugin/install` | `memory-platform` | Install plugin |

---

## 10. Observe API — `/v1/observe/*`, `/v1/stream`

> Backend Service: primarily `observe-service`; search sub-routes → `observe-search`

### Session Management
| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/observe/sessions` | `observe-service` | Start an observe session |
| `POST` | `/v1/observe/sessions/{id}/observe` | `observe-service` | Add an observation |
| `POST` | `/v1/observe/sessions/{id}/end` | `observe-service` | End observe session |
| `GET` | `/v1/observe/sessions/{id}` | `observe-service` | Get session details |
| `GET` | `/v1/observe/sessions` | `observe-service` | List sessions |
| `DELETE` | `/v1/observe/sessions/{id}` | `observe-service` | Delete session |
| `GET` | `/v1/observe/sessions/{id}/observations` | `observe-service` | Get observations in session |

### Hook Endpoints (Alternate Paths)
| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/observe` | `observe-service` | General observe hook (short-form) |
| `POST` | `/v1/observe/session/start` | `observe-service` | Start observe session (hook) |
| `POST` | `/v1/observe/session/end` | `observe-service` | End observe session (hook) |

### Replay
| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/observe/replay/sessions` | `observe-service` | List replay sessions |
| `GET` | `/v1/observe/replay/{id}/timeline` | `observe-service` | Load replay timeline |

### Streaming
| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/observe/stream` | `observe-service` | SSE endpoint (observe-scoped events) |
| `GET` | `/v1/stream` | `observe-service` | SSE subscription (global) |

### Observe Search
| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/observe/search/smart` | `observe-search` | Smart search |
| `POST` | `/v1/observe/search/bm25` | `observe-search` | BM25 full-text search |
| `POST` | `/v1/observe/search/vector` | `observe-search` | Vector similarity search |
| `POST` | `/v1/observe/search/context` | `observe-search` | Build search context |
| `POST` | `/v1/observe/search/index` | `observe-search` | Add document to search index |
| `DELETE` | `/v1/observe/search/index/{docId}` | `observe-search` | Remove from search index |
| `POST` | `/v1/observe/search/rebuild` | `observe-search` | Rebuild search index |
| `GET` | `/v1/observe/search/stats` | `observe-search` | Get search index stats |

---

## 11. Orchestration Service — `/v1/orchestration/*`

> Backend Service: `orchestration-service`

### Actions
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/actions` | Create action |
| `GET` | `/v1/orchestration/actions` | List actions |
| `GET` | `/v1/orchestration/actions/{id}` | Get action |
| `PATCH` | `/v1/orchestration/actions/{id}` | Update action |
| `DELETE` | `/v1/orchestration/actions/{id}` | Delete action |

### Leases
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/leases/acquire` | Acquire lease |
| `POST` | `/v1/orchestration/leases/renew` | Renew lease |
| `POST` | `/v1/orchestration/leases/release` | Release lease |
| `GET` | `/v1/orchestration/leases/{actionId}` | Get lease |

### Signals
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/signals/send` | Send signal |
| `GET` | `/v1/orchestration/signals` | List signals |
| `POST` | `/v1/orchestration/signals/{id}/read` | Mark signal read |
| `DELETE` | `/v1/orchestration/signals/{id}` | Delete signal |

### Routines
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/routines` | Create routine |
| `GET` | `/v1/orchestration/routines` | List routines |
| `POST` | `/v1/orchestration/routines/{id}/execute` | Execute routine |

### Checkpoints
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/checkpoints` | Create checkpoint |
| `GET` | `/v1/orchestration/checkpoints` | List checkpoints |
| `POST` | `/v1/orchestration/checkpoints/{id}/approve` | Approve checkpoint |
| `POST` | `/v1/orchestration/checkpoints/{id}/reject` | Reject checkpoint |

### Sentinels
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/sentinels` | Create sentinel |
| `GET` | `/v1/orchestration/sentinels` | List sentinels |
| `DELETE` | `/v1/orchestration/sentinels/{id}` | Delete sentinel |

### Sketches & Crystals
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orchestration/sketches` | Create sketch |
| `GET` | `/v1/orchestration/sketches` | List sketches |
| `POST` | `/v1/orchestration/sketches/{id}/add-action` | Add action to sketch |
| `POST` | `/v1/orchestration/sketches/{id}/promote` | Promote sketch to crystal |
| `GET` | `/v1/orchestration/crystals` | List crystals |
| `GET` | `/v1/orchestration/crystals/{id}` | Get crystal |

---

## 12. Console APIs — `/v1/console/*`

> **Auth**: All console endpoints require `admin` role (`requireAdmin`).  
> Governance endpoints additionally require `super_admin` role (`requireSuperAdmin`).

### 12.1 Dashboard — `/v1/console/dashboard/*`

> Backend Service: `vnp-dashboard`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/console/dashboard/health` | Aggregated engine health (7 engines) |
| `GET` | `/v1/console/dashboard/metrics` | KPI cards (agents, latency, savings) |
| `GET` | `/v1/console/dashboard/throughput` | Per-engine throughput (`?window=5m\|1h\|24h`) |
| `GET` | `/v1/console/dashboard/heatmap` | Memory density heatmap data |

### 12.2 Memory Explorer — `/v1/console/memory/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/console/memory/search` | `vnp-search-hub` | Unified cross-engine search |
| `GET` | `/v1/console/memory/{id}` | `vnp-search-hub` | Memory detail with provenance |
| `GET` | `/v1/console/memory/{id}/neighbors` | `vnp-search-hub` | Graph neighbors |
| `GET` | `/v1/console/memory/{id}/versions` | `sm-memory` | Version chain (Supermemory) |

### 12.3 Graph Studio — `/v1/console/graph/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/console/graph/subgraph` | `graphiti-store` | Query subgraph by entity |
| `GET` | `/v1/console/graph/entity/{id}` | `graphiti-store` | Entity detail with neighbors |
| `POST` | `/v1/console/graph/timeline` | `graphiti-store` | Temporal subgraph for time range |
| `GET` | `/v1/console/graph/ontology` | `cognee-search` | Get ontology schema |
| `PUT` | `/v1/console/graph/ontology` | `cognee-search` | Update ontology schema |
| `POST` | `/v1/console/graph/query` | `graphiti-store` | Execute Cypher/NL query |

### 12.4 User Profiles — `/v1/console/profiles/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/console/profiles` | `memobase-context` | List user profiles (paginated) |
| `GET` | `/v1/console/profiles/{user_id}` | `memobase-context` | Profile detail |
| `GET` | `/v1/console/profiles/{user_id}/events` | `vnp-event` | Event timeline |
| `GET` | `/v1/console/profiles/{user_id}/context` | `memobase-context` | Context assembly preview |
| `GET` | `/v1/console/profiles/{user_id}/buffers` | `memobase-ingestion` | Buffer zone status |
| `GET` | `/v1/console/profiles/config` | `memobase-context` | Profile schema config |
| `PUT` | `/v1/console/profiles/config` | `memobase-context` | Update profile schema |

### 12.5 Adaptive Memory — `/v1/console/adaptive/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/console/adaptive/memories` | `sm-memory` | List adaptive memories |
| `GET` | `/v1/console/adaptive/memories/{id}/versions` | `sm-memory` | Version chain |
| `GET` | `/v1/console/adaptive/connectors` | `sm-connector` | List external connectors |
| `POST` | `/v1/console/adaptive/connectors` | `sm-connector` | Create connector |
| `POST` | `/v1/console/adaptive/connectors/{id}/sync` | `sm-connector` | Trigger sync |
| `GET` | `/v1/console/adaptive/analytics` | `sm-engine` | Adaptive memory analytics |
| `GET` | `/v1/console/adaptive/forget-rules` | `sm-engine` | Auto-forget rules |
| `PUT` | `/v1/console/adaptive/forget-rules` | `sm-engine` | Update forget rules |

### 12.6 Context Debugger — `/v1/console/debugger/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `POST` | `/v1/console/debugger/trace` | `vnp-search-hub` | Simulate context assembly |
| `GET` | `/v1/console/debugger/traces/{id}` | `vnp-event` | Get saved trace |
| `GET` | `/v1/console/debugger/traces` | `vnp-event` | List recent traces |

### 12.7 Sessions — `/v1/console/sessions/*`

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/console/sessions` | `zep-core` | List sessions (paginated) |
| `GET` | `/v1/console/sessions/live` | `zep-core` | Active live sessions |
| `GET` | `/v1/console/sessions/{id}` | `zep-core` | Session detail with messages |
| `GET` | `/v1/console/sessions/{id}/timeline` | `zep-core` | Session replay timeline |
| `GET` | `/v1/console/sessions/{id}/diff` | `vnp-event` | Memory diff (before vs after) |
| `GET` | `/v1/console/sessions/{id}/working-memory` | `ov-session` | Working memory state |
| `GET` | `/v1/console/sessions/{id}/user-summary` | `memobase-context` | User memory summary |

### 12.8 Governance — `/v1/console/governance/*`

> **Auth**: `super_admin` role required (stricter than other console routes)

| Method | Path | Backend Service | Description |
|--------|------|----------------|-------------|
| `GET` | `/v1/console/governance/tenants` | `vnp-admin` | List tenants |
| `POST` | `/v1/console/governance/tenants` | `vnp-admin` | Create tenant |
| `PUT` | `/v1/console/governance/tenants/{id}` | `vnp-admin` | Update tenant |
| `GET` | `/v1/console/governance/policies` | `vnp-admin` | List OPA policies |
| `POST` | `/v1/console/governance/policies` | `vnp-admin` | Create policy |
| `PUT` | `/v1/console/governance/policies/{id}` | `vnp-admin` | Update policy |
| `GET` | `/v1/console/governance/audit` | `vnp-admin` | Search audit logs |
| `POST` | `/v1/console/governance/gdpr/forget` | `vnp-event` | GDPR cascading forget |
| `POST` | `/v1/console/governance/gdpr/forget/preview` | `vnp-event` | Dry-run forget preview |

### 12.9 Pipelines — `/v1/console/pipelines/*`

> Backend Service: `vnp-pipelines`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/console/pipelines/status` | All engines pipeline overview |
| `GET` | `/v1/console/pipelines/queues` | Queue metrics across engines |
| `GET` | `/v1/console/pipelines/workers` | Worker status |
| `GET` | `/v1/console/pipelines/templates` | Pipeline templates |
| `GET` | `/v1/console/pipelines/{engine}` | Engine pipeline status |
| `GET` | `/v1/console/pipelines/{engine}/jobs` | Active/recent jobs |
| `GET` | `/v1/console/pipelines/{engine}/jobs/{id}` | Job detail with stages |

### 12.10 Infrastructure — `/v1/console/infra/*`

> Backend Service: `vnp-infra`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/console/infra/topology` | Service topology graph |
| `GET` | `/v1/console/infra/services` | All services status |
| `GET` | `/v1/console/infra/services/{name}` | Service detail |
| `GET` | `/v1/console/infra/databases` | DB health (PG, Neo4j, Redis, NATS) |
| `GET` | `/v1/console/infra/resources` | Resource usage per service |
| `GET` | `/v1/console/infra/deployments` | Deployment timeline |

### 12.11 Observability — `/v1/console/observability/*`

> Backend Service: `vnp-observability`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/console/observability/metrics` | Aggregated metrics per engine |
| `GET` | `/v1/console/observability/traces` | List distributed traces |
| `GET` | `/v1/console/observability/traces/{id}` | Trace detail with spans |
| `GET` | `/v1/console/observability/errors` | Error explorer |
| `GET` | `/v1/console/observability/costs` | Cost analytics (LLM, tokens) |

### 12.12 WebSocket Realtime — `/v1/console/ws`

| Protocol | Path | Description |
|----------|------|-------------|
| `WS` | `/v1/console/ws?token=<jwt>` | Authenticated WebSocket |

**Channels:** `engine.health`, `memory.flow`, `pipeline.progress`, `alerts`

---

## 13. MCP Server (Port 8082)

**Transport**: stdio, SSE, HTTP Streamable (JSON-RPC 2.0)

| Tool Name | Description | Target Service |
|-----------|-------------|---------------|
| `memory_store` | Store memory (auto-route) | Auto-route |
| `memory_recall` | Cross-engine recall | `vnp-search-hub` |
| `memory_search` | Semantic search | `cognee-search` |
| `memory_timeline` | Temporal events | `vnp-event` |
| `memory_profile` | Get user profile | `memobase-context` |
| `memory_forget` | Delete memory | Fan-out |
| `graph_query` | Knowledge graph query | `graphiti-store` |
| `ov_read_file` | Read file | `ov-fs` |
| `ov_write_file` | Write file | `ov-fs` |
| `ov_search` | Hierarchical search | `ov-search` |
| `ov_list_dir` | List directory | `ov-fs` |
| `ov_grep` | Content grep | `ov-fs` |
| `ov_tree` | Directory tree | `ov-fs` |
| `ov_session_commit` | Commit session | `ov-session` |
| `ov_ingest` | Ingest resource | `ov-resource` |
| `ov_delete` | Delete file | `ov-fs` |

---

## 14. Error Response Format

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Human-readable error message",
    "details": [
      { "field": "query", "reason": "must not be empty" }
    ],
    "request_id": "01907c3a-..."
  }
}
```

| HTTP Status | gRPC Code | Description |
|------------|-----------|-------------|
| 400 | `INVALID_ARGUMENT` | Bad request parameters |
| 401 | `UNAUTHENTICATED` | Missing/invalid auth |
| 403 | `PERMISSION_DENIED` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 429 | `RESOURCE_EXHAUSTED` | Rate limit exceeded |
| 500 | `INTERNAL` | Server error |
| 503 | `UNAVAILABLE` | Service unavailable (circuit open) |
| 504 | `DEADLINE_EXCEEDED` | Timeout |

---

## 15. Rate Limiting

| Tier | Requests/min | Burst |
|------|-------------|-------|
| Free | 60 | 10 |
| Pro | 600 | 50 |
| Enterprise | 6000 | 200 |

Response headers when rate limited:
```
X-RateLimit-Limit: 600
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1683820800
Retry-After: 30
```

---

## 16. Known Gaps (Missing from Backend)

> These are endpoints required by the frontend but **not yet implemented** in the gateway router.  
> See [CR-001](../crs/v2/api-update/CR-001-frontend-api-alignment.md) for the full Change Request.

| Gap | Status | CR |
|-----|--------|----|
| `POST /v1/auth/login` — Login with email/password | ❌ Missing | CR-001 |
| `POST /v1/auth/logout` — Logout & invalidate session | ❌ Missing | CR-001 |
| `GET /v1/auth/me` — Get current user profile | ❌ Missing | CR-001 |
| `POST /v1/auth/refresh` — Refresh access token | ❌ Missing | CR-001 |
| `GET /v1/console/org/settings` — Get org settings | ❌ Missing | CR-001 |
| `PUT /v1/console/org/settings` — Update org settings | ❌ Missing | CR-001 |
| `GET /v1/console/org/members` — List org members | ❌ Missing | CR-001 |
| `GET /v1/console/org/roles` — List org roles | ❌ Missing | CR-001 |
| `GET /v1/console/sdk/keys` — List SDK API keys | ❌ Missing | CR-001 |
| `POST /v1/console/sdk/keys` — Create SDK API key | ❌ Missing | CR-001 |
| `DELETE /v1/console/sdk/keys/{id}` — Revoke API key | ❌ Missing | CR-001 |
| `GET /v1/console/sdk/rate-limits` — Get rate limit configs | ❌ Missing | CR-001 |
| `GET /v1/console/sdk/webhooks` — List webhooks | ❌ Missing | CR-001 |
| `POST /v1/console/sdk/webhooks` — Create webhook | ❌ Missing | CR-001 |
| `DELETE /v1/console/sdk/webhooks/{id}` — Delete webhook | ❌ Missing | CR-001 |
| `GET /v1/console/sessions` query params: `status`, `user_id`, `agent_id`, `search`, `sort`, `page`, `page_size` | ⚠️ Params undocumented | CR-001 |
