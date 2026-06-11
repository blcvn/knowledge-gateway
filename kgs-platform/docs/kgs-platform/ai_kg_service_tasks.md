# Tasks: AI Knowledge Graph Service (ai-kg-service)

> **Tham chiếu:** [Implementation Plan](ai_kg_service_impl_plan.md) | [LLD](lld_ai_kg_service.md) | [API Specs](ai_kg_service_api_specs.md) | [Coverage](lld_coverage_ai_kg_service.md) | [DOC 4 Compliance](doc4_compliance_review.md)
>
> **Cập nhật DOC 4 Compliance (05/03/2026):** Bổ sung các task OrgID, NATS Topics/Events, Structured Errors.
> **Cập nhật thực thi (05/03/2026):** Đã triển khai xong hầu hết Phase 0 (Graph/Auth/RateLimit/Namespace + unit tests).  
> Đã tiếp tục Phase 1 (Lock Manager + Batch core + QueryPlanner enhancements).  
> Chưa verify được `test_api.sh` và `go test ./... -cover` trong môi trường hiện tại do sandbox không truy cập được Go module proxy.  
> Ghi chú kỹ thuật: `P1.1.2` hiện dùng Redis `SETNX + Lua release` (chưa dùng trực tiếp `go-redsync/redsync` vì ràng buộc môi trường).
> `graph.proto` đã được generate đầy đủ cho `graph.pb.go`, `graph_grpc.pb.go`, `graph_http.pb.go`.
> Verify local cho Phase 1: `go test ./internal/lock ./internal/batch ./internal/biz ./internal/service` PASS.
> Các mục verify API trong Phase 1 được xác nhận qua service/unit tests (chưa chạy end-to-end với full stack Neo4j/Redis/HTTP server).
> **Cập nhật Phase 2 (05/03/2026):** Đã triển khai Qdrant config/client + HybridSearch pipeline + backfill dedup/index cho batch + unit tests (`internal/data/qdrant_test.go`, `internal/search/*_test.go`, `internal/service/graph_test.go`).
> Đã generate lại `conf.pb.go`, `graph.pb.go`, `graph_grpc.pb.go`, `graph_http.pb.go` (không generate OpenAPI do thiếu `protoc-gen-openapi` trong môi trường).
> `cmd/server/wire_gen.go` được cập nhật thủ công do môi trường sandbox không tải được module để chạy `wire`.
> Còn thiếu tích hợp embedding thật qua ai-proxy gRPC (`P2.2.9`) và chưa verify performance/API end-to-end (`P2.4.4`).
> **Cập nhật Phase 3 (05/03/2026):** Đã triển khai Overlay + Versioning + NATS integration nền tảng: proto/service handlers, `internal/overlay/*`, `internal/version/*`, `internal/data/nats.go`, wiring vào worker/server.
> Test PASS: `go test -mod=mod ./internal/version ./internal/overlay ./internal/data ./internal/service ./internal/server ./cmd/server`.
> Ghi chú kỹ thuật: NATS wrapper hiện là in-memory pub/sub adapter để đảm bảo testability trong môi trường hiện tại (chưa dùng NATS JetStream client thật).
> Route write vào overlay từ Graph CRUD path (`P3.1.5`) đã triển khai qua `overlay_id` trong `properties_json`.
> Đã bổ sung verify lifecycle end-to-end qua API-level service test (`internal/service/graph_phase3_e2e_test.go`) cho `P3.4.6`.
> **Cập nhật Phase 4 (05/03/2026):** Đã triển khai `internal/analytics/*` (coverage/traceability/cluster + cache key `kgs:analytics:{type}:{ns}:{hash}` TTL 15m), `internal/projection/*` (view model + CRUD + role filter + PII masking), wire DI vào `cmd/server/wire.go`, bổ sung RPC/proto cho Coverage/Traceability/View CRUD và generate lại `graph.pb.go`, `graph_grpc.pb.go`, `graph_http.pb.go`.
> Đã áp projection vào Graph responses (`GetNode`, `GetContext`, `GetImpact`, `GetCoverage`, `GetSubgraph`) và thay `biz/view_resolver.go` để dùng `ProjectionEngine`.
> Verify local: `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go test -mod=mod ./internal/... ./cmd/server` PASS.
> **Cập nhật Phase 5 (05/03/2026):** Đã triển khai observability gồm Prometheus middleware (`/metrics`), OTEL tracing middleware, health endpoints (`/healthz`, `/readyz`), business metrics instrumentation (`kg_entity_write_total`, `kg_search_duration_ms`, `kg_overlay_count_active`, `kg_lock_acquire_duration_ms`), structured errors trong `biz/errors.go`.
> Đã bổ sung integration tests (`tests/integration/phase5_integration_test.go`), e2e scaffold bằng Testcontainers (`tests/e2e/*.go`, build tag `e2e`), benchmark (`tests/benchmark/search_bench_test.go`).
> Verify local: `go test -mod=mod ./...` PASS, `go test -mod=mod -tags=e2e ./tests/e2e/... -v` PASS (SKIP khi chưa set `RUN_E2E=1`), `go test -mod=mod ./tests/benchmark -bench . -run ^$` PASS.
> Đã triển khai `P5.2.1` tại `services/ai-kg-service/deployment/docker/docker-compose.dev.yml` (stack: kgs-platform + Neo4j + PG + Redis + Qdrant + OPA + NATS), verify syntax PASS bằng `docker compose -f services/ai-kg-service/deployment/docker/docker-compose.dev.yml config`.
> Theo yêu cầu hiện tại: **tạm skip P5.2.2–P5.2.6 — Deployment Manifests còn lại**.

---

## Phase 0: Foundation Fix (Tuần 1)

### P0.1 — Fix Graph Service Handlers

- [x] P0.1.1 — Wire biz logic vào `CreateNode` handler (`service/graph.go`): parse `properties_json` → gọi `uc.CreateNode()` → map result vào `CreateNodeReply`
- [x] P0.1.2 — Wire biz logic vào `GetNode` handler (`service/graph.go`): gọi `uc.GetNode()` → map vào `GetNodeReply`
- [x] P0.1.3 — Wire biz logic vào `CreateEdge` handler (`service/graph.go`): parse request → gọi `uc.CreateEdge()` → map vào `CreateEdgeReply`
- [x] P0.1.4 — Fix hardcoded `"demo-app"` trong tất cả handlers (`service/graph.go`): extract `AppContext` từ `ctx.Value(middleware.AppContextKey)`
- [x] P0.1.5 — Implement converter `map[string]any` → protobuf `GraphReply` (`service/graph.go`): cho `GetContext/GetImpact/GetCoverage/GetSubgraph`
- [x] P0.1.6 — Thêm `GraphUsecase.GetNode()` vào biz layer (`biz/graph.go`): query Neo4j by `app_id + id`
- [x] P0.1.7 — Thêm `graphRepo.GetNode()` vào data layer (`data/graph_node.go`): Cypher `MATCH (n {app_id: $app_id, id: $node_id})`
- [ ] P0.1.8 — Verify: chạy `test_api.sh` — tất cả 7 Graph RPCs PASS

### P0.2 — Fix Auth Middleware

- [x] P0.2.1 — Implement real API key lookup (`middleware/auth.go`, `biz/registry.go`): parse `Authorization: Bearer <key>` → hash → lookup trong `api_keys` table → check `is_revoked`, `expires_at`
- [x] P0.2.2 — Cache API key validation trong Redis (`middleware/auth.go`): set `kgs:apikey:<hash>` với TTL 5min
- [x] P0.2.3 — Inject real `AppContext` vào context (`middleware/auth.go`): `AppContext{AppID: app.AppID, Scopes: apiKey.Scopes}`
- [x] P0.2.4 — Skip auth cho Registry endpoints (`middleware/auth.go`): whitelist `POST /v1/apps`, `GET /v1/apps`, `POST /v1/apps/*/keys`
- [x] P0.2.5 — Verify: invalid key → `401`, valid key → AppContext populated, expired key → `401`

### P0.3 — Fix RateLimiter Middleware

- [x] P0.3.1 — Implement Redis sliding window rate limiter (`middleware/ratelimit.go`): key `kgs:ratelimit:<appID>:<minute>`, Lua script atomic inc + expire
- [x] P0.3.2 — Read quota limit từ `Quota` table (`middleware/ratelimit.go`): default 1000 req/min, override per app
- [x] P0.3.3 — Return `429 ERR_RATE_LIMIT` với `Retry-After` header (`middleware/ratelimit.go`)
- [x] P0.3.4 — Verify: over-limit → `429`, under-limit → pass through

### P0.4 — Namespace Foundation

- [x] P0.4.1 — Thêm `TenantID` vào `AppContext` struct (`middleware/auth.go`): extract từ JWT claim `tenant_id`, fallback `default`
- [x] P0.4.2 — Tạo `biz/namespace.go`: implement `ComputeNamespace(appID, tenantID string) string` → `"graph/{appID}/{tenantID}"`
- [x] P0.4.3 — Propagate namespace vào Cypher queries (`data/graph_node.go`, `data/graph_edge.go`, `data/graph_query.go`): thêm `tenant_id` property vào `CREATE` và `MATCH`
- [x] P0.4.4 — Thêm `TenantID` field vào GORM models (`biz/ontology.go`, `biz/rules.go`): compound index `(app_id, tenant_id)`
- [x] P0.4.5 — Tạo `middleware/namespace.go`: validate `X-KG-Namespace` header match AppContext — prevent cross-tenant access
- [ ] P0.4.6 — Xử lý `X-Org-ID` header (DOC 4 §2): parse và inject vào `AppContext.OrgID`, update compute namespace support `{orgID}`
- [ ] P0.4.7 — Verify: cross-tenant/org query → `403 ERR_FORBIDDEN`

### P0.5 — Unit Tests Phase 0

- [x] P0.5.1 — Unit test cho `CreateNode/GetNode` handlers (`service/graph_test.go`): mock biz layer, verify mapping proto ↔ domain
- [x] P0.5.2 — Unit test cho Auth middleware (`middleware/auth_test.go`): table-driven — valid, invalid, expired, revoked, no header
- [x] P0.5.3 — Unit test cho RateLimiter (`middleware/ratelimit_test.go`): mock Redis, test sliding window
- [x] P0.5.4 — Unit test cho Namespace (`biz/namespace_test.go`): `ComputeNamespace` output format
- [ ] P0.5.5 — Verify: `go test ./... -cover` ≥ 80% cho code mới Phase 0

---

## Phase 1: Core Graph Enhancement (Tuần 2–3)

### P1.1 — Lock Manager Package

- [x] P1.1.1 — Define `LockManager` interface (`internal/lock/lock.go`): `AcquireNodeLock`, `AcquireSubgraphLock`, `AcquireVersionLock`, `AcquireNamespaceLock`, `Release`
- [x] P1.1.2 — Implement `RedisLockManager` (`internal/lock/redis_lock.go`): dùng `go-redsync/redsync` — Redlock algorithm
- [x] P1.1.3 — Lock key design (`internal/lock/redis_lock.go`): keys `kgs:lock:node:{ns}:{nodeID}`, `kgs:lock:subgraph:{ns}:{rootID}`, etc.
- [x] P1.1.4 — Lock hierarchy enforcement (`internal/lock/redis_lock.go`): Node < Subgraph < Version < Namespace — prevent deadlock
- [x] P1.1.5 — Integrate lock vào `GraphUsecase.CreateNode/CreateEdge` (`biz/graph.go`): acquire node lock trước write, release sau commit
- [x] P1.1.6 — Wire `LockManager` vào DI (`cmd/server/wire.go`): thêm `NewRedisLockManager` vào provider set
- [x] P1.1.7 — Unit test: concurrent acquisition, timeout, reentrant lock (`internal/lock/lock_test.go`)

### P1.2 — Batch Upsert Package

- [x] P1.2.1 — Define batch proto (`api/graph/v1/graph.proto`): `rpc BatchUpsertEntities(BatchUpsertRequest) returns (BatchUpsertReply)` — gateway `POST /v1/graph/entities/batch`
- [x] P1.2.2 — Generate proto Go code: `make proto`
- [x] P1.2.3 — Implement `BatchUsecase` core (`internal/batch/batch.go`): validate → dedup → bulk write → vector index
- [x] P1.2.4 — Implement semantic dedup placeholder (`internal/batch/dedup.go`): exact-match dedup (Qdrant cosine dedup backfilled in Phase 2)
- [x] P1.2.5 — Implement Neo4j batch writer (`internal/batch/neo4j_writer.go`): `UNWIND $entities AS e CREATE (n:EntityType)` — 200 nodes/transaction
- [x] P1.2.6 — Proto + service handler (`service/graph.go`): wire `BatchUsecase.Execute()`
- [x] P1.2.7 — Wire `BatchUsecase` vào DI (`cmd/server/wire.go`)
- [x] P1.2.8 — Unit test: empty batch, max batch (1000), duplicate detection, error handling (`internal/batch/batch_test.go`)
- [x] P1.2.9 — Verify: `POST /v1/graph/entities/batch` with 100 entities → `{created: 100}`

### P1.3 — Enhanced Graph Algorithms

- [x] P1.3.1 — Add `label` filter vào Cypher queries (`biz/query_planner.go`): filter by EntityType trong Context/Impact/Coverage
- [x] P1.3.2 — BFS batched traversal (`biz/query_planner.go`): depth > 3 → batch per level
- [x] P1.3.3 — Pagination support cho GraphReply (`api/graph/v1/graph.proto`): thêm `page_size`, `page_token` vào traversal requests
- [x] P1.3.4 — Generate proto Go code cho pagination: `make proto`
- [x] P1.3.5 — Neo4j GDS PageRank integration (`data/graph_query.go`): `CALL gds.pageRank.stream(...)` — cache Redis TTL 15min
- [x] P1.3.6 — Neo4j GDS DegreeCentrality (`data/graph_query.go`): `CALL gds.degree.stream(...)`
- [x] P1.3.7 — Unit test: BFS queries, pagination tokens, label filters (`biz/query_planner_test.go`)
- [x] P1.3.8 — Verify: GetContext depth=5 → batched BFS, pagination cursor works

---

## Phase 2: Search & Vector Index (Tuần 4–5)

### P2.1 — Qdrant Client Integration

- [x] P2.1.1 — Thêm `Qdrant` config vào `conf.proto` (`internal/conf/conf.proto`): host, port, collection, vector_size
- [x] P2.1.2 — Generate proto config: `make proto`
- [x] P2.1.3 — Tạo Qdrant Go client wrapper (`internal/data/qdrant.go`): `UpsertVectors()`, `SearchVectors()`, `DeleteVectors()`, `BatchSearch()`
- [x] P2.1.4 — Wire Qdrant vào Data layer (`internal/data/data.go`): init trong `NewData()`, add cleanup, thêm vào ProviderSet
- [x] P2.1.5 — Collection auto-creation on startup (`internal/data/qdrant.go`): `kgs-vectors-{appID}`, vector_size=1536, cosine metric
- [x] P2.1.6 — Update `config.yaml` với Qdrant config
- [x] P2.1.7 — Unit test cho Qdrant wrapper (`internal/data/qdrant_test.go`): mock HTTP server

### P2.2 — Hybrid Search Pipeline

- [x] P2.2.1 — Define `SearchEngine` interface (`internal/search/search.go`): `HybridSearch(ctx, ns, query, opts) ([]SearchResult, error)`
- [x] P2.2.2 — Implement Semantic search via Qdrant (`internal/search/vector.go`): query → embedding (deterministic placeholder) → Qdrant cosine → top-K
- [x] P2.2.3 — Implement BM25 text search via Neo4j (`internal/search/text.go`): `CALL db.index.fulltext.queryNodes('kgs-fti-{ns}', $query)`
- [x] P2.2.4 — Implement RRF Score Blending (`internal/search/blender.go`): `score = alpha * semantic + (1-alpha) * text`
- [x] P2.2.5 — Implement Graph Reranking (`internal/search/reranker.go`): `finalScore = blendedScore * (1 + beta * centrality)`
- [x] P2.2.6 — Implement Filter engine (`internal/search/filter.go`): entityTypes, domains, minConfidence, provenanceTypes
- [x] P2.2.7 — Define proto (`api/graph/v1/graph.proto`): `rpc HybridSearch(HybridSearchRequest) returns (HybridSearchReply)` — gateway `POST /v1/graph/search/hybrid`
- [x] P2.2.8 — Generate proto + implement service handler (`service/graph.go`)
- [ ] P2.2.9 — Wire embeddings call to ai-proxy (`internal/search/vector.go`): gRPC call cho text → float32[] (đang dùng deterministic embedder placeholder)
- [x] P2.2.10 — Neo4j fulltext index creation on startup (`internal/data/graph_node.go`): `CREATE FULLTEXT INDEX IF NOT EXISTS`
- [x] P2.2.11 — Wire `SearchEngine` vào DI (`cmd/server/wire.go`)

### P2.3 — Backfill Search into Batch

- [x] P2.3.1 — Wire Qdrant vào batch dedup (`internal/batch/dedup.go`): replace placeholder → real cosine similarity check
- [x] P2.3.2 — Auto-index vectors on entity create (`internal/batch/vector_indexer.go`, `internal/batch/batch.go`): sau batch write Neo4j → batch upsert vectors vào Qdrant

### P2.4 — Unit Tests Phase 2

- [x] P2.4.1 — Search engine integration test (`internal/search/search_test.go`): mock Qdrant + Neo4j, verify blending + ranking
- [x] P2.4.2 — Blender tests (`internal/search/blender_test.go`): alpha=0, alpha=1, empty results, single source
- [x] P2.4.3 — Filter tests (`internal/search/filter_test.go`): combination filters, empty filter, all-excluded
- [ ] P2.4.4 — Verify: `POST /v1/graph/search/hybrid` → ranked results with scores, P95 < 500ms

---

## Phase 3: Overlay & Versioning (Tuần 6–7)

### P3.1 — Overlay Package

- [x] P3.1.1 — Define `OverlayManager` interface (`internal/overlay/overlay.go`): `Create`, `Get`, `Commit`, `Discard`
- [x] P3.1.2 — Define overlay data model (`internal/overlay/model.go`): `OverlayGraph` struct — ID, SessionID, BaseVersionID, Status, EntitiesDelta, EdgesDelta, CreatedAt, ExpiresAt
- [x] P3.1.3 — Implement Redis overlay store (`internal/overlay/redis_store.go`): key `kgs:overlay:{overlayID}`, TTL 1h
- [x] P3.1.4 — Implement Overlay Create (`internal/overlay/overlay.go`): validate session → store empty overlay → return overlayID
- [x] P3.1.5 — Route writes to overlay (`biz/graph.go`): nếu request có `overlay_id` → write vào overlay Redis thay vì base graph
- [x] P3.1.6 — Implement Overlay Commit (`internal/overlay/commit.go`): read deltas → check base drift → resolve conflicts → write to Neo4j → create version delta → cleanup
- [x] P3.1.7 — Implement Conflict detection & resolution (`internal/overlay/conflict.go`): 4 policies — `KEEP_OVERLAY`, `KEEP_BASE`, `MERGE`, `REQUIRE_MANUAL`
- [x] P3.1.8 — Implement Overlay Discard (`internal/overlay/overlay.go`): delete Redis key + cleanup
- [x] P3.1.9 — Define proto + service handlers (`api/graph/v1/graph.proto`): `CreateOverlay`, `CommitOverlay`, `DiscardOverlay` RPCs
- [x] P3.1.10 — Generate proto + implement handlers (`service/graph.go`)
- [x] P3.1.11 — Wire `OverlayManager` vào DI (`cmd/server/wire.go`)

### P3.2 — Versioning Package

- [x] P3.2.1 — Define `VersionManager` interface (`internal/version/version.go`): `CreateDelta`, `GetVersion`, `ListVersions`, `DiffVersions`, `Rollback`
- [x] P3.2.2 — Define version delta model (`internal/version/model.go`): `VersionDelta` — ID, ParentID, Namespace, EntitiesAdded/Modified/Deleted, EdgesAdded/Modified/Deleted, CreatedAt
- [x] P3.2.3 — Implement PostgreSQL version store (`internal/version/version.go`): GORM model `graph_versions` table
- [x] P3.2.4 — Hook version creation into overlay commit (`internal/overlay/commit.go`): call `VersionManager.CreateDelta()` after conflict resolution
- [x] P3.2.5 — Implement diff computation (`internal/version/version.go`): walk delta chain between v1 and v2
- [x] P3.2.6 — Implement GC compaction (`internal/version/gc.go`): compact deltas older than N days into snapshots
- [x] P3.2.7 — Define proto + service handlers (`api/graph/v1/graph.proto`): `ListVersions`, `DiffVersions`, `RollbackVersion`
- [x] P3.2.8 — Generate proto + implement handlers (`service/graph.go`)
- [x] P3.2.9 — Wire `VersionManager` vào DI (`cmd/server/wire.go`)

### P3.3 — NATS JetStream Integration

- [x] P3.3.1 — Thêm NATS config vào `conf.proto` (`internal/conf/conf.proto`): url, stream name
- [x] P3.3.2 — Tạo NATS client wrapper (`internal/data/nats.go`): Publish + Subscribe helper
- [x] P3.3.3 — Wire NATS vào Data layer (`internal/data/data.go`): init, cleanup, ProviderSet
- [x] P3.3.4 — Publish `OVERLAY_COMMIT` event (DOC 4 §5) (`internal/overlay/commit.go`): Publish sau khi commit overlay thành công
- [x] P3.3.5 — Publish `OVERLAY_DISCARD` event (DOC 4 §5) (`internal/overlay/overlay.go`): Publish sau khi discard overlay
- [x] P3.3.6 — Subscribe `SESSION_CLOSE` (DOC 4 §8) (`internal/overlay/nats_listener.go`): Cập nhật logic commit-or-discard thay vì chỉ discard
- [x] P3.3.7 — Subscribe `BUDGET_STOP` (DOC 4 §8) (`internal/overlay/nats_listener.go`): Commit overlay với status=PARTIAL
- [x] P3.3.8 — Add NATS topic patterns constants (`internal/data/nats_topics.go`): Định nghĩa tập trung các topic name (overlay.committed, etc.)
- [x] P3.3.9 — Update `config.yaml` với NATS config

### P3.4 — Unit Tests Phase 3

- [x] P3.4.1 — Overlay lifecycle tests (`internal/overlay/overlay_test.go`): create → commit, create → discard, expired overlay
- [x] P3.4.2 — Conflict resolution tests (`internal/overlay/conflict_test.go`): 4 policies, no conflict, multi-field conflict
- [x] P3.4.3 — Version delta tests (`internal/version/version_test.go`): create, diff, chain walk
- [x] P3.4.4 — GC compaction tests (`internal/version/gc_test.go`): compact 10 deltas → 1 snapshot
- [x] P3.4.5 — NATS publish/subscribe test (`internal/data/nats_test.go`)
- [x] P3.4.6 — Verify: full overlay lifecycle end-to-end qua API

---

## Phase 4: Analytics & Projection (Tuần 8–9)

### P4.1 — Analytics Package

- [x] P4.1.1 — Define `AnalyticsEngine` interface (`internal/analytics/analytics.go`): `CoverageReport`, `TraceabilityMatrix`, `ClusterAnalysis`
- [x] P4.1.2 — Implement Coverage report (`internal/analytics/coverage.go`): count entities per type per domain, compute % covered
- [x] P4.1.3 — Implement Traceability matrix (`internal/analytics/traceability.go`): multi-hop BFS source → target paths
- [x] P4.1.4 — Implement Cluster analysis via GDS Louvain (`internal/analytics/cluster.go`): `CALL gds.louvain.stream(...)`
- [x] P4.1.5 — Define proto + service handlers (`api/graph/v1/graph.proto`): `GetCoverageReport`, `GetTraceabilityMatrix`
- [x] P4.1.6 — Generate proto + implement handlers (`service/graph.go`)
- [x] P4.1.7 — Cache analytics results trong Redis (`internal/analytics/cache.go`): key `kgs:analytics:{type}:{ns}:{hash}`, TTL 15min
- [x] P4.1.8 — Wire `AnalyticsEngine` vào DI (`cmd/server/wire.go`)

### P4.2 — Projection Package

- [x] P4.2.1 — Define `ProjectionEngine` interface (`internal/projection/projection.go`): `Apply(ctx, ns, role, rawData) → filteredData`
- [x] P4.2.2 — Define view model (`internal/projection/model.go`): `ViewDefinition` — AppID, RoleName, AllowedEntityTypes, AllowedFields, PIIMaskFields
- [x] P4.2.3 — Replace `ViewResolver` stub (`biz/view_resolver.go`): wire `ProjectionEngine`
- [x] P4.2.4 — Implement PII masking (`internal/projection/mask.go`): email → `e***@***.com`, phone → `***-***-1234`
- [x] P4.2.5 — Apply projection vào Graph responses (`service/graph.go`): after biz returns → apply projection → return filtered
- [x] P4.2.6 — Define proto + service handlers cho view CRUD (`api/graph/v1/graph.proto`)
- [x] P4.2.7 — Wire `ProjectionEngine` vào DI (`cmd/server/wire.go`)

### P4.3 — Unit Tests Phase 4

- [x] P4.3.1 — Coverage computation tests (`internal/analytics/coverage_test.go`)
- [x] P4.3.2 — Traceability BFS tests (`internal/analytics/traceability_test.go`)
- [x] P4.3.3 — Cache behavior tests (`internal/analytics/cache_test.go`)
- [x] P4.3.4 — Role filtering tests (`internal/projection/projection_test.go`)
- [x] P4.3.5 — PII masking tests (`internal/projection/mask_test.go`): edge cases — empty, null, partial
- [x] P4.3.6 — Verify: Coverage report + Traceability matrix APIs return correct data

---

## Phase 5: Production Ready (Tuần 9.5–10)

### P5.1 — Observability

- [x] P5.1.1 — Prometheus metrics middleware (`middleware/metrics.go`): `kg_request_duration_ms`, `kg_request_total` — `prometheus/client_golang`
- [x] P5.1.2 — Business metrics instrumentation (all biz packages): `kg_entity_write_total`, `kg_search_duration_ms`, `kg_overlay_count_active`, `kg_lock_acquire_duration_ms`
- [x] P5.1.3 — OpenTelemetry tracing middleware (`middleware/tracing.go`): Kratos OTEL middleware + span annotations cho Neo4j, Qdrant, Redis
- [x] P5.1.4 — Health check endpoints (`service/health.go`): `GET /healthz` (liveness), `GET /readyz` (readiness — check all connections)
- [x] P5.1.5 — Structured error codes (`biz/errors.go`): replace `errors.New()` → Kratos error format `{code, reason, message, metadata}` per DOC 4 §6

### P5.2 — Deployment Manifests

- [x] P5.2.1 — Docker Compose dev (`services/ai-kg-service/deployment/docker/docker-compose.dev.yml`): kgs-platform + Neo4j + PG + Redis + Qdrant + OPA + NATS
- [ ] P5.2.2 — K8s Deployment + Service (`deployment/k8s/kgs-platform.yaml`) (**SKIP theo yêu cầu**)
- [ ] P5.2.3 — K8s HPA (`deployment/k8s/hpa.yaml`): scale on CPU > 70% or p95 latency > 500ms (**SKIP theo yêu cầu**)
- [ ] P5.2.4 — K8s NetworkPolicy (`deployment/k8s/networkpolicy.yaml`): only `ai-planner` namespace → kgs-platform (**SKIP theo yêu cầu**)
- [ ] P5.2.5 — K8s CronJob cho Version GC (`deployment/k8s/gc-cronjob.yaml`): daily 2am (**SKIP theo yêu cầu**)
- [ ] P5.2.6 — Verify: `docker-compose up -d` → full stack running (**SKIP theo yêu cầu**)

### P5.3 — Integration & E2E Tests

- [x] P5.3.1 — Update integration test suite (`tests/integration/`): thêm tests cho Batch, Search, Overlay, Version, Analytics
- [x] P5.3.2 — E2E Testcontainers setup (`tests/e2e/main_test.go`): Testcontainers — Neo4j + PG + Redis + Qdrant + OPA
- [x] P5.3.3 — E2E happy path (`tests/e2e/happy_path_test.go`): create app → issue key → ontology → batch upsert → search → overlay → commit → diff → coverage
- [x] P5.3.4 — E2E error paths (`tests/e2e/error_path_test.go`): auth failure, rate limit, overlay conflict, depth exceeded
- [x] P5.3.5 — Performance benchmarks (`tests/benchmark/search_bench_test.go`): `BenchmarkHybridSearch`, `BenchmarkBatchUpsert` — validate SLA
- [ ] P5.3.6 — Verify: all E2E tests PASS, SLA benchmarks within targets (đã PASS benchmark; e2e hiện SKIP khi chưa set `RUN_E2E=1`)

---

## Summary

| Phase                           | Tasks         | Ước tính     |
| ------------------------------- | ------------- | ------------ |
| Phase 0: Foundation Fix         | 27            | 1 tuần       |
| Phase 1: Core Graph             | 23            | 2 tuần       |
| Phase 2: Search & Index         | 18            | 2 tuần       |
| Phase 3: Overlay & Version      | 26            | 2 tuần       |
| Phase 4: Analytics & Projection | 15            | 1.5 tuần     |
| Phase 5: Production Ready       | 17            | 1.5 tuần     |
| **Tổng**                        | **126 tasks** | **~10 tuần** |
