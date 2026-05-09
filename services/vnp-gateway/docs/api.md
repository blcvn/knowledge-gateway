---
id: DOC-S02
service: vnp-gateway
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — API Reference

> **Protocol**: REST (external), gRPC-Web, MCP | **Port**: 8080/8081/8082

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

## Memobase API (`/v1/memobase/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/memobase/users/{uid}/blobs` | memobase-ingestion | Insert blob |
| GET | `/v1/memobase/users/{uid}/context` | memobase-context | Get context |
| GET | `/v1/memobase/users/{uid}/profiles` | memobase-context | Get profiles |

## OpenViking API (`/v1/ov/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| GET | `/v1/ov/files/{path}` | ov-fs | Read file |
| PUT | `/v1/ov/files/{path}` | ov-fs | Write file |
| POST | `/v1/ov/search` | ov-search | Hierarchical retrieval |
| POST | `/v1/ov/sessions` | ov-session | Create session |

## Admin API (`/v1/admin/*`)

| Method | Path | Target | Description |
|--------|------|--------|-------------|
| POST | `/v1/admin/tenants` | vnp-admin | Create tenant |
| POST | `/v1/admin/tenants/{id}/keys` | vnp-admin | Issue API key |
| GET | `/v1/admin/health` | vnp-admin | Aggregated health |

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
| API Key | `X-API-Key: <key>` | SHA-256 hashed key lookup |

## Cross-Cutting

| Feature | Implementation |
|---------|---------------|
| Rate Limit | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | sony/gobreaker per downstream service |
| CORS | Configurable origins |
| Request ID | UUID v7, X-Request-ID header |
| Timeout | 30s default, 120s ingestion, 10s search |
