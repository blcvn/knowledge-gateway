# TR-019: Health Monitoring & Diagnostics Test Requirements

**Module:** Health Monitor, Diagnostics  
**Nguồn:** SRS §3.14 (FR-DIAG-001..003), Architecture §12, URD §3.8  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-019-HLT-001 — GET /health: trả về status ok
🔴 P0 | `[INT]` | **FR-DIAG-001**

**Given:** Server running và healthy  
**When:** `GET /agentmemory/health`  
**Then:**
- HTTP 200
- `{status: "ok"}` (hoặc `"healthy"`)

**Traceability:** FR-DIAG-001, UR-032

---

## TR-019-HLT-002 — HealthSnapshot: các fields
🟠 P1 | `[INT]` | **FR-DIAG-001**

**When:** Health được query  
**Then:** HealthSnapshot có:
- `connectionState`: iii-engine connection status
- `workers[]`: worker statuses
- `memory.heapUsed`, `memory.heapTotal`, `memory.rss`
- `cpu.userMicros`, `cpu.systemMicros`
- `eventLoopLag`: milliseconds
- `uptime`: seconds

**Traceability:** FR-DIAG-001, Architecture §12.1

---

## TR-019-HLT-003 — Health status levels
🔴 P0 | `[INT]` | **FR-DIAG-001**

**Given:** Various system states  
**When:** Health check chạy  
**Then:**
- Normal: `status = "healthy"`
- heapUsed > 500MB: `status = "degraded"` và alert generated
- eventLoopLag > 100ms: `status = "degraded"`
- KV connectivity fail: `status = "critical"`

**Traceability:** FR-DIAG-001, Architecture §12.1

---

## TR-019-HLT-004 — Health monitor: runs every 30s
🟡 P2 | `[UNIT]`

**Given:** HealthMonitor khởi tạo  
**When:** Monitor chạy  
**Then:** Metrics collected mỗi 30 giây

**Traceability:** Architecture §12.1

---

## TR-019-HLT-005 — Doctor command: check iii-engine
🔴 P0 | `[E2E]` | **FR-DIAG-002**

**Given:** `agentmemory doctor` được chạy  
**When:** Diagnostic checks complete  
**Then:** iii-engine connectivity được checked và reported

**Traceability:** FR-DIAG-002, UR-031

---

## TR-019-HLT-006 — Doctor command: check port availability
🟠 P1 | `[E2E]`

**Given:** Port 3111 đang bị occupied  
**When:** `agentmemory doctor`  
**Then:** Port conflict được detect và fix suggestion được đưa ra

**Traceability:** FR-DIAG-002

---

## TR-019-HLT-007 — Doctor command: check hook registration
🟠 P1 | `[E2E]`

**Given:** Claude Code chưa có hooks installed  
**When:** `agentmemory doctor`  
**Then:** Hook installation status được checked, `agentmemory connect claude-code` được suggest

**Traceability:** FR-DIAG-002, UR-031

---

## TR-019-HLT-008 — Doctor command: check MCP config
🟠 P1 | `[E2E]`

**Given:** MCP config missing cho target agent  
**When:** `agentmemory doctor`  
**Then:** MCP config issue được detect và instructions được show

**Traceability:** FR-DIAG-002

---

## TR-019-HLT-009 — Doctor: auto suggest fixes
🟠 P1 | `[E2E]`

**Given:** Multiple issues detected  
**When:** `agentmemory doctor` output  
**Then:** Mỗi issue có actionable fix command được suggest

**Traceability:** FR-DIAG-002, UR-031

---

## TR-019-HLT-010 — OpenTelemetry: metrics đúng
🟡 P2 | `[INT]`

**Given:** OpenTelemetry configured  
**When:** Operations xảy ra  
**Then:** Metrics được export:
- `memory_observations_total` (counter)
- `memory_search_latency_ms` (histogram)
- `memory_compress_latency_ms` (histogram)
- `memory_consolidation_runs_total` (counter)

**Traceability:** Architecture §12.2

---

## TR-019-HLT-011 — Viewer: live stream hoạt động
🔴 P0 | `[E2E]`

**Given:** Viewer tại `:3113` open  
**When:** Observation được gửi  
**Then:** Observation xuất hiện trong live stream

**Traceability:** UR-021, Architecture §3.2

---

## TR-019-HLT-012 — PID file: graceful shutdown
🟠 P1 | `[INT]`

**Given:** Worker chạy, `worker.pid` file tồn tại  
**When:** `SIGTERM` được sent  
**Then:**
- Indexes được save (không mất BM25/vector data)
- Process exit cleanly
- PID file cleaned up

**Traceability:** SRS §4.2, Architecture §9.3
