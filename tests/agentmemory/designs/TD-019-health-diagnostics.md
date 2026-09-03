# TD-019: Health & Diagnostics Test Design

**Liên kết Requirements:** [TR-019-health-diagnostics.md](../requirements/TR-019-health-diagnostics.md)  
**Source:** `references/agentmemory/src/health/`, `functions/diagnostics.ts`  
**Test file:** `tests/agentmemory/specs/health-diagnostics.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Test Cases

### Group A: Health Check

#### TC-001 — `GET /health` trả về 200 khi tất cả services OK
**Requirement:** TR-019-HLT-001 | **Type:** integration | 🔴 P0

**Given:** Server running, KV accessible, memory usage trong giới hạn  
**When:** `GET /health`  
**Then:**
- Status 200
- Body: `{status: "ok", uptime: <seconds>, version: "..."}`

---

#### TC-002 — `GET /health` trả về 503 khi KV không accessible
**Requirement:** TR-019-HLT-002 | **Type:** integration | 🔴 P0

**Given:** KV bị inject lỗi (simulate disconnected)  
**When:** `GET /health`  
**Then:**
- Status 503
- Body: `{status: "error", error: "kv unavailable"}`

---

#### TC-003 — Health check không expose sensitive info
**Requirement:** TR-019-HLT-003 | **Type:** security | 🔴 P0

**Given:** Server running  
**When:** `GET /health`  
**Then:**
- Response KHÔNG chứa: API keys, secrets, file paths, stack traces
- Chỉ chứa safe fields: status, uptime, version, build info

---

### Group B: Diagnostics

#### TC-004 — `GET /status` trả về memory metrics
**Requirement:** TR-019-HLT-004 | **Type:** integration | 🟠 P1

**Given:** Server với 50 sessions, 500 observations, 20 memories  
**When:** `GET /status`  
**Then:** Body có:
- `metrics.totalSessions`
- `metrics.totalObservations`
- `metrics.totalMemories`
- `metrics.graphNodeCount`
- `system.heapUsedMB`: number

---

#### TC-005 — `GET /metrics` trả về operational counters
**Type:** integration | 🟡 P2

**Given:** Server đã xử lý 100 observations, 10 recalls  
**When:** `GET /metrics`  
**Then:**
- `counters.observationsProcessed >= 100`
- `counters.recallsExecuted >= 10`
- `counters.memoriesCreated`: number

---

#### TC-006 — `mem::diagnose` trả về dimension consistency report
**Requirement:** TR-019-HLT-005 | **Type:** integration | 🟠 P1

**Given:** Vector index có mixed dimensions (384 và 768 — corrupted data)  
**When:** `mem::diagnose` gọi  
**Then:**
- `{vectorIndex: {mismatches: [{obsId, dim}], consistent: false}}`
- Không crash

---

#### TC-007 — `mem::diagnose` báo cáo BM25 size vs vector size mismatch
**Type:** integration | 🟡 P2

**Given:** BM25 có 100 docs, Vector có 95 docs (5 missing)  
**When:** `mem::diagnose` gọi  
**Then:** `{bm25Count: 100, vectorCount: 95, mismatch: 5}`

---

### Group C: Search Followup Diagnostics

#### TC-008 — `mem:recent-searches` được tracked sau recall
**Requirement:** TR-019-HLT-006 | **Type:** integration | 🟡 P2

**Given:** `mem::recall({sessionId: "sess_A", query: "auth"})` gọi  
**When:** KV inspected  
**Then:** `mem:recent-searches["sess_A"]` = `{query: "auth", timestamp: ...}`

---

## 2. Coverage Notes

- Health endpoint cần test 3 scenarios: OK, KV error, memory pressure
- Metrics endpoint cần test với non-zero counters (seed data first)
- Diagnose cần seed corrupted data thay vì real corruption
