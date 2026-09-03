# TD-008: Knowledge Graph Test Design

**Liên kết Requirements:** [TR-008-knowledge-graph.md](../requirements/TR-008-knowledge-graph.md)  
**Source:** `references/agentmemory/src/functions/graph.ts`  
**Test file:** `tests/agentmemory/specs/knowledge-graph.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Knowledge Graph lưu entities (nodes) và relationships (edges) được trích xuất từ observations.

**Các điểm kiểm thử:**
- Graph extraction từ observation text
- Node deduplication qua name index (`graphNameIndex`)
- Edge deduplication qua edge key index (`graphEdgeKey`)
- Graph retrieval (search by entity, expand from chunks)
- Graph snapshot cho performance tại scale lớn

---

## 2. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| Deduplication | Insert cùng node 2 lần → verify 1 node trong KV |
| Relationship types | Parameterized test với 6 relationship types |
| Entity search | Keyword matching trên node names |
| Scale | Graph với 50 nodes, verify snapshot |

---

## 3. Test Cases

### Group A: Graph Node Operations

#### TC-001 — Node được tạo với đúng structure
**Requirement:** TR-008-GRF-001 | **Type:** unit | 🔴 P0

**Given:** GraphNode được tạo mới  
**When:** Node được ghi vào KV  
**Then:** Node có:
- `id`: unique string
- `name`: entity name
- `type`: valid node type
- `observationIds[]`: observations nó được extract từ
- `createdAt`: ISO timestamp
- `degree`: int (số edges)

---

#### TC-002 — Node được store theo `{scope: "mem:graph:nodes", key: nodeId}`
**Type:** unit | 🔴 P0

**Given:** Node N với id `node_abc`  
**When:** Node được save  
**Then:** Tìm được trong KV tại scope `mem:graph:nodes`, key `node_abc`

---

#### TC-003 — Node dedup: cùng `type|name` → cùng nodeId
**Requirement:** TR-008-GRF-003 | **Type:** integration | 🔴 P0

**Given:** Name index `mem:graph:name-index` trống  
**When:**
1. Extract entity "jose_library" type "library" → nodeId1
2. Extract "jose_library" type "library" lại → nodeId2

**Then:** `nodeId1 = nodeId2` (dedup via name index), không tạo duplicate

---

#### TC-004 — Node name index: key format `{type}|{name}`
**Type:** unit | 🟠 P1

**Given:** Node với type="library", name="jose"  
**When:** Name index được update  
**Then:** Key `"library|jose"` được set trong `mem:graph:name-index` → nodeId

---

### Group B: Graph Edge Operations

#### TC-005 — Edge được tạo với đúng structure
**Requirement:** TR-008-GRF-002 | **Type:** unit | 🔴 P0

**Given:** Edge giữa node A và node B  
**When:** Edge được tạo  
**Then:** Edge có:
- `id`: unique
- `sourceId`, `targetId`
- `type`: valid relationship type
- `weight`: float (strength)
- `observationIds[]`
- `createdAt`

---

#### TC-006 — 6 relationship types được chấp nhận
**Requirement:** TR-008-GRF-005 | **Type:** unit | 🟠 P1

**Given:** Edge được tạo với từng type:
- `uses`, `implements`, `extends`, `calls`, `imports`, `defines`

**When:** Type được validate  
**Then:** Tất cả 6 types được accept, type khác bị reject hoặc fallback

---

#### TC-007 — Edge dedup: cùng `{src}|{tgt}|{type}` → same edgeId
**Requirement:** TR-008-GRF-004 | **Type:** integration | 🔴 P0

**Given:** Edge key index trống  
**When:**
1. Create edge: src=A, tgt=B, type=uses → edgeId1
2. Create edge: src=A, tgt=B, type=uses lại → edgeId2

**Then:** `edgeId1 = edgeId2` (dedup via edge key index)

---

#### TC-008 — Bidirectional search: A→B và B→A đều tìm được qua graph
**Requirement:** TR-008-GRF-006 | **Type:** integration | 🟡 P2

**Given:** Edge A `uses` B  
**When:** Graph search cho "B" (target)  
**Then:** Obs liên quan đến A cũng xuất hiện (traversal cả chiều)

---

### Group C: Graph Snapshot

#### TC-009 — Snapshot được tạo với top-degree nodes
**Requirement:** TR-008-GRF-007 | **Type:** integration | 🟠 P1

**Given:** 20 nodes với degrees khác nhau  
**When:** Snapshot được generated  
**Then:** Snapshot chứa top-N nodes theo degree (không phải toàn bộ graph)

---

#### TC-010 — Snapshot key cố định: `"current"` tại `mem:graph:snapshot`
**Type:** unit | 🟡 P2

**Given:** Snapshot được update  
**When:** KV được inspect  
**Then:** Chỉ có 1 entry tại scope `mem:graph:snapshot`, key `"current"`

---

### Group D: Graph Retrieval

#### TC-011 — `searchByEntities`: tìm obs liên quan đến entity
**Requirement:** TR-008-GRF-008 | **Type:** integration | 🔴 P0

**Given:**
- Node "jose" → liên kết với obsId "obs_jose_1", "obs_jose_2"
- Entity name "jose" được tìm kiếm

**When:** `searchByEntities(["jose"], 2, 10)` gọi  
**Then:** Results chứa obsIds liên quan đến "jose"

---

#### TC-012 — `expandFromChunks`: tìm neighbors của known observations
**Requirement:** TR-008-GRF-009 | **Type:** integration | 🟠 P1

**Given:** 5 top vector results (obsIds)  
**When:** `expandFromChunks(obsIds, maxHops=1, limit=5)` gọi  
**Then:** Neighbor observations được trả về (connected qua graph edges)

---

### Group E: Node Degree Tracking

#### TC-013 — Degree được increment khi edge mới được thêm
**Requirement:** TR-008-GRF-010 | **Type:** integration | 🟠 P1

**Given:** Node A với degree=0  
**When:** Edge A→B được tạo  
**Then:** `graphNodeDegree[A] = 1`

---

#### TC-014 — Degree không âm khi edge bị xóa
**Type:** unit | 🟡 P2

**Given:** Node A với degree=1, edge A→B  
**When:** Edge được xóa  
**Then:** `degree(A) = 0` (không âm)

---

### Group F: Graph Types

#### TC-015 — Node types: valid set được chấp nhận
**Requirement:** TR-008-GRF-001 | **Type:** unit | 🟠 P1

**Given:** Node được tạo với các types: `file`, `function`, `class`, `module`, `concept`, `entity`, `api`, `service`  
**When:** Node được stored  
**Then:** Tất cả types được accept

---

## 4. Coverage Notes

- Graph extraction được trigger indirectly qua `mem::graph-extract` function
- Unit tests tập trung vào graph node/edge KV operations và dedup logic
- Integration tests cần populate observations trước khi test graph queries
