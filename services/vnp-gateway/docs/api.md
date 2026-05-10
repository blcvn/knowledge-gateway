---
id: DOC-S02
service: vnp-gateway
version: 1.3.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# vnp-gateway — API Reference

> **Protocol**: REST (external), gRPC-Web, MCP | **Ports**: 8080 (REST), 8081 (gRPC), 8082 (MCP)
> **Source**: `gateway/internal/adapter/handler/router.go`

## Unified Memory API (`/v1/memory/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/memory/store` | Auto-route | Store memory (auto-classified by type) |
| POST | `/v1/memory/recall` | vnp-search-hub | Cross-engine recall |
| POST | `/v1/memory/forget` | Fan-out → All | Cascading delete |
| GET | `/v1/memory/timeline` | vnp-event | Temporal event query |

## Cognee API (`/v1/cognee/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/cognee/datasets` | cognee-ingestion | Create dataset |
| POST | `/v1/cognee/datasets/{id}/data` | cognee-ingestion | Upload data |
| POST | `/v1/cognee/datasets/{id}/cognify` | cognee-cognify | Trigger KG pipeline |
| POST | `/v1/cognee/search` | cognee-search | Search knowledge |

## Graphiti API (`/v1/graphiti/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/graphiti/episodes` | graphiti-ingestion | Ingest episode |
| POST | `/v1/graphiti/search` | graphiti-search | Hybrid search |
| GET | `/v1/graphiti/nodes/{id}` | graphiti-store | Get node |
| GET | `/v1/graphiti/edges/{id}` | graphiti-store | Get edge |

## Memobase API (`/v1/memobase/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/memobase/users/{uid}/blobs` | memobase-ingestion | Insert blob |
| POST | `/v1/memobase/users/{uid}/flush` | memobase-engine | Flush buffer (YOLO merge) |
| GET | `/v1/memobase/users/{uid}/context` | memobase-context | Get context |
| GET | `/v1/memobase/users/{uid}/profiles` | memobase-context | Get profiles |
| GET | `/v1/memobase/users/{uid}/events` | memobase-context | Get event gists |

## OpenViking API (`/v1/ov/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| GET | `/v1/ov/files/{path...}` | ov-fs | Read file |
| PUT | `/v1/ov/files/{path...}` | ov-fs | Write file |
| DELETE | `/v1/ov/files/{path...}` | ov-fs | Delete file |
| GET | `/v1/ov/tree/{path...}` | ov-fs | Directory listing |
| POST | `/v1/ov/grep` | ov-search | Content search |
| POST | `/v1/ov/search` | ov-search | Hierarchical retrieval |
| POST | `/v1/ov/sessions` | ov-session | Create session |
| POST | `/v1/ov/sessions/{id}/messages` | ov-session | Add message |
| POST | `/v1/ov/sessions/{id}/commit` | ov-session | Commit session |
| POST | `/v1/ov/resources/ingest` | ov-resource | Ingest resource |

## Zep API (`/v1/zep/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/zep/users` | zep-user | Create user |
| GET | `/v1/zep/users/{id}` | zep-user | Get user |
| PATCH | `/v1/zep/users/{id}` | zep-user | Update user |
| POST | `/v1/zep/sessions/{id}/memory` | zep-memory | PutMemory |
| GET | `/v1/zep/sessions/{id}/memory` | zep-memory | GetMemory |
| POST | `/v1/zep/graph/search` | zep-graph | Graph search |
| POST | `/v1/zep/sessions/{id}/search` | zep-search | Session search |
| POST | `/v1/zep/graph/facts` | zep-graph | Add fact |
| POST | `/v1/zep/graph/ontology` | zep-graph | Set ontology |

## Supermemory API (`/v1/sm/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/sm/documents` | sm-document | Create document |
| GET | `/v1/sm/documents/{id}` | sm-document | Get document |
| POST | `/v1/sm/memories` | sm-memory | Create memory |
| POST | `/v1/sm/search` | sm-search | Search memories |
| POST | `/v1/sm/rag` | sm-search | RAG query |
| GET | `/v1/sm/profiles/{uid}` | sm-profile | Get profile |
| POST | `/v1/sm/connections` | sm-connector | Create connection |
| POST | `/v1/sm/connections/{id}/sync` | sm-connector | Sync connection |
| POST | `/v1/sm/projects/spaces` | sm-project | Create space |

## Admin API (`/v1/admin/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/admin/tenants` | vnp-admin | Create tenant |
| POST | `/v1/admin/tenants/{id}/keys` | vnp-admin | Issue API key |
| GET | `/v1/admin/health` | vnp-admin | Aggregated health |
| GET | `/v1/admin/metrics` | Prometheus | Metrics endpoint |

**Total: 43 REST endpoints across 8 API namespaces.**

## Auto-Routing Logic (memory.store)

```go
func (g *Gateway) routeStore(req *StoreRequest) Service {
    switch req.Type {
    case "semantic":       return g.cogneeIngestion
    case "episodic":       return g.graphitiIngestion
    case "conversational": return g.memobaseIngestion
    case "procedural":     return g.ovResource
    case "auto":           return g.routeStore(&StoreRequest{Type: g.classify(req.Data)})
    }
}
```

## Authentication

| Method | Header | Description |
|--------|--------|-------------|
| JWT | `Authorization: Bearer <token>` | RS256 JWT with tenant_id claim |
| API Key | `X-API-Key: <key>` | SHA-256 hashed key lookup via vnp-admin |

## Cross-Cutting

| Feature | Implementation | Config Key |
|---------|---------------|------------|
| Rate Limit | Redis sliding window, per-tenant | `RATELIMIT_ENABLED`, `RATELIMIT_FREE_RPM` |
| Circuit Breaker | sony/gobreaker per downstream service | `CB_MAX_FAILURES`, `CB_TIMEOUT` |
| CORS | Configurable origins | `CORS_ALLOWED_ORIGINS` |
| Request ID | UUID v7, X-Request-ID header | — |
| Timeout (default) | 30s | `TIMEOUT_DEFAULT` |
| Timeout (ingestion) | 120s | `TIMEOUT_INGESTION` |
| Timeout (search) | 10s | `TIMEOUT_SEARCH` |
| Timeout (MCP) | 300s | `TIMEOUT_MCP` |
| OTEL Tracing | gRPC collector | `OTEL_ENDPOINT`, `OTEL_SERVICE_NAME` |
| Health Check | Dedicated port | `HEALTH_PORT` (default: 11080) |

## Middleware Chain (from router.go)

```
Request → Recovery → RequestID → CORS → Logger → Handler
```
