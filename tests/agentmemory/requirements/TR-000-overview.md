# TR-000: Tổng quan & Cross-Cutting Test Requirements

**Module:** Cross-cutting / System-wide  
**Nguồn:** PRD §1-3, SRS §2, §4, Architecture §1-3, URD §all  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Mục tiêu kiểm thử

Bộ test requirements này nhằm xác minh rằng **agentmemory** đáp ứng đầy đủ các yêu cầu từ:
- PRD: Product vision, use cases, performance targets
- SRS: Functional & non-functional requirements
- URD: End-to-end user stories và acceptance criteria
- TDD/Architecture: Technical implementation contracts

---

## 2. Hệ thống dưới test (System Under Test)

| Component | Description |
|---|---|
| **agentmemory worker** | Node.js/TypeScript worker đăng ký với iii-engine |
| **REST API** | 128 endpoints tại `:3111` |
| **MCP Server** | 53 tools qua stdio/HTTP |
| **Viewer** | HTTP UI tại `:3113` |
| **Search Engine** | BM25 + Vector + Graph hybrid |
| **Storage** | iii-engine SQLite KV + file-based indexes |

---

## 3. Môi trường kiểm thử

### TR-000-ENV-001 — Zero-config startup
🔴 P0 | `[UNIT]`

**Given:** Node.js LTS được cài đặt, iii-engine v0.11.2 có sẵn  
**When:** `npx @agentmemory/agentmemory` được chạy mà không có environment variable nào  
**Then:**
- Server khởi động thành công trong vòng 30 giây
- REST API tại `:3111` phản hồi HTTP 200
- Viewer tại `:3113` phản hồi HTTP 200
- Không có lỗi crash

**Traceability:** UR-001, SRS §4.5, PRD §7

---

### TR-000-ENV-002 — 12-factor configuration
🟠 P1 | `[UNIT]`

**Given:** Các environment variables được set  
**When:** Worker khởi động  
**Then:** Mọi config parameter được load từ env var, không từ file cứng

**Test Data:**
```bash
TOKEN_BUDGET=3000
MAX_OBS_PER_SESSION=200
BM25_WEIGHT=0.5
VECTOR_WEIGHT=0.7
III_REST_PORT=4000
```

**Traceability:** UR-027, SRS §4.4

---

### TR-000-ENV-003 — Local-first default (no external calls)
🔴 P0 | `[INT]`

**Given:** Không có API key nào được cấu hình  
**When:** Worker xử lý hook events và search queries  
**Then:**
- Không có HTTP/HTTPS request nào ra bên ngoài
- Synthetic compression được dùng (không gọi LLM)
- BM25 search hoạt động (không cần vector embedding)

**Traceability:** UR-029, UR-038, SRS §4.6

---

## 4. Cross-cutting Concerns

### TR-000-GRD-001 — Graceful degradation: no embedding provider
🔴 P0 | `[INT]`

**Given:** Không có embedding provider được cấu hình (`EMBEDDING_PROVIDER` không set)  
**When:** `mem::smart-search` được gọi  
**Then:**
- Không throw error
- Kết quả trả về dùng BM25-only (không có vector scores)
- `vectorScore` = 0 hoặc null trong kết quả

**Traceability:** SRS §4.2, Architecture §1.3, TDD §11.2

---

### TR-000-GRD-002 — Graceful degradation: LLM provider unavailable
🔴 P0 | `[INT]`

**Given:** LLM API key có cấu hình nhưng provider trả lỗi  
**When:** Bất kỳ function nào cần LLM (compress, consolidate, summarize)  
**Then:**
- Circuit breaker bắt lỗi
- Fallback về synthetic compression
- Không crash worker
- Error được log nhưng không propagate lên caller

**Traceability:** SRS §4.2, TDD §11.1, §8.3

---

### TR-000-GRD-003 — Graceful degradation: iii-engine down
🟠 P1 | `[INT]`

**Given:** iii-engine không khởi động được  
**When:** MCP standalone mode được active  
**Then:**
- 7 core MCP tools vẫn hoạt động với in-memory KV
- Standalone mode tự động kích hoạt
- Không block agent workflow

**Traceability:** TDD §9.2, Architecture §3.2

---

### TR-000-ERR-001 — Invalid payload handling
🔴 P0 | `[UNIT]`

**Given:** Request với payload thiếu required fields  
**When:** Bất kỳ `mem::*` function nào nhận payload  
**Then:**
- Return `{success: false, error: "<message>"}` — KHÔNG throw
- HTTP 400 từ REST endpoint
- Không crash worker

**Test Cases:**
| Field thiếu | Expected error |
|---|---|
| `sessionId` | "sessionId is required" |
| `hookType` | "hookType is required" |
| `timestamp` | "timestamp is required" |

**Traceability:** TDD §11.1, SRS §3.2

---

### TR-000-ERR-002 — UnhandledRejection suppression
🔴 P0 | `[UNIT]`

**Given:** Một async function throw error không được catch  
**When:** `unhandledRejection` event được emitted  
**Then:**
- Worker KHÔNG crash
- Log được throttled (max 1 lần/phút)
- Worker tiếp tục xử lý requests tiếp theo

**Traceability:** TDD §11.3, SRS §4.2

---

### TR-000-LOG-001 — Structured logging
🟡 P2 | `[UNIT]`

**Given:** Bất kỳ operation nào  
**When:** Logger được gọi  
**Then:**
- Output là valid JSON với fields: `level`, `message`, `timestamp`
- Boot logs được buffer và in summary khi "Ready" (không verbose mode)
- Verbose mode: in ngay lập tức lên stderr

**Traceability:** Architecture §12.3, TDD §1.1

---

### TR-000-COMPAT-001 — Platform compatibility (macOS/Linux)
🟠 P1 | `[E2E]`

**Given:** macOS arm64/x64 hoặc Linux x64/arm64  
**When:** Worker khởi động và chạy  
**Then:**
- Tất cả features hoạt động bình thường
- File paths được xử lý đúng (forward slash)
- iii-engine binary tương thích

**Traceability:** SRS §4.6, PRD §12

---

### TR-000-COMPAT-002 — TypeScript strict mode compliance
🟡 P2 | `[UNIT]`

**Given:** Source code  
**When:** TypeScript compiler với `strict: true`  
**Then:**
- Zero TypeScript compilation errors
- No `any` types trong exported interfaces
- All function signatures có return type annotations

**Traceability:** TDD §1.1, SRS §4.4

---

## 5. Non-Functional Requirements Baseline

### TR-000-NFR-001 — Test coverage baseline
🟠 P1

| Module | Min Coverage |
|---|---|
| `state/search-index.ts` | ≥ 90% |
| `state/vector-index.ts` | ≥ 85% |
| `state/hybrid-search.ts` | ≥ 80% |
| `functions/observe.ts` | ≥ 75% |
| `functions/remember.ts` | ≥ 80% |
| `functions/compress-synthetic.ts` | ≥ 85% |
| `providers/*` | ≥ 70% |
| **Overall** | **≥ 75%** |

**Traceability:** TDD §14.4

---

### TR-000-NFR-002 — CI benchmark gate
🟠 P1 | `[PERF]`

**Given:** CI pipeline chạy eval suite  
**When:** `npm run eval:longmemeval`  
**Then:**
- R@5 ≥ 95.2%
- R@10 ≥ 98.6%
- MRR ≥ 88.2%

**Traceability:** PRD §7, SRS §3.5 FR-SEARCH-005

---

## 6. Acceptance Criteria từ URD

Mapping đầy đủ từ URD §4 sang test requirements:

| UR-ID | Acceptance Criterion | Test File |
|---|---|---|
| UR-001 | `npx @agentmemory/agentmemory` starts in <30s | TR-022 |
| UR-006 | 12 hook events captured per session | TR-002 |
| UR-011 | Memories injected at session start | TR-010 |
| UR-012 | Semantic recall without keyword match | TR-006 |
| UR-016 | `memory_governance_delete` cascades | TR-013 |
| UR-017 | Jaccard >0.7 → supersede, `isLatest=false` | TR-007 |
| UR-021 | Viewer live stream at `:3113` | TR-019 |
| UR-024 | Cross-agent memory sharing | TR-011 |
| UR-032 | `GET /health` → `{status: "ok"}` HTTP 200 | TR-019 |
| UR-038 | No external network calls by default | TR-000-ENV-003 |
