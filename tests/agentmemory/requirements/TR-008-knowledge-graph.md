# TR-008: Knowledge Graph Test Requirements

**Module:** Knowledge Graph (graph.ts, graph-retrieval.ts, temporal-graph.ts)  
**Nguồn:** SRS §3.6 (FR-GRAPH-001..004), Architecture §5.4, §6.3, TDD §5.1-5.3  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho knowledge graph extraction, node merging, bi-temporal edges, graph snapshot và graph-based retrieval.

---

## TR-008-GRP-001 — Graph extraction: disabled by default
🔴 P0 | `[UNIT]` | **FR-GRAPH-001**

**Given:** `GRAPH_EXTRACTION_ENABLED=false` (default)  
**When:** Observation được lưu  
**Then:**
- `mem::graph-extract` KHÔNG được trigger
- Không có LLM call cho graph extraction
- Search vẫn hoạt động bình thường (BM25+Vector only)

**Traceability:** FR-GRAPH-001, SRS §9.3

---

## TR-008-GRP-002 — Graph extraction: 13 node types
🟠 P1 | `[INT]` | **FR-GRAPH-001**

**Given:** `GRAPH_EXTRACTION_ENABLED=true`, LLM available  
**When:** Observation với rich content được processed  
**Then:** Hệ thống có thể extract nodes với 13 types:
`file`, `function`, `concept`, `error`, `decision`, `pattern`, `library`, `person`, `project`, `preference`, `location`, `organization`, `event`

**Traceability:** FR-GRAPH-001, SRS §3.6

---

## TR-008-GRP-003 — Graph extraction: 15 edge types
🟠 P1 | `[INT]` | **FR-GRAPH-001**

**Given:** Graph extraction enabled  
**When:** Relations được extracted  
**Then:** Hỗ trợ 15 edge types:
`uses`, `imports`, `modifies`, `causes`, `fixes`, `depends_on`, `related_to`, `works_at`, `prefers`, `blocked_by`, `caused_by`, `optimizes_for`, `rejected`, `avoids`, `located_in`, `succeeded_by`

**Traceability:** FR-GRAPH-001, SRS §3.6

---

## TR-008-GRP-004 — Node merge: dedup by name + type
🔴 P0 | `[UNIT]` | **FR-GRAPH-001**

**Given:** GraphNode "jose" (type=library) đã tồn tại  
**When:** Cùng observation extract entity "jose" (type=library)  
**Then:**
- Không tạo node mới
- Existing node được update: `sourceObservationIds` appended, properties merged
- Name index `graph-name-index/library:jose` vẫn trỏ đến existing node

**Traceability:** TDD §5.1, Architecture §6.3

---

## TR-008-GRP-005 — Node merge: khác type → khác node
🟠 P1 | `[UNIT]`

**Given:** Node "auth" (type=concept) đã tồn tại  
**When:** Entity "auth" (type=function) được extract  
**Then:** Node mới được tạo (khác type → khác key trong name index)

**Traceability:** TDD §5.1

---

## TR-008-GRP-006 — GraphEdge bi-temporal fields
🔴 P0 | `[UNIT]` | **FR-GRAPH-002**

**Given:** Edge được tạo  
**When:** Edge object được lưu  
**Then:** Fields đầy đủ:
```typescript
{
  id: string,
  type: GraphEdgeType,
  sourceNodeId: string,
  targetNodeId: string,
  weight: number,
  tcommit: string,     // when recorded (transaction time)
  tvalid: string,      // when relationship became true
  tvalidEnd?: string,  // null = currently valid
  version: number,
  isLatest: boolean
}
```

**Traceability:** FR-GRAPH-002, SRS §6.2

---

## TR-008-GRP-007 — Temporal query: AS OF
🟠 P1 | `[UNIT]` | **FR-GRAPH-002**

**Given:** Edge with:
- `tvalid = "2026-01-01"`, `tvalidEnd = "2026-06-01"`

**When:** `queryAtTime(nodeId, new Date("2026-03-15"))`  
**Then:** Edge được trả về (T=2026-03-15 is within valid range)

**Traceability:** FR-GRAPH-002, TDD §5.2

---

## TR-008-GRP-008 — Temporal query: AS OF ngoài range
🟠 P1 | `[UNIT]`

**Given:** Edge tvalidEnd = "2026-01-01"  
**When:** `queryAtTime(nodeId, new Date("2026-06-10"))`  
**Then:** Edge KHÔNG được trả về (đã expired)

**Traceability:** FR-GRAPH-002, TDD §5.2

---

## TR-008-GRP-009 — Graph snapshot: rebuild
🟠 P1 | `[INT]` | **FR-GRAPH-003**

**Given:** 100+ nodes trong graph  
**When:** `mem::graph-snapshot-rebuild` được trigger  
**Then:**
- GraphSnapshot được tạo với:
  - `topNodes`: top-N nodes by degree centrality
  - `topEdges`: edges connecting top nodes
  - `topDegrees`: degree scores
  - `dirty = false`

**Traceability:** FR-GRAPH-003, TDD §5.1, Architecture §6.3

---

## TR-008-GRP-010 — Graph snapshot: dirty flag
🟠 P1 | `[UNIT]`

**Given:** GraphSnapshot đã được build (`dirty=false`)  
**When:** Observation mới được processed và graph-extract chạy  
**Then:** `dirty = true` (invalidate snapshot)

**Traceability:** FR-GRAPH-003, Architecture §6.3

---

## TR-008-GRP-011 — Graph retrieval: 2-hop traversal
🔴 P0 | `[INT]` | **FR-GRAPH-004**

**Given:**
- Node A "jose" → edge → Node B "middleware" → edge → Node C (obs2)
- obs1 có entity "jose"

**When:** Graph search với entity "jose", depth=2  
**Then:**
- obs1 được tìm thấy (direct)
- obs2 được tìm thấy (2-hop via middleware)

**Traceability:** FR-GRAPH-004, TDD §5.3, Architecture §5.4

---

## TR-008-GRP-012 — Graph retrieval: entity name lookup
🟠 P1 | `[INT]`

**Given:** Query "jose middleware authentication"  
**When:** `extractEntitiesFromQuery()` chạy  
**Then:** `["jose", "middleware", "authentication"]` extracted

**Traceability:** TDD §5.3, Architecture §5.4

---

## TR-008-GRP-013 — Graph query: pagination
🟡 P2 | `[INT]` | **FR-GRAPH-004**

**Given:** Graph với 100 nodes  
**When:** `GET /graph?limit=10&offset=0`  
**Then:**
- 10 nodes trả về
- `truncated=true` nếu có nhiều results hơn limit
- `fromSnapshot=true` nếu dùng cached snapshot

**Traceability:** FR-GRAPH-004, SRS §3.6

---

## TR-008-GRP-014 — GraphNode structure
🔴 P0 | `[UNIT]`

**Given:** Node được tạo  
**When:** `kv.get(KV.graph, nodeId)`  
**Then:** Structure đúng:
```typescript
{
  id: string,
  type: GraphNodeType,
  name: string,
  properties: Record<string, unknown>,
  sourceObservationIds: string[],
  createdAt: string,
  aliases?: string[],
  stale?: boolean
}
```

**Traceability:** SRS §6.2

---

## TR-008-GRP-015 — Session end triggers graph extraction
🟠 P1 | `[INT]`

**Given:** `GRAPH_EXTRACTION_ENABLED=true`, session có 5 observations  
**When:** Hook `session_end` được gửi  
**Then:** Graph extraction được trigger cho các observations chưa được extract

**Traceability:** SRS §3.6

---

## TR-008-GRP-016 — Graph reset
🟡 P2 | `[INT]`

**Given:** Graph với 50 nodes và 100 edges  
**When:** `memory_graph_reset` MCP tool được gọi  
**Then:**
- Tất cả nodes bị xóa
- Tất cả edges bị xóa
- GraphSnapshot bị xóa
- Audit record được tạo

**Traceability:** SRS §11 MCP tools
