# TASK-CORE-001 — Memory Domain Types & Routing Table

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-001 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-CORE-001](../solutions/SOL-CORE-001-Unified-Memory-Router.md) §2.2 |
| **Component** | `gateway/domain/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Mục tiêu

Tạo domain types và routing table cho Unified Memory Router trong Gateway.

---

## Công việc cụ thể

### 1. Tạo `gateway/domain/routing.go` [NEW]

```go
package domain

const (
    MemoryTypeEpisodic       = "episodic"
    MemoryTypeSemantic       = "semantic"
    MemoryTypeConversational = "conversational"
    MemoryTypeProfile        = "profile"
    MemoryTypeProcedural     = "procedural"
    MemoryTypeAdaptive       = "adaptive"
    MemoryTypeAuto           = "auto"
)

// EngineService maps MemoryType to InProcessRegistry service name
func EngineService(t string) string {
    table := map[string]string{
        MemoryTypeEpisodic:       "graphiti-ingestion",
        MemoryTypeSemantic:       "cognee-ingestion",
        MemoryTypeConversational: "zep-memory",
        MemoryTypeProfile:        "memobase-ingestion",
        MemoryTypeProcedural:     "ov-resource",
        MemoryTypeAdaptive:       "sm-memory",
    }
    return table[t]
}

// EngineName returns human-readable engine name for response
func EngineName(t string) string {
    table := map[string]string{
        MemoryTypeEpisodic:       "graphiti",
        MemoryTypeSemantic:       "cognee",
        MemoryTypeConversational: "zep",
        MemoryTypeProfile:        "memobase",
        MemoryTypeProcedural:     "openviking",
        MemoryTypeAdaptive:       "supermemory",
    }
    return table[t]
}

// ValidMemoryType checks if type is a valid MemoryType
func ValidMemoryType(t string) bool {
    valid := map[string]bool{
        MemoryTypeEpisodic: true, MemoryTypeSemantic: true,
        MemoryTypeConversational: true, MemoryTypeProfile: true,
        MemoryTypeProcedural: true, MemoryTypeAdaptive: true,
        MemoryTypeAuto: true,
    }
    return valid[t]
}
```

### 2. Tạo `gateway/domain/entity.go` [MODIFY] — Add StoreRequest/StoreResponse

```go
// Add to existing entity.go

type StoreRequest struct {
    UserID   string         `json:"user_id"`
    Content  string         `json:"content"`
    Type     string         `json:"type"`             // one of MemoryType* constants
    Metadata map[string]any `json:"metadata,omitempty"`
}

type StoreResponse struct {
    ID     string `json:"id"`
    Type   string `json:"type"`    // resolved type (if auto)
    Engine string `json:"engine"`  // engine selected
    Status string `json:"status"`  // "processing"
}

type RecallRequest struct {
    UserID    string     `json:"user_id"`
    Query     string     `json:"query"`
    Types     []string   `json:"types,omitempty"`
    TimeRange *TimeRange `json:"time_range,omitempty"`
    Limit     int        `json:"limit"`
}

type TimeRange struct {
    From string `json:"from"` // ISO8601
    To   string `json:"to"`   // ISO8601
}
```

---

## Acceptance Criteria

- [ ] `EngineService("episodic")` returns `"graphiti-ingestion"`
- [ ] `EngineService("semantic")` returns `"cognee-ingestion"`
- [ ] `ValidMemoryType("invalid")` returns false
- [ ] `ValidMemoryType("auto")` returns true
- [ ] `go build ./gateway/domain/...` passes

## Files tạo ra / sửa

```
gateway/domain/routing.go   [NEW]
gateway/domain/entity.go    [MODIFY — add Store/Recall types]
```
