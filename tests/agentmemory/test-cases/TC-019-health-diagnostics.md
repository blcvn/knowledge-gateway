# TC-019: Health & Diagnostics — Test Cases

**Test Design tham chiếu:** [TD-019](../designs/TD-019-health-diagnostics.md)  
**Requirements tham chiếu:** [TR-019](../requirements/TR-019-health-diagnostics.md)  
**Module:** Health Check, Status Metrics, Diagnose  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-019-001: GET /health → 200 khi tất cả services healthy

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-019-HLT-001 |

**Điều kiện tiên quyết:**
- Server đang chạy
- KV accessible (iii-engine running)
- Memory usage trong giới hạn bình thường

**HTTP Request:** `GET /health`

**Kết quả mong đợi:**
- HTTP 200
- `Content-Type: application/json`
- Body:
  - `status = "ok"`
  - `uptime` là number > 0 (giây kể từ start)
  - `version` là string (semver format)

---

## TC-019-002: GET /health → 503 khi KV không accessible

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-019-HLT-002 |

**Điều kiện tiên quyết:** KV được inject lỗi (simulate: iii-engine stopped hoặc connection refused)

**HTTP Request:** `GET /health`

**Kết quả mong đợi:**
- HTTP 503 (Service Unavailable)
- Body:
  - `status = "error"`
  - `error` field chứa thông báo về KV

---

## TC-019-003: Health response không expose sensitive info

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-003 |
| **Loại** | Security |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-019-HLT-003 |

**Điều kiện tiên quyết:** Server running với `AGENTMEMORY_SECRET = "super-secret-key"` và `ANTHROPIC_API_KEY = "sk-ant-test"` được set

**HTTP Request:** `GET /health`

**Kết quả mong đợi — Response KHÔNG chứa:**
- `super-secret-key` (API secret)
- `sk-ant-test` (Anthropic API key)
- Absolute file paths (e.g., `/Users/binhnt/...`)
- Stack traces
- Environment variable values

**Các fields an toàn được phép có:**
- `status`, `uptime`, `version`, `build` (commit hash OK)

---

## TC-019-004: GET /status trả về aggregate metrics

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-019-HLT-004 |

**Điều kiện tiên quyết:** KV có 50 sessions, 500 obs, 20 memories, 10 graph nodes

**HTTP Request:** `GET /status`

**Kết quả mong đợi:**
- HTTP 200
- Body có các metrics:
  - `metrics.totalSessions` = 50
  - `metrics.totalObservations` = 500
  - `metrics.totalMemories` = 20
  - `metrics.graphNodeCount` = 10
  - `system.heapUsedMB` — number > 0
  - `system.uptime` — number > 0

---

## TC-019-005: mem::diagnose báo cáo vector dimension mismatch

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-019-HLT-005 |

**Setup (seed corrupted data):**
- `obs_correct`: 4-dim vector
- `obs_wrong`: 8-dim vector (dimension mismatch)

**Các bước thực hiện:**
1. Seed cả 2 observations vào vector index
2. Gọi `mem::diagnose`
3. Kiểm tra kết quả

**Kết quả mong đợi:**
- `result.vectorIndex.consistent = false`
- `result.vectorIndex.mismatches` chứa entry với `obsId = "obs_wrong"`, `dim = 8`
- Không crash

---

## TC-019-006: mem::diagnose báo cáo BM25 vs Vector count mismatch

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🟡 P2 |
| **Requirement** | TR-019-HLT-005 |

**Setup:**
- BM25 index: 100 documents
- Vector index: 95 documents (5 missing)

**Kết quả mong đợi:**
- `result.bm25Count = 100`
- `result.vectorCount = 95`
- `result.countMismatch = 5`

---

## TC-019-007: GET /metrics trả về operational counters

| Trường | Giá trị |
|---|---|
| **ID** | TC-019-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🟡 P2 |

**Điều kiện tiên quyết:** Server đã xử lý ít nhất 10 observations và 5 recalls

**HTTP Request:** `GET /metrics`

**Kết quả mong đợi:**
- `counters.observationsProcessed >= 10`
- `counters.recallsExecuted >= 5`
- `counters.memoriesCreated` — number >= 0

---

## Tổng kết TC-019

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-019-001 | /health OK → 200 | 🔴 P0 | Integration |
| TC-019-002 | /health KV fail → 503 | 🔴 P0 | Integration |
| TC-019-003 | Health no sensitive info | 🔴 P0 | Security |
| TC-019-004 | /status metrics | 🟠 P1 | Integration |
| TC-019-005 | diagnose dim mismatch | 🟠 P1 | Integration |
| TC-019-006 | diagnose count mismatch | 🟡 P2 | Integration |
| TC-019-007 | /metrics counters | 🟡 P2 | Integration |
