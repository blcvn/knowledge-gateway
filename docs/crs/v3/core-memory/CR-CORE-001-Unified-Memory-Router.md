# Change Request: CR-CORE-001 — Unified Memory Router

**CR ID:** CR-CORE-001
**Component:** `backend/gateway` — Memory routing layer
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Core Memory & Integration
**Solution:** [S2 — Unified Memory API](../../../bussiness/solutions/S2-unified-api.md)
**Features:** [F01 — Unified Memory API](../../../features/01-unified-memory-api/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-02 | AI Agent Developer | Phải tích hợp 6 APIs khác nhau cho 6 engine — 500 LOC boilerplate |
| PP-P6-01 | Framework Integrator | Không có standard API — mỗi framework phải viết adapter riêng |
| PP-P1-02 | AI Agent Developer | Memory fragmented — không biết query engine nào cho loại memory nào |

**Business Impact:** Developer mất 2-3 tuần chỉ để setup memory routing logic.
**After:** 20 LOC, 1 API call, tự động route.

---

## 2. Mô tả

Xây dựng **Unified Memory Router** trong API Gateway — layer nhận `POST /v1/memory/store` với `type` field và tự động route tới đúng engine backend.

Khi `type=auto`: Gateway gọi LLM để classify content rồi route.

---

## 3. API Contract

```http
POST /v1/memory/store
Authorization: Bearer <api-key>
Content-Type: application/json

{
  "user_id": "u_123",
  "content": "Tôi muốn học machine learning",
  "type": "auto",          // auto | episodic | semantic | conversational | profile | procedural | adaptive
  "metadata": {
    "source": "chat",
    "session_id": "s_456"
  }
}

→ HTTP 202 Accepted
{
  "id": "mem_abc123",
  "type": "profile",       // resolved type (nếu auto)
  "engine": "memobase",    // engine được chọn
  "status": "processing"
}
```

---

## 4. Routing Rules

| type | Engine | gRPC service |
|---|---|---|
| `episodic` | Graphiti | `graphiti-ingestion` |
| `semantic` | Cognee | `cognee-ingestion` |
| `conversational` | Zep | `zep-memory` |
| `profile` | Memobase | `memobase-ingestion` |
| `procedural` | OpenViking | `ov-fs` |
| `adaptive` | Supermemory | `sm-memory` |
| `auto` | LLM classify → route | runtime decision |

---

## 5. Thay đổi đề xuất

### 5.1 `backend/gateway/internal/adapter/handler/memory_handler.go`

```go
// [MODIFY] Add Store handler với type-based routing
func (h *MemoryHandler) Store(w http.ResponseWriter, r *http.Request) {
    var req StoreRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Auto-classify nếu type == "auto"
    if req.Type == MemoryTypeAuto {
        req.Type = h.classifier.Classify(r.Context(), req.Content)
    }
    
    // Route to engine via InProcessRegistry
    conn := h.registry.Lookup(engineService(req.Type))
    client := ingestpb.NewIngestServiceClient(conn)
    res, err := client.Ingest(r.Context(), toProto(req))
    
    writeJSON(w, http.StatusAccepted, StoreResponse{
        ID:     res.Id,
        Type:   string(req.Type),
        Engine: engineName(req.Type),
        Status: "processing",
    })
}
```

### 5.2 `backend/gateway/domain/routing.go` [NEW]

```go
// EngineService maps MemoryType to InProcessRegistry service name
func engineService(t MemoryType) string {
    return map[MemoryType]string{
        MemoryTypeEpisodic:       "graphiti-ingestion",
        MemoryTypeSemantic:       "cognee-ingestion",
        MemoryTypeConversational: "zep-memory",
        MemoryTypeProfile:        "memobase-ingestion",
        MemoryTypeProcedural:     "ov-fs",
        MemoryTypeAdaptive:       "sm-memory",
    }[t]
}
```

### 5.3 `backend/gateway/internal/usecase/classifier.go` [NEW]

```go
// LLM-based memory type classifier (dùng khi type=auto)
type MemoryClassifier interface {
    Classify(ctx context.Context, content string) MemoryType
}

// Prompt: "Phân loại nội dung sau vào: episodic/semantic/conversational/profile/procedural/adaptive"
// Response: one of the 6 types
```

---

## 6. Acceptance Criteria

- [ ] `POST /v1/memory/store` với mọi `type` value trả về `202 Accepted` trong `< 50ms`
- [ ] `type=auto` gọi LLM classifier và resolve đúng type
- [ ] Non-blocking: response trả về trước khi engine xử lý xong
- [ ] NATS event `memory.blob.inserted` được publish sau khi engine store thành công
- [ ] TenantID được inject vào tất cả downstream calls (zero cross-tenant leak)
- [ ] `GET /healthz` liệt kê tất cả engines available

---

## 7. Dependencies

- CR-CORE-002 (Recall cần router đã hoạt động)
- All v1 engine CRs phải deployed
- InProcessRegistry phải register tất cả engine services

---

## 8. Tham chiếu

- [Feature F01](../../../features/01-unified-memory-api/README.md)
- [Solution S2](../../../bussiness/solutions/S2-unified-api.md)
- [ADR-001](../../../adr/ADR-001-monolith-first.md) — InProcessRegistry pattern
- [ADR-002](../../../adr/ADR-002-grpc-bufconn.md) — gRPC + bufconn
