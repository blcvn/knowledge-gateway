# SOL-CORE-001 — Solution: Unified Memory Router

| Field | Value |
|---|---|
| **Solution ID** | SOL-CORE-001 |
| **CR** | [CR-CORE-001 — Unified Memory Router](../../../../docs/crs/v3/core-memory/CR-CORE-001-Unified-Memory-Router.md) |
| **TDD ref** | [01-gateway.md](../../../tdd/architecture/01-gateway.md) · [backend-api-specs.md](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |

---

## 1. Phân tích kiến trúc

Theo TDD `01-gateway.md`, `POST /v1/memory/store` hiện đã được route qua `MemoryHandler.Store` → `memory-service`. Tuy nhiên **routing logic chưa implement đầy đủ** — cần:
1. Type-based routing tới từng engine (6 types)
2. LLM classifier cho `type=auto`
3. Non-blocking response (202 trước khi engine xử lý)

**Gateway InProcessRegistry** (`gateway/infra/registry/`) đã register tất cả 42 services — chỉ cần implement routing logic.

---

## 2. Giải pháp

### 2.1 `gateway/adapter/handler/memory_handler.go` [MODIFY]

```go
package handler

// POST /v1/memory/store
func (h *MemoryHandler) Store(w http.ResponseWriter, r *http.Request) {
    tenant := tenant.FromContext(r.Context()) // from AuthMiddleware
    
    var req domain.StoreRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }
    
    // Auto-classify nếu type == "auto" (Bifrost LLM call)
    if req.Type == domain.MemoryTypeAuto {
        classified, err := h.classifier.Classify(r.Context(), req.Content)
        if err != nil {
            // Fallback: default to "semantic"
            classified = domain.MemoryTypeSemantic
        }
        req.Type = classified
    }
    
    // Lookup engine service từ InProcessRegistry
    svcName := domain.EngineService(req.Type)
    conn, err := h.registry.Get(svcName)
    if err != nil {
        writeError(w, http.StatusServiceUnavailable, "engine_unavailable", svcName)
        return
    }
    
    // Non-blocking: fan-out to engine async
    memID := uuid.NewString()
    go func() {
        ctx := context.Background()
        client := ingestpb.NewIngestServiceClient(conn)
        _, err := client.Ingest(ctx, &ingestpb.IngestRequest{
            MemoryId: memID,
            TenantId: tenant,
            UserId:   req.UserID,
            Content:  req.Content,
            Type:     string(req.Type),
            Metadata: req.Metadata,
        })
        if err != nil {
            h.logger.Error("engine ingest failed", "engine", svcName, "error", err)
        }
    }()
    
    writeJSON(w, http.StatusAccepted, domain.StoreResponse{
        ID:     memID,
        Type:   string(req.Type),
        Engine: domain.EngineName(req.Type),
        Status: "processing",
    })
}
```

### 2.2 `gateway/domain/routing.go` [NEW]

```go
package domain

// EngineService maps MemoryType to InProcessRegistry service name
func EngineService(t string) string {
    return map[string]string{
        MemoryTypeEpisodic:       "graphiti-ingestion",
        MemoryTypeSemantic:       "cognee-ingestion",
        MemoryTypeConversational: "zep-memory",
        MemoryTypeProfile:        "memobase-ingestion",
        MemoryTypeProcedural:     "ov-resource",
        MemoryTypeAdaptive:       "sm-memory",
    }[t]
}

// EngineName returns human-readable engine name
func EngineName(t string) string {
    return map[string]string{
        MemoryTypeEpisodic:       "graphiti",
        MemoryTypeSemantic:       "cognee",
        MemoryTypeConversational: "zep",
        MemoryTypeProfile:        "memobase",
        MemoryTypeProcedural:     "openviking",
        MemoryTypeAdaptive:       "supermemory",
    }[t]
}
```

### 2.3 `gateway/internal/usecase/classifier.go` [NEW]

```go
package usecase

// MemoryClassifier uses Bifrost LLM to classify content type
type MemoryClassifier struct {
    llm port.LLMClient
}

const classifyPrompt = `Phân loại nội dung sau vào một trong các loại:
- episodic: sự kiện, activities, what happened
- semantic: kiến thức, facts, concepts
- conversational: cuộc trò chuyện, messages
- profile: thông tin cá nhân, preferences
- procedural: instructions, workflows, how-to
- adaptive: personal learning, patterns over time

Nội dung: %s
Chỉ trả về một trong 6 từ trên.`

func (c *MemoryClassifier) Classify(ctx context.Context, content string) (string, error) {
    resp, err := c.llm.Complete(ctx, &port.CompletionRequest{
        Prompt:      fmt.Sprintf(classifyPrompt, content),
        MaxTokens:   10,
        Temperature: 0.0,
    })
    if err != nil {
        return "", err
    }
    t := strings.TrimSpace(strings.ToLower(resp.Content))
    valid := map[string]bool{
        "episodic": true, "semantic": true, "conversational": true,
        "profile": true, "procedural": true, "adaptive": true,
    }
    if !valid[t] {
        return domain.MemoryTypeSemantic, nil // safe default
    }
    return t, nil
}
```

### 2.4 NATS Event sau khi engine xử lý

```go
// Trong engine handler sau khi store thành công:
h.nats.Publish("memory.blob.inserted", &MemoryEvent{
    MemoryID: memID,
    TenantID: tenantID,
    Type:     memType,
    Engine:   engineName,
})
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `gateway/adapter/handler/memory_handler.go` | MODIFY | Add type-based routing + async dispatch |
| `gateway/domain/entity.go` | MODIFY | Add EngineService(), EngineName() helpers |
| `gateway/domain/routing.go` | NEW | Routing table (MemoryType → service name) |
| `gateway/internal/usecase/classifier.go` | NEW | Bifrost LLM classifier |
| `gateway/internal/port/classifier.go` | NEW | MemoryClassifier interface |
| `backend/api/proto/memory/v1/ingest.proto` | VERIFY | Ensure IngestRequest has MemoryId field |

---

## 4. Acceptance Criteria

- [ ] `POST /v1/memory/store` với mọi `type` trả về `202 Accepted` trong `< 50ms`
- [ ] `type=auto` gọi Bifrost classifier và resolve đúng type
- [ ] Cross-tenant isolation: TenantID injected vào tất cả downstream calls
- [ ] NATS event `memory.blob.inserted` published sau khi engine store xong
- [ ] Circuit breaker (shared/pkg/resilience) bao quanh mỗi engine call
- [ ] Metrics: `memory_store_total{type, engine, status}` Prometheus counter

---

## 5. Dependencies

- InProcessRegistry register đủ 6 engine services
- Bifrost LLM proxy running (`LLM_PROXY_URL` env)
- All v1 engine CRs deployed (cognee, graphiti, zep, memobase, openviking, supermemory)
