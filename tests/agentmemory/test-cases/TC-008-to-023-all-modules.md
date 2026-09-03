# TC-008 đến TC-023: Các Module Còn Lại — Test Cases

**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

> Tài liệu này gộp các module từ TC-008 đến TC-023 để tiện tham khảo.  
> Mỗi module sẽ được tách thành file riêng khi cần.

---

# TC-008: Knowledge Graph

**Design ref:** [TD-008](../designs/TD-008-knowledge-graph.md)

---

## TC-008-001: Graph node được tạo với đúng structure

| **ID** | TC-008-001 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Điều kiện tiên quyết:** KV store trống

**Dữ liệu đầu vào:**
- name: `"jose_library"`, type: `"library"`, observationIds: `["obs_001"]`

**Kết quả mong đợi:**
- Node có: `id`, `name`, `type`, `observationIds[]`, `createdAt`, `degree = 0`
- Node lưu trong KV tại scope `mem:graph:nodes`, key = nodeId

---

## TC-008-002: Node dedup — cùng `{type}|{name}` → cùng nodeId

| **ID** | TC-008-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước thực hiện:**
1. Extract entity `("library", "jose")` từ obs_A → lưu `nodeId1`
2. Extract entity `("library", "jose")` từ obs_B → lưu `nodeId2`
3. So sánh `nodeId1` và `nodeId2`

**Kết quả mong đợi:** `nodeId1 === nodeId2` (không tạo duplicate)

---

## TC-008-003: Edge dedup — cùng `{src}|{tgt}|{type}` → cùng edgeId

| **ID** | TC-008-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước thực hiện:**
1. Create edge: src=nodeA, tgt=nodeB, type=`uses` → `edgeId1`
2. Create edge: src=nodeA, tgt=nodeB, type=`uses` → `edgeId2`

**Kết quả mong đợi:** `edgeId1 === edgeId2`

---

## TC-008-004: 6 relationship types được chấp nhận

| **ID** | TC-008-004 | **Priority** | 🟠 P1 | **Type** | Unit |
|---|---|---|---|---|---|

**Test với từng type:** `uses`, `implements`, `extends`, `calls`, `imports`, `defines`

**Kết quả mong đợi:** Tất cả 6 types được accept

---

## TC-008-005: searchByEntities tìm obs liên quan đến entity

| **ID** | TC-008-005 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Node "jose" liên kết với obsIds: `["obs_jose_1", "obs_jose_2"]`

**Kết quả mong đợi:**
- `searchByEntities(["jose"], 1, 5)` trả về results chứa obs_jose_1 và/hoặc obs_jose_2

---

## TC-008-006: Degree tăng khi edge được thêm

| **ID** | TC-008-006 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Node A, degree = 0

**Kết quả mong đợi:** Sau khi thêm edge A→B: `degree(A) = 1`

---

## TC-008-007: Graph snapshot chứa top-degree nodes

| **ID** | TC-008-007 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 20 nodes với degrees khác nhau

**Kết quả mong đợi:**
- Snapshot tồn tại tại KV scope `mem:graph:snapshot`, key `"current"`
- Snapshot chứa nodes có degree cao nhất (top-N)

---

---

# TC-009: Consolidation Pipeline

**Design ref:** [TD-009](../designs/TD-009-consolidation-pipeline.md)

---

## TC-009-001: Không trigger consolidation dưới ngưỡng

| **ID** | TC-009-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `CONSOLIDATION_THRESHOLD = 5`, session có 4 observations

**Kết quả mong đợi:**
- Sau hook thứ 4: KV `mem:summaries` KHÔNG có entry cho session này
- Không có consolidation job chạy

---

## TC-009-002: Trigger khi đạt đúng ngưỡng

| **ID** | TC-009-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `CONSOLIDATION_THRESHOLD = 5`, session có 4 observations

**Kết quả mong đợi:**
- Sau hook thứ 5: `mem:summaries[sessionId]` được tạo

---

## TC-009-003: Session summary có đúng structure

| **ID** | TC-009-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi — Summary có fields:**
- `sessionId`, `observationCount`, `timeRangeStart`, `timeRangeEnd`, `generatedAt`

---

## TC-009-004: Concurrent triggers không tạo duplicate summaries

| **ID** | TC-009-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 2 consolidation triggers đồng thời cho cùng session

**Kết quả mong đợi:**
- Chỉ 1 summary tồn tại (mutex bảo vệ)
- Không có duplicate memories từ cùng batch

---

## TC-009-005: Trigger mỗi N observations (periodic)

| **ID** | TC-009-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Threshold = 5, thêm 10 observations vào session đã có 0

**Kết quả mong đợi:**
- Consolidation trigger sau obs 5 (batch 1)
- Consolidation trigger sau obs 10 (batch 2)
- 2 summaries hoặc 1 summary được update

---

---

# TC-010: Context Injection (Recall)

**Design ref:** [TD-010](../designs/TD-010-context-injection.md)

---

## TC-010-001: Recall trả về kết quả khi có observations liên quan

| **ID** | TC-010-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 5 observations, 3 có từ "auth"

**Dữ liệu đầu vào:** `query = "auth"`, `sessionId = "sess_test"`, `maxObs = 5`

**Kết quả mong đợi:**
- `results.length >= 1`
- Mỗi result có `observation`, `combinedScore`
- Sorted by score descending

---

## TC-010-002: `maxObs` giới hạn số observations

| **ID** | TC-010-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 20 observations, tất cả liên quan query

**Kết quả mong đợi:** `result.observations.length ≤ 5` (với maxObs=5)

---

## TC-010-003: Recall chỉ trả về `isLatest = true` memories

| **ID** | TC-010-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- M1: `isLatest = false` (đã bị supersede)
- M2: `isLatest = true` (superseder)

**Kết quả mong đợi:** Recall response chứa M2, không chứa M1

---

## TC-010-004: Default chỉ recall trong cùng sessionId

| **ID** | TC-010-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 2 sessions, cả 2 có observations về "auth"

**Kết quả mong đợi:** Recall với `sessionId = "sess_A"` chỉ trả về obs từ sess_A

---

---

# TC-011: Multi-Agent

**Design ref:** [TD-011](../designs/TD-011-multi-agent.md)

---

## TC-011-001: Acquire lease thành công khi resource free

| **ID** | TC-011-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** `{success: true, leaseId: "..."}`, KV `mem:leases` có entry

---

## TC-011-002: Second agent bị từ chối khi resource đã có lease

| **ID** | TC-011-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Agent A đang giữ lease

**Kết quả mong đợi:** Agent B nhận `{success: false, error: "...locked..."}` hoặc tương đương

---

## TC-011-003: Lease hết TTL → resource released

| **ID** | TC-011-003 | **Priority** | 🔴 P0 | **Type** | Unit (fake timers) |
|---|---|---|---|---|---|

**Setup:** Lease với TTL = 5s

**Các bước thực hiện:**
1. Acquire lease
2. Advance time past TTL
3. Agent B cố acquire

**Kết quả mong đợi:** Agent B acquire thành công sau khi TTL hết

---

## TC-011-004: Observations mang agentId từ env

| **ID** | TC-011-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENT_ID = cursor-agent-1`

**Kết quả mong đợi:** `observation.agentId = "cursor-agent-1"`

---

## TC-011-005: Signal được deliver đến cùng teamId

| **ID** | TC-011-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Agents A và B cùng team, Agent C khác team

**Kết quả mong đợi:**
- Agent A publish → B nhận được
- Agent C không nhận

---

---

# TC-012: Orchestration

**Design ref:** [TD-012](../designs/TD-012-orchestration.md)

---

## TC-012-001: Action được tạo với status "pending"

| **ID** | TC-012-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- `action.status = "pending"`
- Action lưu trong `mem:actions`

---

## TC-012-002: State transition pending → in-progress → completed

| **ID** | TC-012-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:** pending → (update) → in-progress → (update) → completed

**Kết quả mong đợi:** Mỗi transition được accept

---

## TC-012-003: Sketch write/read/append/clear

| **ID** | TC-012-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Write `"Initial note"` → verify stored
2. Append `"Additional"` → verify `"Initial note\nAdditional"`
3. Clear → verify `""`

---

---

# TC-013: Governance & Audit

**Design ref:** [TD-013](../designs/TD-013-governance-audit.md)

---

## TC-013-001: mem::remember tạo audit record

| **ID** | TC-013-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- Audit record trong `mem:audit`
- Fields: `operation = "remember"`, `memoryId`, `timestamp`, `sessionId`

---

## TC-013-002: mem::forget tạo audit record

| **ID** | TC-013-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Audit record: `operation = "forget"`, `memoryId`

---

## TC-013-003: Audit records là immutable (không xóa khi memory bị forget)

| **ID** | TC-013-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Memory M với audit record R

**Kết quả mong đợi:** Sau `forget(M)`, record R vẫn tồn tại trong `mem:audit`

---

## TC-013-004: Retention sweep xóa expired memories

| **ID** | TC-013-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Memory A: `forgetAfter = now - 1 day` (expired)
- Memory B: `forgetAfter = now + 7 days` (active)
- Memory C: không có `forgetAfter`

**Kết quả mong đợi:**
- Sau sweep: A bị xóa, B và C vẫn còn

---

## TC-013-005: API key hợp lệ → 200

| **ID** | TC-013-005 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET = "test-key-123"`

**Request:** `Authorization: Bearer test-key-123`

**Kết quả mong đợi:** HTTP 200

---

## TC-013-006: API key sai → 401

| **ID** | TC-013-006 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request:** `Authorization: Bearer wrong-key`

**Kết quả mong đợi:** HTTP 401

---

---

# TC-014: MCP Server

**Design ref:** [TD-014](../designs/TD-014-mcp-server.md)

---

## TC-014-001: tools/list trả về đầy đủ tools

| **ID** | TC-014-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Response chứa: `mem_observe`, `mem_recall`, `mem_remember`, `mem_forget`, `mem_search`, `mem_status`

---

## TC-014-002: Mỗi tool có JSON Schema hợp lệ

| **ID** | TC-014-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Mỗi tool có `name`, `description`, `inputSchema.type = "object"`, `inputSchema.properties`

---

## TC-014-003: mem_recall trả về JSONRPC 2.0 response

| **ID** | TC-014-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Input:** `{query: "auth", sessionId: "sess_test"}`

**Kết quả mong đợi:**
- JSONRPC 2.0 format
- `result.content[0].type = "text"`
- `result.content[0].text` không rỗng

---

## TC-014-004: Tool invocation với invalid args trả về JSONRPC error

| **ID** | TC-014-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Input:** mem_recall không có sessionId

**Kết quả mong đợi:** Response có `error` field, không có `result`

---

---

# TC-015: REST API

**Design ref:** [TD-015](../designs/TD-015-rest-api.md)

---

## TC-015-001: `GET /sessions` trả về danh sách sessions

| **ID** | TC-015-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 3 sessions trong KV

**Kết quả mong đợi:**
- Status 200
- Body: `{sessions: [...]}` với 3 items

---

## TC-015-002: `GET /sessions/:id` với ID không tồn tại → 404

| **ID** | TC-015-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Status 404, body có `error` field

---

## TC-015-003: `GET /search?q=auth` trả về results sorted by score

| **ID** | TC-015-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- Status 200
- `results` array sorted by `combinedScore` descending

---

## TC-015-004: `POST /memories` tạo memory mới

| **ID** | TC-015-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request body:** `{content: "Auth uses JWT", type: "architecture"}`

**Kết quả mong đợi:**
- Status 201
- Body chứa memory với `version = 1`, `isLatest = true`

---

## TC-015-005: `DELETE /memories/:id` xóa memory

| **ID** | TC-015-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- Status 200, `{success: true, deleted: 1}`
- Subsequent GET → 404

---

## TC-015-006: Missing required field → 422

| **ID** | TC-015-006 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request:** `POST /memories` với body `{type: "fact"}` (thiếu content)

**Kết quả mong đợi:** Status 422

---

## TC-015-007: `GET /health` → 200 khi healthy

| **ID** | TC-015-007 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Status 200, `{status: "ok"}`

---

## TC-015-008: 500 error không expose stack trace

| **ID** | TC-015-008 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** KV inject lỗi

**Kết quả mong đợi:** Status 500, body `{error: "internal server error"}` (không có stack)

---

---

# TC-016: Session Replay

**Design ref:** [TD-016](../designs/TD-016-session-replay.md)

---

## TC-016-001: Import JSONL với valid transcript

| **ID** | TC-016-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** JSONL file với 10 valid observation records

**Kết quả mong đợi:**
- 10 observations trong KV
- Session được tạo với `source = "replay"`
- `{success: true, importedCount: 10}`

---

## TC-016-002: Malformed JSONL — skip invalid lines

| **ID** | TC-016-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** 8 valid lines + 2 invalid JSON lines

**Kết quả mong đợi:** `{importedCount: 8, skippedCount: 2}` (hoặc warning)

---

## TC-016-003: Import không duplicate khi gọi 2 lần

| **ID** | TC-016-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi (lần 2):** `{importedCount: 0, skippedCount: 10}`

---

## TC-016-004: Observation data được preserved đầy đủ

| **ID** | TC-016-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** toolName, toolInput, toolOutput, agentId đều preserved sau import

---

---

# TC-017: Memory Slots

**Design ref:** [TD-017](../designs/TD-017-memory-slots.md)

---

## TC-017-001: Slot write và read

| **ID** | TC-017-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- Write `{name: "current-task", content: "Implementing auth"}` → success
- Read `{name: "current-task"}` → `{slot: {name: "current-task", content: "Implementing auth"}}`

---

## TC-017-002: Slot overwrite khi cùng name

| **ID** | TC-017-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- Sau write lần 2 với cùng name: chỉ có 1 entry, content = new content

---

## TC-017-003: Session slot chỉ visible trong session đó

| **ID** | TC-017-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Slot A trong session-1, Slot B trong session-2

**Kết quả mong đợi:** `slot-list({sessionId: "session-1"})` → chỉ Slot A

---

## TC-017-004: Slot delete

| **ID** | TC-017-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Sau delete, slot read → not found

---

## TC-017-005: Slot name validation

| **ID** | TC-017-005 | **Priority** | 🟠 P1 | **Type** | Unit |
|---|---|---|---|---|---|

**Input:** `name = "my slot!"` (invalid chars)

**Kết quả mong đợi:** `{success: false, error: "invalid slot name"}`

---

---

# TC-018: Provider System

**Design ref:** [TD-018](../designs/TD-018-provider-system.md)

---

## TC-018-001: EMBEDDING_PROVIDER=none → zero vector, không crash

| **ID** | TC-018-001 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Trả về Float32Array với tất cả values = 0, length = 384

---

## TC-018-002: AGENTMEMORY_AUTO_COMPRESS=false → không gọi LLM

| **ID** | TC-018-002 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Spy confirm 0 HTTP calls đến Anthropic API

---

## TC-018-003: Embedding output là Float32Array với đúng dimension

| **ID** | TC-018-003 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Kết quả mong đợi:** `typeof output = "Float32Array"`, `output.length = 384`

---

## TC-018-004: Provider fail → fallback gracefully

| **ID** | TC-018-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Embedding API mock trả về 503

**Kết quả mong đợi:** Không crash, trả về zero vector, warning logged

---

---

# TC-019: Health & Diagnostics

**Design ref:** [TD-019](../designs/TD-019-health-diagnostics.md)

---

## TC-019-001: GET /health → 200 khi healthy

| **ID** | TC-019-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Status 200, `{status: "ok", uptime: <num>, version: "..."}`

---

## TC-019-002: GET /health → 503 khi KV không accessible

| **ID** | TC-019-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Status 503, `{status: "error"}`

---

## TC-019-003: Health response không expose sensitive info

| **ID** | TC-019-003 | **Priority** | 🔴 P0 | **Type** | Security |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Response không chứa API keys, secrets, file paths, stack traces

---

## TC-019-004: GET /status trả về metrics

| **ID** | TC-019-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Body có `totalSessions`, `totalObservations`, `totalMemories`, `graphNodeCount`

---

---

# TC-020: Security

**Design ref:** [TD-020](../designs/TD-020-security.md)

---

## TC-020-001: Valid Bearer token → 200

| **ID** | TC-020-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET = "test-key-123"`, request có `Authorization: Bearer test-key-123`

**Kết quả mong đợi:** Status 200

---

## TC-020-002: Wrong token → 401

| **ID** | TC-020-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Status 401, `{error: "unauthorized"}`

---

## TC-020-003: No secret set → local mode (no auth needed)

| **ID** | TC-020-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET` không set

**Kết quả mong đợi:** Request không có auth được chấp nhận (200)

---

## TC-020-004: Private data không xuất hiện trong KV sau observation

| **ID** | TC-020-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Hook với toolOutput chứa `sk-ant-api03-FAKEKEY`

**Kết quả mong đợi:** KV chỉ chứa `[REDACTED_SECRET]`, không có key thật

---

## TC-020-005: Path traversal sessionId bị từ chối

| **ID** | TC-020-005 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Input:** `sessionId = "../../../etc/passwd"`

**Kết quả mong đợi:** `{success: false, error: "invalid sessionId"}`

---

## TC-020-006: Timing-safe comparison

| **ID** | TC-020-006 | **Priority** | 🟠 P1 | **Type** | Unit |
|---|---|---|---|---|---|

**Verification:** Inspect source code — auth comparison dùng `crypto.timingSafeEqual()`, không dùng `===`

---

---

# TC-021: Performance

**Design ref:** [TD-021](../designs/TD-021-performance.md)

---

## TC-021-001: BM25 search p50 ≤ 14ms (1000 docs)

| **ID** | TC-021-001 | **Priority** | 🔴 P0 | **Type** | Performance |
|---|---|---|---|---|---|

**Setup:** 1000 docs indexed, 100 queries (warm-up 5 trước)

**Kết quả mong đợi:** p50 ≤ 14ms, p95 ≤ 100ms

---

## TC-021-002: Observe pipeline p50 ≤ 50ms (no LLM)

| **ID** | TC-021-002 | **Priority** | 🔴 P0 | **Type** | Performance |
|---|---|---|---|---|---|

**Setup:** MockKV, `EMBEDDING_PROVIDER=none`, 50 observe calls

**Kết quả mong đợi:** p50 ≤ 50ms, p95 ≤ 200ms

---

## TC-021-003: Recall p50 ≤ 50ms (1000 observations)

| **ID** | TC-021-003 | **Priority** | 🔴 P0 | **Type** | Performance |
|---|---|---|---|---|---|

**Setup:** 1000 observations, BM25-only, 50 recall queries

**Kết quả mong đợi:** p50 ≤ 50ms

---

## TC-021-004: Memory usage ổn định qua 1000 observe calls (no leak)

| **ID** | TC-021-004 | **Priority** | 🟠 P1 | **Type** | Performance |
|---|---|---|---|---|---|

**Các bước:**
1. Ghi heap trước
2. 1000 observe calls
3. Force GC
4. Ghi heap sau

**Kết quả mong đợi:** Heap growth < 50MB

---

---

# TC-022: Deployment

**Design ref:** [TD-022](../designs/TD-022-deployment.md)

---

## TC-022-001: `agentmemory --help` không crash, exit 0

| **ID** | TC-022-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Exit code 0, output có usage info

---

## TC-022-002: Server khởi động và respond /health với 200

| **ID** | TC-022-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Start server process
2. Wait ready signal
3. `GET /health`

**Kết quả mong đợi:** Status 200

---

## TC-022-003: SIGTERM → graceful shutdown

| **ID** | TC-022-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Exit code 0, không corrupt KV

---

## TC-022-004: iii-engine binary không tìm thấy → informative error

| **ID** | TC-022-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** iii-engine không có trong PATH

**Kết quả mong đợi:** Non-zero exit, error message mention "iii-engine"

---

## TC-022-005: Port conflict → informative error

| **ID** | TC-022-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Port đã bị chiếm

**Kết quả mong đợi:** Non-zero exit, stderr mention "EADDRINUSE" hoặc "port in use"

---

---

# TC-023: Export & Import

**Design ref:** [TD-023](../designs/TD-023-export-import.md)

---

## TC-023-001: Export toàn bộ KV state sang JSON

| **ID** | TC-023-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 3 sessions, 30 obs, 10 memories, 5 graph nodes

**Kết quả mong đợi:**
- JSON hợp lệ, parseable
- Có `version`, `exportedAt`, `data`
- Data counts match

---

## TC-023-002: Import từ valid export JSON

| **ID** | TC-023-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Clean KV + export JSON từ TC-023-001

**Kết quả mong đợi:**
- 3 sessions, 30 obs, 10 memories được restore
- `{success: true, importedSessions: 3, ...}`

---

## TC-023-003: Import idempotent — gọi 2 lần không tạo duplicates

| **ID** | TC-023-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi (lần 2):** `{importedCount: 0, skippedCount: N}`

---

## TC-023-004: Import corrupt JSON → fail gracefully, không partial import

| **ID** | TC-023-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Input:** Malformed JSON string

**Kết quả mong đợi:** `{success: false, error: "invalid JSON"}`, KV không thay đổi

---

## TC-023-005: Migration — tự động backfill missing fields

| **ID** | TC-023-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** KV có memories ở format cũ (thiếu `isLatest` field)

**Kết quả mong đợi:**
- Sau `mem::migrate`: tất cả memories có `isLatest = true`
- Không có data loss

---

## TC-023-006: Export không chứa expired memories

| **ID** | TC-023-006 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Memory A (expired yesterday) + Memory B (no expiry)

**Kết quả mong đợi:** Export chứa B, không chứa A

---

## TC-023-007: Import không overwrite newer data

| **ID** | TC-023-007 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Current KV: Memory M1 với `updatedAt = 2026-06-10T14:00:00Z`
- Import data: Memory M1 với `updatedAt = 2026-06-09T14:00:00Z` (cũ hơn)

**Kết quả mong đợi:** Existing M1 (newer) được giữ nguyên, import M1 bị skip
