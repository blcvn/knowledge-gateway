# TD-000: Test Infrastructure & Chiến lược Kiểm thử

**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Mục tiêu tổng thể

Bộ test cho **agentmemory** nhằm xác minh:
- Tính đúng đắn của từng module (unit test)
- Tính nhất quán giữa các module qua iii-engine KV (integration test)
- Hành vi end-to-end từ hook đến recall (e2e test)
- Đáp ứng SLA latency (performance test)

---

## 2. Phân loại kiểm thử

| Loại | Mục đích | Công cụ | Tỷ lệ | Môi trường |
|---|---|---|---|---|
| **Unit** | Kiểm thử hàm/class độc lập | Vitest + mockKV + mockSdk | 60% | In-memory |
| **Integration** | Kiểm thử qua KV state | Vitest + mockKV | 30% | In-memory |
| **E2E** | Kiểm thử server thực + HTTP | Vitest + supertest | 8% | Test server |
| **Performance** | Đo latency và throughput | Vitest + `performance.now()` | 2% | Isolated |

---

## 3. Môi trường kiểm thử

### 3.1 Unit/Integration Tests
- **KV store:** In-memory Map (thay thế iii-engine SQLite)
- **SDK:** Mock object với function registry
- **LLM Provider:** noop (không có API calls)
- **Embedding:** None (BM25-only mode)
- **Environment variables mặc định:**
  - `AGENTMEMORY_AUTO_COMPRESS=false`
  - `EMBEDDING_PROVIDER=none`
  - Không set `AGENTMEMORY_SECRET` (local mode)

### 3.2 E2E Tests
- **Server:** Thực sự khởi động agentmemory process trên port ngẫu nhiên
- **Transport:** HTTP/REST trực tiếp
- **Cleanup:** Server được kill sau mỗi test suite

### 3.3 Performance Tests
- Chạy riêng biệt, không mix với unit tests
- Cần môi trường sạch (không background processes)
- Kết quả được log ra với p50/p95

---

## 4. Chiến lược Mock

### 4.1 MockKV
Thay thế iii-engine KV store bằng in-memory Map 2 cấp:
- Cấp 1: `scope` (namespace như `mem:sessions`, `mem:obs:sessId`)
- Cấp 2: `key → value`

**Operations cần mock:**
- `get(scope, key)` → value hoặc null
- `set(scope, key, value)` → value
- `delete(scope, key)` → void
- `list(scope)` → value[]
- `update(scope, key, patches[])` → void (apply path patches)

### 4.2 MockSdk
Thay thế iii-sdk với function registry in-memory:
- `registerFunction(id, handler)` → đăng ký
- `trigger({function_id, payload})` → gọi handler theo id
- `registerTrigger` → spy only (không xử lý)

### 4.3 Noop Embedding Provider
Trả về zero vector (384 dims) — đủ để test cấu trúc mà không call API ngoài.

### 4.4 Deterministic Embedding Provider (cho hybrid search tests)
Dùng hash của text để tạo sparse unit vector — cho phép kiểm soát similarity.

---

## 5. Shared Fixtures

| Fixture | Mục đích | Fields mặc định |
|---|---|---|
| `makeObs()` | CompressedObservation | id, sessionId, timestamp, type, title, concepts, facts, files |
| `makeRaw()` | RawObservation | id, sessionId, timestamp, hookType, toolName, toolInput, toolOutput |
| `makeMemory()` | Memory | id, type="architecture", strength=7, version=1, isLatest=true |
| `makeSession()` | Session | id, project, cwd, status="active", observationCount=0 |
| `makeHookPayload()` | HookPayload | sessionId, hookType, timestamp, project, cwd, data |

Mỗi fixture hỗ trợ `overrides` để customize từng field.

---

## 6. Kỹ thuật kiểm thử

### 6.1 Equivalence Partitioning
Phân vùng dữ liệu đầu vào thành các lớp tương đương:
- **Valid:** Dữ liệu đúng format, đúng kiểu
- **Invalid:** Thiếu field bắt buộc, sai kiểu
- **Boundary:** Đúng ngưỡng giới hạn (ví dụ MAX_OBS_PER_SESSION)
- **Edge case:** Empty string, null, undefined, zero length

### 6.2 Boundary Value Analysis
Áp dụng cho các ngưỡng có giá trị cụ thể:
- `MAX_OBS_PER_SESSION`: test tại 499, 500, 501
- `similarity threshold 0.7`: test tại 0.69, 0.70, 0.71
- `title/narrative truncation`: test tại n-1, n, n+1 chars
- `TTL 5 phút`: test lúc 4m59s, 5m00s, 5m01s

### 6.3 State Transition Testing
Cho các state machine (session status, memory version, action status):
- Mỗi trạng thái hợp lệ → trạng thái hợp lệ tiếp theo
- Trạng thái không hợp lệ → hành vi từ chối

### 6.4 Error Injection
- Inject lỗi vào KV operations → kiểm tra rollback
- Inject lỗi embedding → kiểm tra fallback
- Inject lỗi LLM → kiểm tra noop path

---

## 7. Coverage Strategy

### 7.1 Targets

| Module | Target Coverage |
|---|---|
| `state/search-index.ts` | 90% |
| `state/vector-index.ts` | 85% |
| `state/hybrid-search.ts` | 80% |
| `state/keyed-mutex.ts` | 95% |
| `functions/observe.ts` | 75% |
| `functions/remember.ts` | 80% |
| `functions/compress-synthetic.ts` | 90% |
| `functions/privacy.ts` | 95% |
| `functions/dedup.ts` | 90% |
| `functions/governance.ts` | 80% |

### 7.2 Excluded từ coverage
- `src/version.ts` (generated)
- `src/xenova.d.ts` (type declarations)
- `src/eval/` (benchmark scripts)

---

## 8. Isolation & Cleanup

- Mỗi test file có KV instance độc lập (không shared state)
- `beforeEach`: khởi tạo KV và SDK mới
- `afterEach`: clear fake timers, restore spies
- Không dùng global state giữa test cases
- Env vars được save/restore quanh từng test

---

## 9. Test Data Management

### 9.1 Static fixtures
Lưu tại `tests/agentmemory/fixtures/`:
- `sample-session.json` — Session với 12 observations đa dạng
- `sample-memories.json` — 10 memories các loại
- `sample-graph.json` — Graph với 20 nodes, 30 edges
- `claude-transcript.jsonl` — Transcript cho replay import test

### 9.2 Factory pattern
Dùng factory functions (makeObs, makeMemory, ...) thay vì hardcoded objects — cho phép test tập trung vào field cần test, các field còn lại có default hợp lệ.

---

## 10. CI/CD Integration

| Stage | Tests chạy | Trigger |
|---|---|---|
| PR check | Unit + Integration | Mỗi PR |
| Main merge | Unit + Integration + E2E | Merge vào main |
| Nightly | Unit + Integration + E2E + Performance | Mỗi đêm |
| Release | Toàn bộ bao gồm LongMemEval | Trước release |
