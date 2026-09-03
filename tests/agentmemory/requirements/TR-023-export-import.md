# TR-023: Export, Import & Data Migration Test Requirements

**Module:** Export/Import, Migration (migrate.ts)  
**Nguồn:** SRS §7.1, TDD §13, URD §3.9  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-023-EXP-001 — Export: đầy đủ data structures
🔴 P0 | `[INT]`

**When:** `GET /agentmemory/export`  
**Then:** ExportData bao gồm:
```typescript
{
  version: string,           // current version
  exportedAt: string,        // ISO timestamp
  sessions: Session[],
  observations: Record<sessionId, CompressedObservation[]>,
  memories: Memory[],
  summaries: SessionSummary[]
}
```

**Traceability:** TDD §13.2, SRS §7.1

---

## TR-023-EXP-002 — Export: optional full backup
🟡 P2 | `[INT]`

**When:** `GET /agentmemory/export?full=true`  
**Then:** Additional fields present:
- `graphNodes?: GraphNode[]`
- `graphEdges?: GraphEdge[]`
- `semanticMemories?: SemanticMemory[]`
- `proceduralMemories?: ProceduralMemory[]`
- `actions?: Action[]`

**Traceability:** TDD §13.2

---

## TR-023-EXP-003 — Export: version field
🟠 P1 | `[UNIT]`

**When:** Export được generated  
**Then:** `version` field = current agentmemory version (e.g., "0.9.27")

**Traceability:** TDD §13.2

---

## TR-023-EXP-004 — Import: restore sessions và memories
🔴 P0 | `[INT]`

**Given:** Clean KV state  
**When:** `POST /agentmemory/import` với valid ExportData  
**Then:**
- Sessions restored trong KV
- Memories restored
- Observations restored
- Search index rebuilt từ imported data

**Traceability:** SRS §7.1, TDD §13.2

---

## TR-023-EXP-005 — Import: idempotent (safe to re-import)
🟠 P1 | `[INT]`

**Given:** Data đã được imported trước đó  
**When:** Same ExportData được import lại  
**Then:**
- Không tạo duplicate records
- IDs giữ nguyên
- Không conflict errors

**Traceability:** TDD §13.2

---

## TR-023-EXP-006 — Export pagination
🟡 P2 | `[INT]`

**Given:** 10,000 observations trong KV  
**When:** `GET /agentmemory/export?page=1&pageSize=1000`  
**Then:**
- 1000 observations trong response
- `pagination.total`, `pagination.page`, `pagination.totalPages` correct

**Traceability:** TDD §13.2

---

## TR-023-EXP-007 — Migration: schema version tracking
🔴 P0 | `[INT]` | **TDD §13.1**

**Given:** Fresh iii-engine KV  
**When:** Worker start  
**Then:**
- Schema version được read từ KV (hoặc default nếu không có)
- Pending migrations được identified
- Migrations chạy theo version order

**Traceability:** TDD §13.1

---

## TR-023-EXP-008 — Migration 0.5.0: add isLatest field
🟠 P1 | `[INT]`

**Given:** KV có memories không có `isLatest` field  
**When:** Migration 0.5.0 chạy  
**Then:** Tất cả memories được update với `isLatest = true`

**Traceability:** TDD §13.1

---

## TR-023-EXP-009 — Migration: không chạy lại đã-executed
🔴 P0 | `[INT]`

**Given:** Migration 0.5.0 đã chạy, schema version = 0.5.0  
**When:** Worker restart  
**Then:** Migration 0.5.0 KHÔNG chạy lại

**Traceability:** TDD §13.1

---

## TR-023-EXP-010 — Index persistence: save sau khi index thay đổi
🔴 P0 | `[INT]`

**Given:** New observations được added (index dirty)  
**When:** 30 giây debounce expires  
**Then:**
- `bm25-index.json` được written
- `vector-index.json` được written (nếu vector index exists)

**Traceability:** TDD §10.2, Architecture §9.3

---

## TR-023-EXP-011 — Index persistence: save on SIGTERM
🔴 P0 | `[INT]`

**Given:** Pending index changes (dirty)  
**When:** `SIGTERM` signal nhận được  
**Then:** Indexes được saved ngay lập tức trước khi shutdown

**Traceability:** TDD §10.2

---

## TR-023-EXP-012 — Index persistence: load on startup
🔴 P0 | `[INT]`

**Given:** `bm25-index.json` và `vector-index.json` tồn tại  
**When:** Worker khởi động  
**Then:**
- BM25 index được restored từ file
- Vector index được restored
- Dimension validation chạy trước restore
- Search hoạt động ngay lập tức (không cần rebuild)

**Traceability:** TDD §10.2, Architecture §9.3
