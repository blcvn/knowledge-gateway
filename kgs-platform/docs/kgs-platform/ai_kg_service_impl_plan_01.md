# Kế hoạch Fix: AI KG Service — 3 Issues từ Report

> **Ngày tạo:** 05/03/2026  
> **Tham chiếu:** [Report](report_impl_ai_kg_service.md) | [API Specs](ai_kg_service_api_specs.md) | [LLD](lld_ai_kg_service.md)  
> **Scope:** Fix 3 issues: NATS JetStream, X-Org-ID, Embedding Client  
> **Codebase:** `services/ai-kg-service/kgs-platform/`

---

## Issue C1: NATS JetStream — Thay In-Memory Adapter bằng Real Client

### Phân tích hiện trạng

- **File:** `internal/data/nats.go` (135 lines)
- **Vấn đề:** `NATSClient` dùng `map[string]natsSubscription` in-memory, `Publish()` gọi handler trực tiếp
- **Library `nats.go` chưa có trong `go.mod`** — cần thêm `github.com/nats-io/nats.go`
- **Public interface hiện tại:**
  - `NATSClient.Publish(ctx, subject, payload) error`
  - `NATSClient.Subscribe(subject, handler) (func(), error)`
  - `NATSClient.Ping(ctx) error`
  - `NATSHandler = func(context.Context, []byte)`
- **Callers (không cần đổi):**
  - `overlay/overlay.go` — Publish `OVERLAY_DISCARD`, `OVERLAY_COMMIT`
  - `overlay/nats_listener.go` — Subscribe `SESSION_CLOSE`, `BUDGET_STOP`
  - `service/health.go` — `Ping()`
- **Topics (giữ nguyên):** `nats_topics.go` — 4 patterns đúng DOC 4 §5
- **Config hiện tại:** `conf.proto` có `Data.NATS { url, stream }`

### Tasks

#### C1.1 — Thêm dependency `nats.go`

**Files:** `go.mod`, `go.sum`

```bash
cd services/ai-kg-service/kgs-platform
go get github.com/nats-io/nats.go@latest
```

---

#### C1.2 — Implement real NATS client (giữ public interface)

**File:** `internal/data/nats.go` — REWRITE

**Yêu cầu:**

- Struct `NATSClient` wrap `*nats.Conn` thay vì `map[string]natsSubscription`
- `NewNATSClientFromConfig()` → `nats.Connect()` với options: Timeout, ReconnectWait, MaxReconnects, DisconnectErrHandler (log)
- `Publish(ctx, subject, payload)` → `nc.Publish(subject, payload)` + ctx deadline check
- `Subscribe(subject, handler)` → `nc.Subscribe(subject, func(msg *nats.Msg) { handler(ctx, msg.Data) })`, return `sub.Unsubscribe` func
- `Ping(ctx)` → `nc.FlushTimeout(timeout)` (thay vì TCP dial hiện tại)
- `Close()` method bổ sung → `nc.Drain()` + `nc.Close()`
- Giữ fallback `nil` return khi `cfg.Url == ""` (backward compatible)

**Signature giữ nguyên:**

```go
type NATSHandler func(context.Context, []byte)

func NewNATSClientFromConfig(cfg *conf.Data_NATS, logger *log.Helper) (*NATSClient, error)
func (c *NATSClient) Publish(ctx context.Context, subject string, payload []byte) error
func (c *NATSClient) Subscribe(subject string, handler NATSHandler) (func(), error)
func (c *NATSClient) Ping(ctx context.Context) error
```

---

#### C1.3 — Cập nhật DI/Wire

**File:** `internal/data/data.go` (nếu cần)

- Đảm bảo `NewNATSClientFromConfig` vẫn được inject đúng
- Thêm `Close()` vào cleanup func trong `NewData()`

---

#### C1.4 — Cập nhật config mặc định

**File:** `configs/config.yaml`

```yaml
nats:
  url: 'nats://localhost:4222' # thay vì ""
  stream: 'kg-events'
```

---

#### C1.5 — Unit tests cho NATS client

**File:** `internal/data/nats_test.go` — NEW

- Test `NewNATSClientFromConfig()` với empty URL → returns nil
- Test `Publish()` / `Subscribe()` với mock nats server (hoặc dùng `nats-server -p ...` embedded)
- Test `Ping()` success + failure
- Test `Subscribe()` → `unsubscribe()` cleanup

---

#### C1.6 — Cập nhật docker-compose.dev.yml

**File:** `deployment/docker/docker-compose.dev.yml`

- Verify NATS container có `ports: ["4222:4222"]`
- Verify kgs-platform `NATS_URL` env var đúng

---

## Issue C2: X-Org-ID Header — Thêm OrgID vào AppContext

### Phân tích hiện trạng

- **Auth middleware** (`middleware/auth.go` L25-29): `AppContext` chỉ có `AppID`, `Scopes`, `TenantID`
- **Namespace middleware** (`middleware/namespace.go`): Dùng `appCtx.AppID` + `appCtx.TenantID` → không include OrgID
- **DOC 4 §2 yêu cầu:** `X-Org-ID` header optional, dùng cho enterprise multi-org

### Tasks

#### C2.1 — Thêm `OrgID` vào `AppContext`

**File:** `internal/server/middleware/auth.go`

**Thay đổi:**

```go
type AppContext struct {
    AppID    string
    Scopes   string
    TenantID string
    OrgID    string   // NEW — DOC 4 §2
}
```

---

#### C2.2 — Parse `X-Org-ID` từ request header

**File:** `internal/server/middleware/auth.go` — trong hàm `Auth()`

**Thêm sau dòng 65 (trước `return handler(ctx, req)`):**

```go
// Parse X-Org-ID header (DOC 4 §2)
if orgID := strings.TrimSpace(tr.RequestHeader().Get("X-Org-ID")); orgID != "" {
    appCtx.OrgID = orgID
}
```

---

#### C2.3 — Include OrgID trong namespace computation (nếu cần)

**File:** `internal/biz/namespace.go`

**Xem xét:** Nếu multi-org cần namespace isolation, update `ComputeNamespace()`:

```go
// BEFORE: namespace = appID/tenantID
// AFTER (optional): namespace = orgID/appID/tenantID (nếu orgID non-empty)
```

> ⚠️ **Quyết định cần user input:** Có nên include OrgID trong namespace không? Nếu không, OrgID chỉ dùng cho audit/logging.

---

#### C2.4 — Cập nhật Redis cache cho AppContext

**File:** `internal/server/middleware/auth.go`

- Update `cacheAppContext()` / `readCachedAppContext()` để serialize/deserialize `OrgID` field
- Đảm bảo JSON format tương thích (thêm `org_id` field)

---

#### C2.5 — Unit tests

**File:** `internal/server/middleware/auth_test.go` (nếu có, hoặc tạo mới)

- Test `X-Org-ID: "org-123"` → `AppContext.OrgID == "org-123"`
- Test `X-Org-ID` header missing → `AppContext.OrgID == ""`
- Test `X-Org-ID` header empty/whitespace → `AppContext.OrgID == ""`
- Test Redis cache round-trip với OrgID

---

## Issue C3: Embedding Client — Thay DeterministicEmbeddingClient bằng AI Proxy

### Phân tích hiện trạng

- **Interface:** `EmbeddingClient { Embed(ctx, text) ([]float32, error) }` (`search/vector.go` L14-16)
- **Implementation hiện tại:** `DeterministicEmbeddingClient` (SHA-256 hash → vector 1536-dim)
- **Callers:**
  - `search/vector.go` → `VectorSearcher.Search()` — vector search
  - `batch/vector_indexer.go` → `QdrantIndexer.Index()` — entity indexing
  - `batch/dedup.go` → `SemanticDeduper` — dedup check

### Phân tích AI Proxy Proto

**Proto:** `kratos-proto/schema/ai-proxy/ai_proxy.proto`

- **Chỉ có `Complete` và `StreamComplete` RPC** — **KHÔNG có `Embed` RPC**
- `CompletePayload`: `model_id`, `prompt`, `messages`, `max_tokens`, `temperature`, `top_p`, `stop`, `stream`
- `CompletionResponse`: `text`, `prompt_tokens`, `completion_tokens`, `total_tokens`, `latency_ms`

### Chiến lược

> ⚠️ **AI Proxy hiện tại KHÔNG có Embed RPC.** Có 2 options:
>
> **Option A (khuyến nghị):** Dùng `Complete` RPC với prompt yêu cầu trả JSON embedding vector
>
> - Pro: Không cần sửa ai-proxy proto
> - Con: Chất lượng phụ thuộc vào model + prompt, output cần parse JSON
>
> **Option B:** Bổ sung `Embed` RPC vào ai-proxy proto
>
> - Pro: Clean & standard
> - Con: Cần sửa cả ai-proxy service

### Tasks (theo Option A — dùng Complete RPC)

#### C3.1 — Thêm ai-proxy proto dependency

**Files:** `go.mod`, thêm ai-proxy proto generated code

```bash
# Nếu dùng module path từ kratos-proto
go get github.com/blcvn/kratos-proto/go/ai-proxy@latest
```

Hoặc copy generated Go code vào `internal/client/aiproxy/`.

---

#### C3.2 — Implement `AIProxyEmbeddingClient`

**File:** `internal/client/aiproxy/embedding_client.go` — NEW

```go
package aiproxy

type AIProxyEmbeddingClient struct {
    grpcClient aiproxy.AIProxyServiceClient
    modelID    string        // e.g. "text-embedding-3-small"
    vectorSize int           // 1536
}

func NewAIProxyEmbeddingClient(conn *grpc.ClientConn, modelID string, vectorSize int) *AIProxyEmbeddingClient

// Embed implement EmbeddingClient interface
func (c *AIProxyEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error)
```

**Logic `Embed()`:**

- Gọi `Complete` RPC với prompt: `"Generate a JSON array of {vectorSize} float32 embedding values for: {text}"`
- Parse response `completion.Text` → `[]float32`
- Normalize vector
- Fallback: Nếu parse fail → return error (hoặc fallback sang DeterministicEmbeddingClient)

---

#### C3.3 — Config cho AI Proxy connection

**File:** `internal/conf/conf.proto`

```protobuf
message Data {
    // ... existing fields
    AIProxy ai_proxy = 7;
}

message AIProxy {
    string url = 1;              // "ai-proxy:50051"
    string embedding_model = 2;  // "text-embedding-3-small"
    int32 vector_size = 3;       // 1536
}
```

**File:** `configs/config.yaml`

```yaml
ai_proxy:
  url: 'ai-proxy:50051'
  embedding_model: 'text-embedding-3-small'
  vector_size: 1536
```

---

#### C3.4 — Cập nhật DI/Wire

**File:** `internal/data/data.go` hoặc DI provider tương ứng

- Tạo gRPC dial connection tới ai-proxy
- Inject `AIProxyEmbeddingClient` thay thế `DeterministicEmbeddingClient`
- Giữ fallback: nếu ai-proxy URL empty → dùng `DeterministicEmbeddingClient` (cho dev/test)

---

#### C3.5 — Cập nhật `VectorSearcher` constructor

**File:** `internal/search/vector.go` L50-55

**Thay đổi:**

```go
// BEFORE:
func NewVectorSearcher(qdrant *data.QdrantClient, embedder *DeterministicEmbeddingClient) *VectorSearcher

// AFTER: Accept interface thay vì concrete type
func NewVectorSearcher(qdrant *data.QdrantClient, embedder EmbeddingClient) *VectorSearcher
```

---

#### C3.6 — Cập nhật batch indexer + dedup

**Files:**

- `internal/batch/vector_indexer.go` — update constructor nhận `EmbeddingClient` interface
- `internal/batch/dedup.go` — update `embedDeterministic()` → dùng injected `EmbeddingClient`

---

#### C3.7 — Unit tests cho AIProxyEmbeddingClient

**File:** `internal/client/aiproxy/embedding_client_test.go` — NEW

- Test `Embed()` với mocked gRPC Complete response
- Test parse float32 array from JSON text
- Test error handling khi ai-proxy trả lỗi
- Test vector normalization

---

## Tổng hợp Tasks

### Timeline đề xuất

| Phase                         | Duration | Tasks                                    |
| ----------------------------- | -------- | ---------------------------------------- |
| **Phase 1: X-Org-ID**         | 0.5 ngày | C2.1, C2.2, C2.3, C2.4, C2.5             |
| **Phase 2: NATS JetStream**   | 2 ngày   | C1.1, C1.2, C1.3, C1.4, C1.5, C1.6       |
| **Phase 3: Embedding Client** | 2 ngày   | C3.1, C3.2, C3.3, C3.4, C3.5, C3.6, C3.7 |
| **Phase 4: Integration Test** | 1 ngày   | E2E test toàn bộ 3 fixes                 |

### Task Checklist

#### Phase 1: X-Org-ID (0.5 ngày)

- [ ] C2.1 — Thêm `OrgID string` vào `AppContext` struct (`middleware/auth.go`)
- [ ] C2.2 — Parse `X-Org-ID` header trong `Auth()` middleware (`middleware/auth.go`)
- [ ] C2.3 — Quyết định: OrgID có include trong namespace computation? (`biz/namespace.go`)
- [ ] C2.4 — Cập nhật Redis cache serialize/deserialize OrgID (`middleware/auth.go`)
- [ ] C2.5 — Unit tests: X-Org-ID parsing + cache round-trip (`middleware/auth_test.go`)

#### Phase 2: NATS JetStream (2 ngày)

- [ ] C1.1 — `go get github.com/nats-io/nats.go@latest` (`go.mod`)
- [ ] C1.2 — Rewrite `NATSClient` dùng real `nats.Conn` (`data/nats.go`)
  - [ ] C1.2.1 — `NewNATSClientFromConfig()` → `nats.Connect()` với reconnect options
  - [ ] C1.2.2 — `Publish()` → `nc.Publish(subject, payload)` + ctx check
  - [ ] C1.2.3 — `Subscribe()` → `nc.Subscribe()`, return unsubscribe func
  - [ ] C1.2.4 — `Ping()` → `nc.FlushTimeout(timeout)`
  - [ ] C1.2.5 — `Close()` → `nc.Drain()` + `nc.Close()`
- [ ] C1.3 — Cập nhật DI cleanup trong `NewData()` (`data/data.go`)
- [ ] C1.4 — Cập nhật default NATS URL trong config (`configs/config.yaml`)
- [ ] C1.5 — Unit tests cho real NATS client (`data/nats_test.go`)
- [ ] C1.6 — Verify docker-compose NATS container config (`deployment/docker/docker-compose.dev.yml`)

#### Phase 3: Embedding Client (2 ngày)

- [ ] C3.1 — Thêm ai-proxy proto dependency (`go.mod` hoặc copy generated code)
- [ ] C3.2 — Implement `AIProxyEmbeddingClient` dùng `Complete` RPC (`client/aiproxy/embedding_client.go`)
  - [ ] C3.2.1 — gRPC dial connection setup
  - [ ] C3.2.2 — `Embed()` → Build embedding prompt → call `Complete` → parse JSON array
  - [ ] C3.2.3 — Vector normalization + validation
  - [ ] C3.2.4 — Error handling + timeout
- [ ] C3.3 — Thêm AIProxy config vào `conf.proto` + `config.yaml`
- [ ] C3.4 — Cập nhật DI/Wire: inject `AIProxyEmbeddingClient` với fallback (`data/data.go`)
- [ ] C3.5 — Đổi `NewVectorSearcher()` nhận `EmbeddingClient` interface (`search/vector.go`)
- [ ] C3.6 — Đổi batch indexer + dedup nhận `EmbeddingClient` interface (`batch/vector_indexer.go`, `batch/dedup.go`)
- [ ] C3.7 — Unit tests cho AIProxyEmbeddingClient (`client/aiproxy/embedding_client_test.go`)

#### Phase 4: Integration Test (1 ngày)

- [ ] E2E.1 — Verify NATS pub/sub cross-process: overlay commit → event published → subscriber receives
- [ ] E2E.2 — Verify X-Org-ID header round-trip: request → AppContext → audit log
- [ ] E2E.3 — Verify embedding: entity index → vector search → semantic results
- [ ] E2E.4 — `docker-compose up -d` → full stack running with real NATS

---

## Verification Plan

### Automated Tests

```bash
# Chạy tất cả unit tests
cd services/ai-kg-service/kgs-platform
go test ./internal/data/ -run TestNATS -v
go test ./internal/server/middleware/ -run TestAuth -v
go test ./internal/client/aiproxy/ -run TestEmbedding -v

# Chạy integration tests
go test ./tests/integration/ -v -tags=integration
```

### Manual Verification

1. **NATS:** `docker-compose up -d` → tạo overlay → commit → kiểm tra NATS message trên `overlay.committed.*` topic bằng `nats sub overlay.committed.>` CLI
2. **X-Org-ID:** `curl -H "X-Org-ID: org-123" ... /v1/graph/context/...` → verify response header hoặc log chứa OrgID
3. **Embedding:** Gọi `HybridSearch` API → verify semantic score khác 0 và kết quả có ý nghĩa ngữ nghĩa (không phải hash match)

---

## Lưu ý quan trọng

> [!IMPORTANT]
> **AI Proxy hiện tại KHÔNG có `Embed` RPC.** Task C3.2 sử dụng `Complete` RPC làm workaround.
> Nếu muốn clean hơn, cần bổ sung `Embed` RPC vào `kratos-proto/schema/ai-proxy/ai_proxy.proto` trước.

> [!WARNING]
> **NATS interface phải backward compatible.** Signature `Publish()`, `Subscribe()`, `Ping()` giữ nguyên
> để không cần sửa callers (`overlay/`, `service/health.go`).
