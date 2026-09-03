# Change Request: CR-COGNEE-001 — Memify (Graph Enrichment)

**CR ID:** CR-COGNEE-001  
**Component:** `services/kg-service` | `gateway`  
**Priority:** High  
**Status:** Implemented  
**Reference:** Cognee PRD §4.1.5, SRS FR-PROC-02, URD UR-PROC-03  
**Spec:** `references/cognee/specs/services/03-cognee-cognify.md`

---

## 1. Mô tả

Bổ sung tính năng **Memify** (Graph Enrichment) cho phép làm giàu knowledge graph hiện tại một cách **non-destructive**: suy luận ra các derived facts, infer relationships mới, extract thêm triplets, và cập nhật triplet embeddings — **mà không cần chạy lại pipeline Cognify từ đầu**.

---

## 2. Vấn đề hiện tại

- Trong `services/cognee-cognify`, toàn bộ flow `StartCognify` (classify → chunk → extract → ontology → embed → community detect) là **destructive**: mỗi lần cognify, graph được rebuild lại.
- Khi dataset đã có knowledge graph với hàng nghìn nodes/edges, việc chạy lại cognify rất tốn kém (LLM tokens + thời gian xử lý).
- Không có cơ chế để **bổ sung thêm facts** hoặc **cải thiện triplet index** mà không phá vỡ graph hiện có.

---

## 3. Thay đổi đề xuất

### 3.1. Service: `services/cognee-cognify`

**[NEW]** `internal/usecase/memify.go`

```go
// MemifyUseCase — non-destructive graph enrichment
type MemifyUseCase struct {
    graphRepo    port.GraphRepository     // Read existing nodes/edges
    vectorRepo   port.VectorRepository    // Update triplet embeddings
    llmClient    port.LLMClient           // Structured fact derivation
    embedder     port.EmbedderClient      // Re-embed triplets
    eventPub     port.EventPublisher
}

// Pipeline steps:
// 1. GetExistingGraph(dataset_id, tenant_id) — load nodes + edges từ Neo4j
// 2. DeriveFacts(nodes, edges) — LLM suy luận derived relationships
// 3. BuildEnrichmentDiff(existingGraph, derivedFacts) — chỉ lấy facts mới
// 4. UpsertNodes/Edges(diff) — thêm vào Neo4j, không xóa cũ
// 5. EmbedTriplets(diff) — create/update Qdrant triplet vectors
// 6. Publish cognee.cognify.memify.completed
```

**[MODIFY]** `internal/adapter/grpc/handler.go`

Thêm gRPC method:
```protobuf
// api/proto/cognee/cognify/v1/cognify.proto
rpc Memify(MemifyRequest) returns (MemifyResponse);

message MemifyRequest {
  string dataset_id = 1;
  string tenant_id  = 2;
  optional MemifyConfig config = 3;
}

message MemifyConfig {
  bool   derive_facts   = 1;  // default: true
  bool   embed_triplets = 2;  // default: true
  int32  batch_size     = 3;  // default: 50
}

message MemifyResponse {
  string pipeline_run_id  = 1;
  string status           = 2; // QUEUED | RUNNING | COMPLETED | FAILED
}
```

**[MODIFY]** `internal/adapter/event/publisher.go`

Thêm NATS subjects:
```go
cognee.cognify.memify.completed  // payload: {dataset_id, new_nodes, new_edges, tenant_id}
cognee.cognify.memify.failed     // payload: {pipeline_run_id, error}
```

### 3.2. Service: `services/vnp-gateway`

**[MODIFY]** `internal/adapter/http/cognee_routes.go`

Thêm 2 REST routes:
```
POST /api/v1/cognee/datasets/{id}/memify
GET  /api/v1/cognee/datasets/{id}/memify/status
```

Mapping:
```
POST /api/v1/cognee/datasets/{id}/memify → cognee-cognify:9012 gRPC Memify()
GET  /api/v1/cognee/datasets/{id}/memify/status → cognee-cognify:9012 gRPC GetPipelineStatus()
```

---

## 4. Traceability

| Item | Ref |
|---|---|
| gRPC port | `cognee-cognify:9012` |
| New gRPC method | `CognifyService.Memify` |
| New REST routes | `POST /api/v1/cognee/datasets/{id}/memify` |
| NATS published | `cognee.cognify.memify.completed` |
| NATS subscribed | (none, fire-and-forget) |
| Graph DB | Neo4j — upsert only, no delete |
| Vector DB | Qdrant — update triplet collection |

---

## 5. Acceptance Criteria

- [x] `POST /api/v1/cognee/datasets/{id}/memify` trả về `202 Accepted` với `pipeline_run_id`.
- [x] Job memify chạy background, **không xóa** nodes/edges cũ trên Neo4j.
- [x] Sau memify, số lượng edges trong dataset tăng thêm (derived facts).
- [x] Triplet embeddings được cập nhật trong Qdrant collection `cognee_triplets_{tenant_id}`.
- [x] Event `cognee.cognify.memify.completed` được publish lên NATS JetStream.
- [x] `GET /api/v1/cognee/datasets/{id}/memify/status` trả về trạng thái chính xác.

---

## 6. Implementation Notes

**Implemented in:** `services/kg-service` (MERGE-P2-T2 — merged service, không tách riêng)

| File | Change |
|------|--------|
| `services/kg-service/internal/domain/cognee/entity.go` | `[NEW]` `MemifyJob`, `MemifyConfig` entities |
| `services/kg-service/internal/adapter/cognee/noop.go` | `[MODIFY]` `Interface` + `NoopClient` thêm `Memify()`, `GetMemifyStatus()` |
| `services/kg-service/internal/adapter/cognee/client.go` | `[MODIFY]` `HTTPClient` thêm `Memify()`, `GetMemifyStatus()` → calls Cognee Python `/api/v1/datasets/{id}/memify` |
| `services/kg-service/internal/usecase/cognee/service.go` | `[NEW]` `MemifyUseCase` với `Memify()` + `GetMemifyStatus()` |
| `services/kg-service/internal/adapter/grpc/router.go` | `[MODIFY]` `KGHandler` thêm `Memify`, `GetMemifyStatus` handlers + routes |
| `services/kg-service/cmd/server/main.go` | `[MODIFY]` Wire up `MemifyUseCase` |
| `gateway/adapter/handler/services.go` | `[MODIFY]` `CogneeHandler` thêm `Memify()`, `GetMemifyStatus()` |
| `gateway/adapter/handler/router.go` | `[MODIFY]` Routes `POST/GET /v1/cognee/datasets/{id}/memify[/status]` |
