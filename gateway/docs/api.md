---
id: DOC-S02
service: vnp-gateway
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — API Reference

> **Protocols**: REST (8080), gRPC (8081), MCP (8082), WebDAV (8080/webdav)

---

## 1. Authentication

All requests require one of:
- **JWT Bearer Token**: `Authorization: Bearer <token>` — RS256 signed, contains `tenant_id`, `user_id`, `roles`
- **API Key**: `X-API-Key: vnp_<key>` — resolved to tenant via `vnp-admin` lookup

### Required Headers (All Requests)

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Authorization` | string | Yes* | JWT Bearer token |
| `X-API-Key` | string | Yes* | API key (alternative to JWT) |
| `X-Tenant-ID` | string | Auto | Extracted from token/key, propagated to services |
| `X-Request-ID` | string | Auto | UUID v7, generated if not provided |

> *Either `Authorization` or `X-API-Key` is required, not both.

---

## 2. Unified Memory API (`/v1/memory/*`)

### POST `/v1/memory/store` — Store Memory (Auto-Routed)

Stores data in the appropriate engine based on content classification.

**Request:**
```json
{
  "type": "auto | semantic | episodic | conversational | profile | procedural",
  "data": {
    "content": "string — the memory content",
    "metadata": {},
    "source_id": "optional — source identifier",
    "user_id": "optional — target user"
  }
}
```

**Auto-Routing Rules:**
| Type | Target Service | Engine |
|------|---------------|--------|
| `semantic` | cognee-ingestion | Cognee |
| `episodic` | graphiti-ingestion | Graphiti |
| `conversational` | memobase-ingestion | Memobase |
| `profile` | memobase-ingestion | Memobase |
| `procedural` | ov-resource | OpenViking |
| `auto` | LLM classifier → re-route | Auto |

**Response (200):**
```json
{
  "id": "mem_abc123",
  "engine": "cognee",
  "status": "accepted",
  "created_at": "2026-05-09T12:00:00Z"
}
```

### POST `/v1/memory/recall` — Cross-Engine Recall

**Request:**
```json
{
  "query": "string — search query",
  "engines": ["cognee", "graphiti", "zep"],
  "limit": 10,
  "reranking": "rrf | mmr | cross_encoder"
}
```

**Response (200):**
```json
{
  "results": [
    {
      "id": "res_001",
      "engine": "cognee",
      "content": "...",
      "score": 0.95,
      "metadata": {}
    }
  ],
  "total": 42,
  "latency_ms": 180
}
```

### POST `/v1/memory/forget` — Cascading Delete

**Request:**
```json
{
  "target_id": "mem_abc123",
  "engines": ["all"],
  "cascade": true
}
```

### GET `/v1/memory/timeline` — Temporal Event Query

**Query params:** `?user_id=xxx&from=2026-01-01&to=2026-05-09&limit=50`

---

## 3. Cognee API (`/v1/cognee/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognee/datasets` | Create dataset |
| POST | `/v1/cognee/datasets/{id}/data` | Upload data to dataset |
| POST | `/v1/cognee/datasets/{id}/cognify` | Trigger KG construction pipeline |
| POST | `/v1/cognee/search` | Search knowledge (15 retrieval strategies) |

### POST `/v1/cognee/search`

**Request:**
```json
{
  "query": "What are the key features?",
  "strategy": "GRAPH_COMPLETION | SEMANTIC | HYBRID | CHUNKS | SUMMARIES",
  "dataset_ids": ["ds_001"],
  "limit": 10
}
```

---

## 4. Graphiti API (`/v1/graphiti/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/graphiti/episodes` | Ingest episode |
| POST | `/v1/graphiti/search` | Hybrid search with temporal filtering |
| GET | `/v1/graphiti/nodes/{id}` | Get node by ID |
| GET | `/v1/graphiti/edges/{id}` | Get edge by ID |

---

## 5. Memobase API (`/v1/memobase/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/memobase/users/{uid}/blobs` | Insert chat/doc/event blob |
| POST | `/v1/memobase/users/{uid}/flush` | Force flush buffer to engine |
| GET | `/v1/memobase/users/{uid}/context` | Get assembled context string |
| GET | `/v1/memobase/users/{uid}/profiles` | Get user profiles |
| GET | `/v1/memobase/users/{uid}/events` | Get user event timeline |

---

## 6. OpenViking API (`/v1/ov/*`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/ov/files/{path}` | Read file content |
| PUT | `/v1/ov/files/{path}` | Write/update file |
| DELETE | `/v1/ov/files/{path}` | Delete file |
| GET | `/v1/ov/tree/{path}` | Directory tree listing |
| POST | `/v1/ov/grep` | Content search (grep) |
| POST | `/v1/ov/search` | Hierarchical retrieval (L0/L1/L2) |
| POST | `/v1/ov/sessions` | Create session |
| POST | `/v1/ov/sessions/{id}/messages` | Add message to session |
| POST | `/v1/ov/sessions/{id}/commit` | 2-phase commit session |
| POST | `/v1/ov/resources/ingest` | Ingest resource |
| WebDAV | `/webdav/*` | Full WebDAV protocol for file access |

---

## 7. Zep API (`/v1/zep/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/zep/users` | Create user |
| GET | `/v1/zep/users/{id}` | Get user |
| PATCH | `/v1/zep/users/{id}` | Update user metadata (merge-patch) |
| POST | `/v1/zep/sessions/{id}/memory` | PutMemory — ingest messages |
| GET | `/v1/zep/sessions/{id}/memory` | GetMemory — retrieve context (<200ms) |
| POST | `/v1/zep/graph/search` | Graph semantic search |
| POST | `/v1/zep/sessions/{id}/search` | Session-scoped search |
| POST | `/v1/zep/graph/facts` | Add graph fact |
| POST | `/v1/zep/graph/ontology` | Set graph ontology |

---

## 8. Supermemory API (`/v1/sm/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/sm/documents` | Create document |
| GET | `/v1/sm/documents/{id}` | Get document |
| POST | `/v1/sm/memories` | Create memory (fact extraction) |
| POST | `/v1/sm/search` | Hybrid search (vector + fulltext) |
| POST | `/v1/sm/rag` | RAG completion |
| GET | `/v1/sm/profiles/{uid}` | Get user profile |
| POST | `/v1/sm/connections` | Create external connection |
| POST | `/v1/sm/connections/{id}/sync` | Trigger connection sync |
| POST | `/v1/sm/projects/spaces` | Create space |

---

## 9. Admin API (`/v1/admin/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/admin/tenants` | Create tenant |
| POST | `/v1/admin/tenants/{id}/keys` | Issue API key |
| GET | `/v1/admin/health` | Aggregated health (all 35 services) |
| GET | `/v1/admin/metrics` | System metrics |

---

## 10. MCP Server (Port 8082)

**Transport**: stdio, SSE, HTTP Streamable (JSON-RPC 2.0)

### Tools

| Tool Name | Description | Target Service |
|-----------|-------------|---------------|
| `memory_store` | Store memory (auto-route) | Auto-route |
| `memory_recall` | Cross-engine recall | vnp-search-hub |
| `memory_search` | Semantic search | cognee-search |
| `memory_timeline` | Temporal events | vnp-event |
| `memory_profile` | Get user profile | memobase-context |
| `memory_forget` | Delete memory | Fan-out |
| `graph_query` | Knowledge graph query | graphiti-store |
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

## 11. Error Response Format

All errors follow a consistent JSON format:

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Human-readable error message",
    "details": [
      {
        "field": "query",
        "reason": "must not be empty"
      }
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

## 12. Rate Limiting

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
