# TASK-CONSOLE-003 — Memory Explorer Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-CONSOLE-003 |
| **Wave** | 2 |
| **Solution** | [SOL-CONSOLE-002](../solutions/SOL-CONSOLE-002-Memory-Explorer-APIs.md) §2 |
| **Component** | `gateway/adapter/handler/memory_explorer_handler.go` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Mục tiêu

Implement 4 Memory Explorer endpoints: search, detail, neighbors, versions.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/memory_explorer_handler.go` [NEW]

```go
type MemoryExplorerHandler struct {
    registry  port.GRPCRegistry
    searchHub vnpsearchpb.VnpSearchHubServiceClient
}

// POST /v1/console/memory/search
// → vnp-search-hub.Recall() với cross-engine RRF

// GET /v1/console/memory/{id}?engine=graphiti
// → route to engine-specific service (engineSearchService helper)

// GET /v1/console/memory/{id}/neighbors
// → graphiti-search.GetSubgraph(seed=id, max_depth=1)

// GET /v1/console/memory/{id}/versions
// → sm-memory.GetVersionHistory(memory_id=id)
```

`engineSearchService(engine)` helper:
```go
func engineSearchService(engine string) string {
    return map[string]string{
        "graphiti": "graphiti-search", "cognee": "cognee-search",
        "zep": "zep-search", "memobase": "memobase-context",
        "openviking": "ov-search", "supermemory": "sm-search",
    }[engine]
}
```

### 2. Routes `router.go` [MODIFY]

```go
r.Post("/v1/console/memory/search",          explorerHandler.Search)
r.Get("/v1/console/memory/{id}",             explorerHandler.GetMemory)
r.Get("/v1/console/memory/{id}/neighbors",   explorerHandler.GetNeighbors)
r.Get("/v1/console/memory/{id}/versions",    explorerHandler.GetVersions)
```

---

## Acceptance Criteria

- [ ] Search with `types: ["graphiti"]` only queries graphiti
- [ ] `engine` param required for `/memory/{id}` (400 if missing)
- [ ] Neighbors returns graph edges with relationship labels
- [ ] Versions in chronological order (oldest first)
- [ ] p95 < 500ms

## Files

```
gateway/adapter/handler/memory_explorer_handler.go  [NEW]
gateway/adapter/handler/router.go                   [MODIFY]
```
