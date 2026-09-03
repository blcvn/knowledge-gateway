# Change Request: CR-GR-006 — API Gateway & MCP Server Protocol Translation

**CR ID:** CR-GR-006  
**Component:** `gateway` [EXTEND] | MCP Server [EXTEND]  
**Priority:** High  
**Status:** In Progress
**Reference:** graphiti PRD §5.5, SRS §5.1–5.2, specs/services/01-gateway-service.md  
**Maps to Python:** `server/graph_service/` (FastAPI) + `mcp_server/` (FastMCP)

---

## 1. Mô tả

Mở rộng **graphiti-gateway** (đã có trong VNP Memory) để:
1. **REST API** đầy đủ cho graphiti operations (episodes, search, entities, edges, sagas).
2. **MCP Server** với 9 graphiti tools cho Claude Code, Cursor, Claude Desktop.
3. **Protocol translation**: REST/MCP → gRPC → graphiti-ingestion/search/store/knowledge/admin.
4. **JWT Authentication** + **Rate Limiting** per tenant.
5. **Multi-tenant** routing via `X-Group-ID` header → gRPC metadata propagation.

---

## 2. Vấn đề hiện tại

`gateway` hiện tại trong VNP Memory:
- ✅ Có basic REST routing và JWT auth.
- ❌ Thiếu **graphiti REST endpoints** (episodes, sagas, entities, edges).
- ❌ MCP server có 6 tools cơ bản, thiếu graphiti-specific tools.
- ❌ Không có **`/healthz`** endpoint chi tiết.
- ❌ Không có **group_id propagation** qua gRPC metadata.
- ❌ Không có **rate limiting** per-tenant cho ingestion.

---

## 3. Thay đổi đề xuất

### 3.1. [EXTEND] `gateway/internal/adapter/http/` — Graphiti REST Handlers

```go
// gateway/internal/adapter/http/graphiti/
// New router group: /v1/graphiti/

// Ingestion routes
POST   /v1/graphiti/episodes                    → ingestion.IngestEpisode
POST   /v1/graphiti/episodes/bulk               → ingestion.IngestEpisodeBulk
DELETE /v1/graphiti/episodes/{uuid}             → ingestion.RemoveEpisode
GET    /v1/graphiti/episodes                    → ingestion.ListEpisodes
GET    /v1/graphiti/episodes/{uuid}             → ingestion.GetEpisode
POST   /v1/graphiti/triplets                    → ingestion.AddTriplet

// Saga routes
POST   /v1/graphiti/sagas                       → ingestion.CreateSaga
POST   /v1/graphiti/sagas/{id}/summarize        → ingestion.SummarizeSaga
GET    /v1/graphiti/sagas/{id}                  → ingestion.GetSaga

// Search routes
POST   /v1/graphiti/search                      → search.Search (hybrid RRF, default)
POST   /v1/graphiti/search/advanced             → search.SearchAdvanced
GET    /v1/graphiti/search/nodes                → search.SearchNodes
GET    /v1/graphiti/search/edges                → search.SearchEdges
GET    /v1/graphiti/search/episodes             → search.SearchEpisodes
GET    /v1/graphiti/search/communities          → search.SearchCommunities

// Retrieve routes
GET    /v1/graphiti/entities/{uuid}             → store.GetEntityNode
GET    /v1/graphiti/edges/{uuid}                → store.GetEntityEdge

// Ontology routes
POST   /v1/graphiti/ontology/{group_id}         → knowledge.SaveOntology
GET    /v1/graphiti/ontology/{group_id}         → knowledge.GetOntology
DELETE /v1/graphiti/ontology/{group_id}         → knowledge.DeleteOntology
POST   /v1/graphiti/ontology/{group_id}/preset  → knowledge.ApplyPreset

// Admin routes
POST   /v1/graphiti/admin/communities           → admin.RebuildCommunities
POST   /v1/graphiti/admin/indices               → admin.BuildIndices
DELETE /v1/graphiti/admin/data/{group_id}       → admin.ClearData
GET    /v1/graphiti/admin/token-usage           → knowledge.GetTokenUsage

// Health
GET    /healthz                                 → aggregate health check
```

### 3.2. [EXTEND] MCP Server — 9 Graphiti Tools

```go
// gateway/internal/adapter/mcp/tools/graphiti/

// Tool 1: add_memory — ingest episode
var addMemoryTool = mcp.Tool{
    Name: "add_memory",
    Description: "Add a new memory (episode) to the knowledge graph. The memory will be automatically processed: entities extracted, relationships identified, and temporal context maintained.",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "content": {Type: "string", Description: "The content/body of the memory to add"},
            "source_description": {Type: "string", Description: "Brief description of where this memory came from"},
            "group_id": {Type: "string", Description: "Namespace/partition for multi-tenant isolation"},
        },
        Required: []string{"content", "source_description", "group_id"},
    },
    Handler: handleAddMemory,  // → POST /v1/graphiti/episodes
}

// Tool 2: search_memory — hybrid search
var searchMemoryTool = mcp.Tool{
    Name: "search_memory",
    Description: "Search the knowledge graph using hybrid retrieval (semantic + keyword + graph traversal). Returns relevant facts, entities, and relationships.",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "query":       {Type: "string"},
            "group_ids":   {Type: "array", Items: &mcp.Property{Type: "string"}},
            "num_results": {Type: "integer", Default: 10},
        },
        Required: []string{"query"},
    },
    Handler: handleSearchMemory,  // → POST /v1/graphiti/search
}

// Tool 3: get_episodes — list recent episodes
var getEpisodesTool = mcp.Tool{
    Name:        "get_episodes",
    Description: "Retrieve recent episodes from the knowledge graph",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "group_id": {Type: "string"},
            "last_n":   {Type: "integer", Default: 10},
        },
    },
    Handler: handleGetEpisodes,  // → GET /v1/graphiti/episodes
}

// Tool 4: delete_episode
var deleteEpisodeTool = mcp.Tool{
    Name:        "delete_episode",
    Description: "Delete an episode and cascade remove its episodic edges",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "episode_uuid": {Type: "string"},
        },
        Required: []string{"episode_uuid"},
    },
    Handler: handleDeleteEpisode,  // → DELETE /v1/graphiti/episodes/{uuid}
}

// Tool 5: delete_entity_node
var deleteEntityNodeTool = mcp.Tool{
    Name:        "delete_entity_node",
    Description: "Delete an entity node and all its associated edges from the knowledge graph",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "node_uuid": {Type: "string"},
        },
    },
    Handler: handleDeleteEntityNode,
}

// Tool 6: delete_entity_edge
var deleteEntityEdgeTool = mcp.Tool{
    Name:        "delete_entity_edge",
    Description: "Delete a specific fact/relationship edge from the knowledge graph",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "edge_uuid": {Type: "string"},
        },
    },
    Handler: handleDeleteEntityEdge,
}

// Tool 7: get_entity_edge
var getEntityEdgeTool = mcp.Tool{
    Name:        "get_entity_edge",
    Description: "Retrieve full details of a specific fact/relationship edge including temporal fields",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "edge_uuid": {Type: "string"},
        },
    },
    Handler: handleGetEntityEdge,  // → GET /v1/graphiti/edges/{uuid}
}

// Tool 8: clear_graph
var clearGraphTool = mcp.Tool{
    Name:        "clear_graph",
    Description: "Clear ALL data for a specific group_id. This is irreversible.",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "group_id": {Type: "string"},
        },
        Required: []string{"group_id"},
    },
    Handler: handleClearGraph,  // → DELETE /v1/graphiti/admin/data/{group_id}
}

// Tool 9: get_status
var getStatusTool = mcp.Tool{
    Name:        "get_status",
    Description: "Get current server health status including all graphiti services",
    Handler:     handleGetStatus,  // → GET /healthz
}
```

### 3.3. Multi-Tenant Propagation

```go
// gateway/internal/middleware/group_id.go
// Extract group_id from:
// 1. JWT claim: "group_id" or "tenant_id"
// 2. Header: X-Group-ID (fallback)
// 3. Query param: ?group_id=... (for search endpoints)
// 4. Default: "default" (dev mode only)

// Propagate to gRPC:
// outgoingCtx = metadata.AppendToOutgoingContext(ctx, "x-group-id", groupID)

// All downstream services read from gRPC metadata
```

### 3.4. Rate Limiting (per-tenant per-endpoint)

```go
// Rate limits (Redis sliding window):
// POST /v1/graphiti/episodes      → 100 req/min per tenant
// POST /v1/graphiti/episodes/bulk → 10 req/min per tenant
// POST /v1/graphiti/search        → 500 req/min per tenant
// DELETE /v1/graphiti/admin/data  → 5 req/hour per tenant (destructive)

// Response when rate limited:
// HTTP 429 Too Many Requests
// Retry-After header: next available window
```

### 3.5. Health Check `/healthz`

```go
// GET /healthz → aggregate health across all graphiti services

type HealthResponse struct {
    Status    string                    `json:"status"`  // healthy | degraded | unhealthy
    Services  map[string]ServiceHealth `json:"services"`
    Timestamp time.Time                `json:"timestamp"`
}

type ServiceHealth struct {
    Status  string `json:"status"`
    Latency int64  `json:"latency_ms"`
    Error   string `json:"error,omitempty"`
}

// Services checked:
// graphiti-ingestion: gRPC ping
// graphiti-search: gRPC ping
// graphiti-knowledge: gRPC ping + LLM provider check
// graphiti-store: gRPC ping + DB ping
// graphiti-admin: gRPC ping
// Redis: PING command
// NATS: connection check
// Neo4j (via store): Bolt ping

// Returns 200 if all healthy
// Returns 503 if any critical service unhealthy
```

### 3.6. JWT Authentication

```go
// RS256 JWT validation (same as existing VNP Memory auth)
// Claims used:
// - sub: user/service identifier
// - tenant_id or group_id: multi-tenant partition
// - scope: "read" | "write" | "admin"

// Scope enforcement:
// GET endpoints: require "read" scope
// POST/DELETE endpoints: require "write" scope
// Admin endpoints: require "admin" scope
```

---

## 4. Acceptance Criteria

- [ ] MCP tool `add_memory` từ Claude Desktop → episode được ingest, entities visible trong graph.
- [ ] MCP tool `search_memory` query "who works at Acme" → trả về relevant edges trong ≤ 1s.
- [ ] MCP tool `get_episodes` → trả về list 10 most recent episodes.
- [ ] MCP tool `get_status` → trả về `{status: "healthy", services: {...}}`.
- [ ] `POST /v1/graphiti/episodes` với valid JWT → 200 OK.
- [ ] `POST /v1/graphiti/episodes` không có JWT → 401 Unauthorized.
- [ ] `POST /v1/graphiti/episodes` 101 lần trong 1 phút → 102nd request → 429 Too Many Requests.
- [ ] `X-Group-ID: tenant-A` → data isolated từ `X-Group-ID: tenant-B`.
- [ ] `GET /healthz` khi Neo4j down → `{status: "unhealthy", services: {store: {status: "unhealthy"}}}` với HTTP 503.
- [ ] `DELETE /v1/graphiti/admin/data/group-alpha` → xóa tất cả data của group-alpha; `search` cho group-alpha → empty results.
