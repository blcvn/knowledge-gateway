# AI KG Service — Tasks Fix (3 Issues)

> **Impl Plan:** [ai_kg_service_impl_plan_01.md](ai_kg_service_impl_plan_01.md)  
> **Report:** [report_impl_ai_kg_service.md](report_impl_ai_kg_service.md)  
> **Codebase:** `services/ai-kg-service/kgs-platform/`  
> **Tổng:** 26 tasks | 4 phases | ~5.5 ngày

---

## Phase 1 — X-Org-ID Header (0.5 ngày)

> Thêm xử lý `X-Org-ID` vào middleware, đảm bảo DOC 4 §2 compliance.

- [x] C2.1 — Thêm `OrgID string` vào `AppContext` struct (`internal/server/middleware/auth.go`)
- [x] C2.2 — Parse `X-Org-ID` header trong hàm `Auth()`, gán vào `appCtx.OrgID` (`internal/server/middleware/auth.go`)
- [x] C2.3 — Xem xét include OrgID trong `ComputeNamespace()` nếu multi-org cần namespace isolation (`internal/biz/namespace.go`)
- [x] C2.4 — Cập nhật `cacheAppContext()` / `readCachedAppContext()` serialize/deserialize field `OrgID` (`internal/server/middleware/auth.go`)
- [x] C2.5 — Unit tests: parse `X-Org-ID` header (present/missing/empty) + Redis cache round-trip (`internal/server/middleware/auth_test.go` — NEW nếu chưa có)

---

## Phase 2 — NATS JetStream (2 ngày)

> Thay in-memory NATS adapter bằng real `nats.go` client, giữ nguyên public interface.

- [x] C1.1 — Thêm dependency: `go get github.com/nats-io/nats.go@latest` (`go.mod`, `go.sum`)
- [x] C1.2 — Rewrite `NATSClient` struct wrap `*nats.Conn` thay vì `map[string]natsSubscription` (`internal/data/nats.go`)
  - [x] C1.2.1 — `NewNATSClientFromConfig()`: gọi `nats.Connect()` với Timeout(3s), ReconnectWait(500ms), MaxReconnects(5), DisconnectErrHandler(log)
  - [x] C1.2.2 — `Publish(ctx, subject, payload)`: check `ctx.Done()` → `nc.Publish(subject, payload)`
  - [x] C1.2.3 — `Subscribe(subject, handler)`: `nc.Subscribe(subject, func(msg) { handler(ctx, msg.Data) })`, return `sub.Unsubscribe` func
  - [x] C1.2.4 — `Ping(ctx)`: `nc.FlushTimeout(timeout)` thay TCP dial hiện tại
  - [x] C1.2.5 — Thêm `Close()` method: `nc.Drain()` + `nc.Close()`
- [x] C1.3 — Cập nhật DI cleanup: thêm `natsClient.Close()` vào cleanup func trong `NewData()` (`internal/data/data.go`)
- [x] C1.4 — Cập nhật default config: `nats.url: "nats://localhost:4222"` thay vì `""` (`configs/config.yaml`)
- [x] C1.5 — Unit tests cho real NATS client (`internal/data/nats_test.go` — NEW)
  - [x] C1.5.1 — Test `NewNATSClientFromConfig()` empty URL → return nil (backward compatible)
  - [x] C1.5.2 — Test `Publish()` + `Subscribe()` round-trip
  - [x] C1.5.3 — Test `Ping()` success + failure
  - [x] C1.5.4 — Test `Subscribe()` → `unsubscribe()` cleanup
- [x] C1.6 — Verify docker-compose NATS container: ports `4222:4222`, kgs-platform env `NATS_URL` (`deployment/docker/docker-compose.dev.yml`)

---

## Phase 3 — Embedding Client (2 ngày)

> Thay `DeterministicEmbeddingClient` (SHA-256 hash) bằng `AIProxyEmbeddingClient` gọi ai-proxy `Complete` RPC.
>
> ⚠️ AI Proxy proto hiện **không có `Embed` RPC**, dùng `Complete` RPC làm workaround.

- [ ] C3.1 — Thêm ai-proxy proto generated code vào project (`go.mod` hoặc copy vào `internal/client/aiproxy/`)
- [ ] C3.2 — Implement `AIProxyEmbeddingClient` (`internal/client/aiproxy/embedding_client.go` — NEW)
  - [ ] C3.2.1 — Struct: `grpcClient aiproxy.AIProxyServiceClient`, `modelID string`, `vectorSize int`
  - [ ] C3.2.2 — `NewAIProxyEmbeddingClient(conn *grpc.ClientConn, modelID string, vectorSize int)`
  - [ ] C3.2.3 — `Embed(ctx, text)`: build embedding prompt → call `Complete` RPC → parse response text thành `[]float32`
  - [ ] C3.2.4 — Vector normalization + validation (reuse `normalizeVector()` từ `search/vector.go`)
  - [ ] C3.2.5 — Error handling: timeout, parse failure, ai-proxy unavailable
- [ ] C3.3 — Thêm AIProxy config section
  - [ ] C3.3.1 — Thêm `AIProxy { url, embedding_model, vector_size }` vào `conf.proto` + regenerate
  - [ ] C3.3.2 — Thêm default values vào `configs/config.yaml`: `url: "ai-proxy:50051"`, `embedding_model: "text-embedding-3-small"`, `vector_size: 1536`
- [ ] C3.4 — Cập nhật DI/Wire: tạo gRPC connection, inject `AIProxyEmbeddingClient`; fallback `DeterministicEmbeddingClient` nếu URL empty (`internal/data/data.go`)
- [ ] C3.5 — Đổi `NewVectorSearcher()` nhận `EmbeddingClient` interface thay vì `*DeterministicEmbeddingClient` concrete (`internal/search/vector.go`)
- [ ] C3.6 — Đổi `NewQdrantIndexer()` + `NewSemanticDeduper()` nhận `EmbeddingClient` interface (`internal/batch/vector_indexer.go`, `internal/batch/dedup.go`)
- [ ] C3.7 — Unit tests cho `AIProxyEmbeddingClient` (`internal/client/aiproxy/embedding_client_test.go` — NEW)
  - [ ] C3.7.1 — Test `Embed()` với mocked gRPC `Complete` response
  - [ ] C3.7.2 — Test parse `[]float32` từ JSON text response
  - [ ] C3.7.3 — Test error handling khi ai-proxy trả lỗi / timeout
  - [ ] C3.7.4 — Test vector normalization output

---

## Phase 4 — Integration & E2E Test (1 ngày)

> Verify toàn bộ 3 fixes hoạt động end-to-end.

- [x] E2E.1 — NATS cross-process: overlay commit → verify event nhận trên `overlay.committed.*` topic
- [x] E2E.2 — X-Org-ID round-trip: request với `X-Org-ID: "org-123"` → verify `AppContext.OrgID` trong handler
- [ ] E2E.3 — Embedding: index entities → `HybridSearch` → verify semantic results có ý nghĩa (không phải hash match)
- [ ] E2E.4 — Full stack: `docker-compose up -d` → tất cả services running với real NATS

### Cập nhật triển khai (2026-03-05)

- Đã hoàn thành toàn bộ PHASE 1 (C2.1 → C2.5).
- Đã hoàn thành toàn bộ PHASE 2 (C1.1 → C1.6).
- Đã hoàn thành PHASE 4: E2E.1 và E2E.2.
- Theo yêu cầu hiện tại: skip E2E.3 và E2E.4.
- Verification đã chạy:
  - `go test ./internal/server/middleware ./internal/data ./internal/overlay` (PASS)
