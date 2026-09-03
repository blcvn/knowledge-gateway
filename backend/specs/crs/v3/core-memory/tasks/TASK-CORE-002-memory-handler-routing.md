# TASK-CORE-002 — Memory Handler: Store & Recall Routing

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-002 |
| **Wave** | 1 |
| **Solution** | [SOL-CORE-001](../solutions/SOL-CORE-001-Unified-Memory-Router.md) §2.1 |
| **Component** | `gateway/adapter/handler/memory_handler.go` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CORE-001, TASK-CORE-003 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** MemoryHandler + RouteUseCase (route.go): Store → classify → route to engine; Recall → cross-engine search
---

## Mục tiêu

Implement `POST /v1/memory/store` với type-based routing và async dispatch.

---

## Công việc cụ thể

### 1. `gateway/adapter/handler/memory_handler.go` [MODIFY]

Thêm `Store` method:

```go
func (h *MemoryHandler) Store(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())

    var req domain.StoreRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }

    if req.Content == "" {
        writeError(w, http.StatusBadRequest, "missing_content", "content is required")
        return
    }

    if !domain.ValidMemoryType(req.Type) {
        writeError(w, http.StatusBadRequest, "invalid_type",
            "type must be: episodic|semantic|conversational|profile|procedural|adaptive|auto")
        return
    }

    // Auto-classify
    if req.Type == domain.MemoryTypeAuto {
        classified, err := h.classifier.Classify(r.Context(), req.Content)
        if err != nil {
            classified = domain.MemoryTypeSemantic // safe default
        }
        req.Type = classified
    }

    // Lookup engine
    svcName := domain.EngineService(req.Type)
    conn, err := h.registry.Get(svcName)
    if err != nil {
        writeError(w, http.StatusServiceUnavailable, "engine_unavailable",
            fmt.Sprintf("engine %s not available", svcName))
        return
    }

    // Generate memory ID
    memID := uuid.NewString()

    // Async dispatch to engine
    go func() {
        ctx, cancel := context.Background(), func() {}
        ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
        defer cancel()

        // Use shared ingest proto
        client := ingestpb.NewIngestServiceClient(conn)
        _, err := client.Ingest(ctx, &ingestpb.IngestRequest{
            MemoryId: memID,
            TenantId: tenantID,
            UserId:   req.UserID,
            Content:  req.Content,
            Type:     req.Type,
            Metadata: marshalMetadata(req.Metadata),
        })
        if err != nil {
            h.logger.Error("memory store failed",
                "engine", svcName, "memory_id", memID, "error", err)
        }
        // Publish NATS event
        h.pub.Publish(context.Background(), "memory.blob.inserted", map[string]string{
            "memory_id": memID, "tenant_id": tenantID, "type": req.Type,
        })
    }()

    // Immediate 202 response
    writeJSON(w, http.StatusAccepted, domain.StoreResponse{
        ID:     memID,
        Type:   req.Type,
        Engine: domain.EngineName(req.Type),
        Status: "processing",
    })
}
```

### 2. Route registration: `gateway/adapter/handler/router.go` [MODIFY]

```go
r.Post("/v1/memory/store", memoryHandler.Store)
r.Post("/v1/memory/recall", memoryHandler.Recall)
r.Post("/v1/memory/forget", memoryHandler.Forget)
r.Get("/v1/memory/timeline", memoryHandler.Timeline)
```

---

## Acceptance Criteria

- [ ] `POST /v1/memory/store` với type episodic → dispatches to graphiti-ingestion
- [ ] Response: `202 Accepted` trong < 50ms
- [ ] Unknown type → `400 Bad Request`
- [ ] Engine unavailable → `503 Service Unavailable`
- [ ] NATS event `memory.blob.inserted` published after engine success
- [ ] TenantID từ JWT context được inject vào downstream call

## Files

```
gateway/adapter/handler/memory_handler.go  [MODIFY — add Store]
gateway/adapter/handler/router.go          [MODIFY — register routes]
```
