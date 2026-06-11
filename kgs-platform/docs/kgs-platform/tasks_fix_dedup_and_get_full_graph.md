# Tasks: Fix Duplication Bug & Thêm API GetFullGraph

> Tham khảo chi tiết: [fix_dedup_and_get_full_graph_plan.md](./fix_dedup_and_get_full_graph_plan.md)

---

## Phase 1: Fix Node/Edge Duplication Bug

### 1.1 Đổi Cypher CREATE → MERGE

- [x] **T1-1** Đổi `CREATE` → `MERGE` trong `writeChunk` (`internal/batch/neo4j_writer.go:65`)
  - Thêm `ON CREATE SET n += e, n.created_at = datetime()`
  - Thêm `ON MATCH SET n += e, n.updated_at = datetime()`
  - Đổi return alias `created` → `upserted`
- [x] **T1-2** Đổi `CREATE` → `MERGE` trong `CreateNode` (`internal/data/graph_node.go:50`)
  - Thêm `ON CREATE SET` / `ON MATCH SET` với timestamps
- [x] **T1-3** Đổi `CREATE` → `MERGE` trong `CreateEdge` (`internal/data/graph_edge.go:33`)
  - Thêm `ON CREATE SET` / `ON MATCH SET` với timestamps
- [x] **T1-4** Verify compile — `go build ./...` trong kgs-platform
  - Kết quả: `go build ./...` pass

### 1.2 Thêm UNIQUE Constraint

- [x] **T1-5** Xác định Neo4j edition đang dùng (Community vs Enterprise) → quyết định dùng NODE KEY hay composite `_unique_key`
  - Chọn phương án tương thích rộng: composite `_unique_key` + UNIQUE constraint trên label `Entity`
- [x] **T1-6** Tạo file `internal/data/neo4j_constraints.go` — hàm `EnsureConstraints`
  - Nếu Community: thêm composite property `_unique_key = app_id/tenant_id/id` + UNIQUE constraint
  - Nếu Enterprise: dùng NODE KEY constraint trên `(app_id, tenant_id, id)`
- [x] **T1-7** Thêm label chung `Entity` cho tất cả nodes khi MERGE (cập nhật Cypher trong T1-1, T1-2)
- [x] **T1-8** Gọi `EnsureConstraints` khi khởi động trong `internal/data/data.go`
  - Triển khai theo chế độ an toàn: log warning nếu constraint fail do legacy dirty data, không crash startup

### 1.3 Cleanup dữ liệu duplicate hiện có

- [x] **T1-9** Viết script Cypher detect duplicate nodes (cùng `app_id`, `tenant_id`, `id`)
- [x] **T1-10** Viết script Cypher merge duplicate nodes — giữ node đầu, chuyển relationships, xóa phần thừa
- [x] **T1-11** Viết script Cypher xóa duplicate edges (cùng source, target, type, `id`)
  - Script đã tạo: `services/ai-kg-service/kgs-platform/docs/neo4j_dedup_cleanup.cypher`
- [ ] **T1-12** Chạy cleanup scripts trên môi trường dev/staging
- [ ] **T1-13** Verify sau cleanup: count nodes/edges đúng expected
  - Ghi chú: chưa execute trực tiếp vì task hiện tại tập trung implement code + script, chưa thao tác DB live.

### 1.4 Unit Tests — Duplication Fix

- [x] **T1-14** Test MERGE tạo node mới khi chưa tồn tại
- [x] **T1-15** Test MERGE cập nhật node khi đã tồn tại (cùng `app_id`, `tenant_id`, `id`) — không tạo duplicate
- [x] **T1-16** Test MERGE edge giữa 2 nodes không tạo duplicate relationship
- [x] **T1-17** Test `ON CREATE SET` — `created_at` được set khi tạo mới
- [x] **T1-18** Test `ON MATCH SET` — `updated_at` được set khi cập nhật
  - Đã thêm test query-level cho upsert semantics:
    - `internal/batch/neo4j_writer_test.go`
    - `internal/data/graph_node_test.go`
    - `internal/data/graph_edge_test.go`
- [ ] **T1-19** Test batch MERGE (`writeChunk`) với entities đã tồn tại trong DB — node count không đổi
- [ ] **T1-20** Test constraint violation khi cố tạo duplicate bằng raw Cypher (bypass MERGE)
  - Ghi chú: 2 test này cần integration test với Neo4j runtime thực.

### Kết quả chạy verify cho Phase 1

- [x] `go test ./internal/batch ./internal/data ./internal/biz ./internal/service ./internal/search ./internal/server/middleware`
- [x] `go build ./...`

---

## Phase 2: Thêm API GetFullGraph

### 2.1 Proto Definition

- [x] **T2-1** Thêm message `GetFullGraphRequest` vào `proto/kgs/v1/graph.proto`
  - Fields: `app_id`, `tenant_id`, `node_limit`, `node_offset`
- [x] **T2-2** Thêm message `GraphNode` (id, label, properties map)
  - Cập nhật theo hướng backward-compatible: giữ `properties_json`, bổ sung thêm `map<string,string> properties`.
- [x] **T2-3** Thêm message `GraphEdge` (id, relation_type, source_node_id, target_node_id, properties map)
  - Cập nhật theo hướng backward-compatible: giữ `source/target/type/properties_json`, bổ sung fields mới cho full graph.
- [x] **T2-4** Thêm message `GetFullGraphResponse` (repeated nodes, repeated edges, total_nodes, total_edges)
- [x] **T2-5** Thêm RPC `GetFullGraph` vào `GraphService`
- [x] **T2-6** Generate Go code từ proto (`make proto` hoặc `buf generate`)
  - Đã generate bằng `protoc` trực tiếp cho `api/graph/v1/graph.proto` (không dùng `make api` do thiếu plugin `protoc-gen-openapi` trong local env).

### 2.2 Data Layer

- [x] **T2-7** Tạo types `FullGraphResult`, `NodeResult`, `EdgeResult`
  - Triển khai trong `internal/biz/full_graph.go` để tránh vòng phụ thuộc package (`data -> biz`).
- [x] **T2-8** Implement `GetFullGraph` method trên `graphRepo`
  - Query 1: Count total nodes theo `app_id` + `tenant_id`
  - Query 2: Count total edges theo `app_id` + `tenant_id`
  - Query 2: Fetch nodes với pagination (`SKIP`/`LIMIT`, `ORDER BY n.id`)
  - Query 3: Fetch edges giữa các nodes đã fetch (dùng `WHERE a.id IN $node_ids AND b.id IN $node_ids`)
- [x] **T2-9** Thêm `GetFullGraph` vào interface `GraphRepo`
- [x] **T2-10** Thêm index trên `(app_id, tenant_id)` cho Neo4j để tối ưu query performance
  - Đã bổ sung trong `EnsureConstraints`:
    - `CREATE INDEX kgs_entity_app_tenant IF NOT EXISTS FOR (n:Entity) ON (n.app_id, n.tenant_id)`

### 2.3 Service Layer

- [x] **T2-11** Implement `GetFullGraph` gRPC handler trong `internal/service/graph.go`
  - Gọi `graphUsecase.GetFullGraph` (delegate xuống `GraphRepo.GetFullGraph`)
  - Convert `FullGraphResult` → `pb.GetFullGraphResponse`
  - Structured logging (start, success/error, duration)
- [x] **T2-12** Register RPC `GetFullGraph` trong gRPC server (`cmd/server/main.go` hoặc tương đương)
  - Không cần sửa thêm ở `cmd/server/main.go`; `graph.RegisterGraphServer(srv, g)` đã register toàn bộ methods của service sau khi regenerate proto.

### 2.4 Tests — GetFullGraph

- [x] **T2-13** Test GetFullGraph trả đúng tất cả nodes theo `app_id` + `tenant_id`
- [x] **T2-14** Test GetFullGraph trả đúng tất cả edges giữa các nodes
- [x] **T2-15** Test GetFullGraph với pagination (`node_limit` + `node_offset`)
- [x] **T2-16** Test GetFullGraph với tenant không có data → trả empty response
- [x] **T2-17** Test GetFullGraph không trả nodes/edges của tenant khác (multi-tenancy isolation)
  - Đã thêm test unit:
    - `internal/service/graph_test.go` (`TestGraphServiceGetFullGraph`, `TestGraphServiceGetFullGraphEmptyGraph`, `TestGraphServiceGetFullGraphRejectsScopeMismatch`)
    - `internal/data/graph_query_test.go` (verify query pattern count/nodes/edges)
- [ ] **T2-18** Test GetFullGraph performance với graph >1000 nodes

### 2.5 Publish Proto

- [ ] **T2-19** Publish proto mới lên registry (kratos-proto hoặc buf registry)
- [ ] **T2-20** Verify consumer có thể import proto mới

### Kết quả chạy verify cho Phase 2

- [x] `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go test ./internal/biz ./internal/data ./internal/service ./internal/search ./internal/server/middleware ./internal/batch`
- [x] `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go build ./...`

---

## Phase 3: Tích hợp ba-agent-service

### 3.1 Cập nhật Proto Dependency

- [x] **T3-1** Cập nhật proto dependency trong ba-agent-service (`go.mod` hoặc buf lock)
  - `go.mod` đã dùng `replace kgs-platform => ../ai-kg-service/kgs-platform` từ trước; giữ nguyên và sử dụng proto mới từ submodule local.
- [x] **T3-2** Generate Go client code từ proto mới
  - Đồng bộ generated files vào vendor để tương thích `-mod=vendor`:
    - `vendor/kgs-platform/api/graph/v1/graph.proto`
    - `vendor/kgs-platform/api/graph/v1/graph.pb.go`
    - `vendor/kgs-platform/api/graph/v1/graph_grpc.pb.go`
    - `vendor/kgs-platform/api/graph/v1/graph_http.pb.go`

### 3.2 Converter

- [x] **T3-3** Thêm method `FullGraphResponseToGraph` vào `DomainConverter`
  - Convert `pb.GraphNode` → domain `v32.GraphNode`
  - Convert `pb.GraphEdge` → domain `v32.GraphEdge`
  - Assemble thành `v32.Graph`
- [x] **T3-4** Viết unit tests cho `FullGraphResponseToGraph`
  - Đã bổ sung thêm các test liên quan converter fallback cho field mới của proto FullGraph.

### 3.3 Repository

- [x] **T3-5** Thay thế `getGraphByScope` trong `KGServiceRepository` — dùng `GetFullGraph` RPC thay vì HybridSearch workaround
- [x] **T3-6** Viết unit tests cho `getGraphByScope` mới (mock gRPC client)
  - Đã cập nhật `kg_repository_test.go` sang flow `GetFullGraph` và mock `GetFullGraph`.

### 3.4 End-to-End Verification

- [ ] **T3-7** Test E2E: `SaveGraph` 1 lần → `GetFullGraph` → verify node count = expected
- [ ] **T3-8** Test E2E: `SaveGraph` 2 lần (cùng data) → verify node count không đổi (dedup works)
- [ ] **T3-9** Test E2E: `SaveGraph` 2 lần (data thay đổi) → verify properties được cập nhật
- [ ] **T3-10** Test E2E: ba-agent-service gọi `GetFullGraph` qua gRPC → verify response mapping đúng

### Kết quả chạy verify cho Phase 3

- [x] `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go test -mod=vendor ./repository/kgservice`
- [x] `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go test -mod=vendor ./repository/...`
- [x] `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go build -mod=vendor ./...`

---

## Tổng kết

| Phase | Số tasks | Mô tả |
|-------|----------|-------|
| Phase 1 | 20 | Fix duplication bug (MERGE + constraint + cleanup + tests) |
| Phase 2 | 20 | Thêm API GetFullGraph (proto + data + service + tests + publish) |
| Phase 3 | 10 | Tích hợp ba-agent-service (proto dep + converter + repository + E2E) |
| **Tổng** | **50** | |

**Thứ tự:** Phase 1 → Phase 2 → Phase 3 (tuần tự, mỗi phase phụ thuộc vào phase trước)
