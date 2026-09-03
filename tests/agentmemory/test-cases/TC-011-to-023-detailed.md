# TC-011 đến TC-023: Module Test Cases

**Ngày:** 2026-06-11  
Đây là các test cases chi tiết cho các module còn lại.

---

# TC-011: Multi-Agent

**Design ref:** [TD-011](../designs/TD-011-multi-agent.md)

---

## TC-011-001: Acquire lease thành công khi resource free

| **ID** | TC-011-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** resource = `"shared-state-123"`, agentId = `"agent-A"`, TTL = 30s

**Các bước:**
1. Gọi `acquire-lease({resource, agentId, ttl: 30})`
2. Kiểm tra response
3. Đọc KV `mem:leases`

**Kết quả mong đợi:**
- `{success: true, leaseId: "..."}` (leaseId là string)
- KV `mem:leases["shared-state-123"]` tồn tại với `holder = "agent-A"`

---

## TC-011-002: Acquire bị từ chối khi resource đã có lease

| **ID** | TC-011-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Agent A đang giữ lease cho `"shared-state-123"`

**Các bước:**
1. Agent B cố gọi `acquire-lease({resource: "shared-state-123", agentId: "agent-B"})`
2. Kiểm tra response

**Kết quả mong đợi:**
- `{success: false, error: "...locked..."}`
- Không tạo thêm lease

---

## TC-011-003: Lease hết TTL → resource released

| **ID** | TC-011-003 | **Priority** | 🔴 P0 | **Type** | Unit (fake timers) |
|---|---|---|---|---|---|

**Điều kiện tiên quyết:** Fake timers được enable

**Các bước:**
1. Agent A acquire lease với TTL = 5s
2. Advance time by **4 minutes 59 seconds** → Agent B acquire → bị từ chối
3. Advance time thêm 2s (past TTL)
4. Agent B acquire lại

**Kết quả mong đợi:** Bước 4: Agent B acquire thành công (TTL đã expire)

---

## TC-011-004: Release lease trước TTL → resource available ngay

| **ID** | TC-011-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Agent A acquire lease
2. Agent A gọi `release-lease({leaseId})`
3. Agent B acquire

**Kết quả mong đợi:** Agent B acquire thành công ngay (không cần chờ TTL)

---

## TC-011-005: Signal delivery đến cùng teamId

| **ID** | TC-011-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Team "team-A": Agents A và B
- Team "team-B": Agent C

**Các bước:**
1. Agent B subscribe signal type `"file-changed"`
2. Agent C subscribe signal type `"file-changed"`
3. Agent A publish signal `{type: "file-changed", file: "auth.ts", teamId: "team-A"}`
4. Kiểm tra signals nhận được

**Kết quả mong đợi:**
- Agent B nhận signal với `file = "auth.ts"`
- Agent C KHÔNG nhận signal

---

## TC-011-006: Observation có agentId từ AGENT_ID env var

| **ID** | TC-011-006 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENT_ID = cursor-agent-1`

**Kết quả mong đợi:** `observation.agentId = "cursor-agent-1"`

---

## TC-011-007: agentScope="own" chỉ trả về obs của agent đó

| **ID** | TC-011-007 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Obs từ Agent A (agentId=`"agent-A"`)
- Obs từ Agent B (agentId=`"agent-B"`)

**Kết quả mong đợi:** `recall({agentScope: "own", agentId: "agent-A"})` → chỉ obs của agent-A

---

---

# TC-012: Orchestration (Actions, Routines, Sketches)

**Design ref:** [TD-012](../designs/TD-012-orchestration.md)

---

## TC-012-001: Action được tạo với status "pending"

| **ID** | TC-012-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** title = `"Refactor auth module"`, steps = `["Analyze", "Write tests", "Implement"]`

**Kết quả mong đợi:**
- `action.status = "pending"`
- `action.steps.length = 3`
- `action.createdAt` — ISO timestamp
- Action lưu trong KV `mem:actions`

---

## TC-012-002: State transition pending → in-progress → completed

| **ID** | TC-012-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Create action → `status = "pending"`
2. `action-update({id, status: "in-progress"})` → verify `status = "in-progress"`
3. `action-update({id, status: "completed"})` → verify `status = "completed"`

**Kết quả mong đợi:** Mỗi transition accepted

---

## TC-012-003: Invalid state transition bị từ chối

| **ID** | TC-012-003 | **Priority** | 🟠 P1 | **Type** | Unit |
|---|---|---|---|---|---|

**Setup:** Action ở `"completed"`

**Kết quả mong đợi:** Transition sang `"pending"` (backward) bị từ chối với error

---

## TC-012-004: Sketch write/read/append/clear lifecycle

| **ID** | TC-012-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước và kết quả mong đợi:**

| Bước | Action | Expected content |
|---|---|---|
| 1 | `sketch-write("Initial note")` | `"Initial note"` |
| 2 | Verify read | `"Initial note"` |
| 3 | `sketch-append("Additional note")` | `"Initial note\nAdditional note"` |
| 4 | `sketch-clear` | `""` |

---

## TC-012-005: Routine được tạo với schedule và enabled=true

| **ID** | TC-012-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** schedule = `"0 */6 * * *"` (every 6 hours)

**Kết quả mong đợi:**
- `routine.schedule = "0 */6 * * *"`
- `routine.enabled = true`
- Routine lưu trong `mem:routines`

---

---

# TC-013: Governance & Audit

**Design ref:** [TD-013](../designs/TD-013-governance-audit.md)

---

## TC-013-001: mem::remember tạo audit record với đúng fields

| **ID** | TC-013-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- KV `mem:audit` có entry với:
  - `operation = "remember"`
  - `memoryId` (string)
  - `timestamp` (ISO)
  - `sessionId` (string)

---

## TC-013-002: mem::forget tạo audit record

| **ID** | TC-013-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Audit record: `operation = "forget"`, `memoryId`, `timestamp`

---

## TC-013-003: Audit records không bị xóa khi memory bị forget

| **ID** | TC-013-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Create memory M → audit record R tạo ra
2. `forget(M)` → M bị xóa
3. Kiểm tra R trong `mem:audit`

**Kết quả mong đợi:** R vẫn tồn tại (immutable audit trail)

---

## TC-013-004: Retention sweep xóa expired memories, giữ lại active memories

| **ID** | TC-013-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Memory A: `forgetAfter = [hôm qua]` (expired)
- Memory B: `forgetAfter = [1 tuần sau]` (active)
- Memory C: không có `forgetAfter`

**Các bước:**
1. Run retention sweep
2. Kiểm tra KV

**Kết quả mong đợi:**
- Memory A: bị xóa
- Memory B: còn tồn tại
- Memory C: còn tồn tại

---

## TC-013-005: API key hợp lệ → HTTP 200

| **ID** | TC-013-005 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET = "valid-key-16chars"`

**Request:** `GET /status` với `Authorization: Bearer valid-key-16chars`

**Kết quả mong đợi:** HTTP 200

---

## TC-013-006: API key sai → HTTP 401

| **ID** | TC-013-006 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request:** `Authorization: Bearer wrong-key`

**Kết quả mong đợi:** HTTP 401, body `{error: "unauthorized"}` hoặc tương đương

---

## TC-013-007: Không có AGENTMEMORY_SECRET → local mode (không cần auth)

| **ID** | TC-013-007 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET` không set

**Kết quả mong đợi:** Request không có Authorization header được chấp nhận (200)

---

---

# TC-014: MCP Server

**Design ref:** [TD-014](../designs/TD-014-mcp-server.md)

---

## TC-014-001: tools/list trả về đầy đủ tools

| **ID** | TC-014-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Khởi động MCP server
2. Gửi JSONRPC request `{"method": "tools/list", "id": 1}`
3. Kiểm tra response

**Kết quả mong đợi:** Response chứa tất cả tools sau:
`mem_observe`, `mem_recall`, `mem_remember`, `mem_forget`, `mem_search`, `mem_status`, `mem_profile`

---

## TC-014-002: Mỗi tool có JSON Schema hợp lệ

| **ID** | TC-014-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi — mỗi tool có:**
- `name`: string
- `description`: string không rỗng
- `inputSchema.type = "object"`
- `inputSchema.properties`: object không rỗng

---

## TC-014-003: mem_recall trả về JSONRPC 2.0 response

| **ID** | TC-014-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** `{query: "auth", sessionId: "sess_test"}`

**Kết quả mong đợi:**
- Response có `"jsonrpc": "2.0"`
- `result.content[0].type = "text"`
- `result.content[0].text` không rỗng

---

## TC-014-004: mem_remember lưu memory và trả về confirmation

| **ID** | TC-014-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** `{content: "Auth uses JWT", type: "architecture"}`

**Kết quả mong đợi:**
- `result.content[0].text` chứa confirmation
- Memory được lưu trong KV (verify bằng mem_recall)

---

## TC-014-005: Tool với invalid args → JSONRPC error

| **ID** | TC-014-005 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** mem_recall với payload `{}` (thiếu sessionId)

**Kết quả mong đợi:**
- Response có `"error"` field
- `"result"` field không có

---

## TC-014-006: Concurrent tool invocations không conflict

| **ID** | TC-014-006 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 5 concurrent mem_recall requests

**Kết quả mong đợi:** Tất cả 5 responses nhận được, không có error do concurrency

---

---

# TC-015: REST API

**Design ref:** [TD-015](../designs/TD-015-rest-api.md)

---

## TC-015-001: GET /sessions → danh sách sessions

| **ID** | TC-015-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 3 sessions trong KV

**Kết quả mong đợi:**
- HTTP 200
- Body: `{sessions: [...]}` với 3 items
- Mỗi item có `id`, `project`, `status`, `observationCount`

---

## TC-015-002: GET /sessions/:id/observations → obs của session

| **ID** | TC-015-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Session `sess_abc` với 5 observations

**Kết quả mong đợi:**
- HTTP 200
- `observations.length = 5`
- Mỗi obs có `id`, `timestamp`, `type`, `title`

---

## TC-015-003: GET /sessions/:id không tồn tại → 404

| **ID** | TC-015-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** HTTP 404, body có `error` field

---

## TC-015-004: GET /search?q=auth → sorted results

| **ID** | TC-015-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 5 observations có "auth" trong content

**Kết quả mong đợi:**
- HTTP 200
- `results` array sorted by `combinedScore` descending
- Mỗi result có `observation`, `bm25Score`, `combinedScore`

---

## TC-015-005: GET /search?limit=5 giới hạn kết quả

| **ID** | TC-015-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 20 matching observations

**Kết quả mong đợi:** `results.length = 5`

---

## TC-015-006: GET /memories → chỉ isLatest=true

| **ID** | TC-015-006 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** M1 (isLatest=false), M2 (isLatest=true)

**Kết quả mong đợi:** Response chứa M2, không chứa M1

---

## TC-015-007: POST /memories → tạo memory mới

| **ID** | TC-015-007 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request body:** `{content: "Auth uses JWT", type: "architecture"}`

**Kết quả mong đợi:**
- HTTP 201
- Body chứa `version = 1`, `isLatest = true`

---

## TC-015-008: DELETE /memories/:id → xóa

| **ID** | TC-015-008 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** HTTP 200, `{success: true, deleted: 1}`, subsequent GET → 404

---

## TC-015-009: POST /memories thiếu content → 422

| **ID** | TC-015-009 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request:** body `{type: "fact"}` (thiếu content)

**Kết quả mong đợi:** HTTP 422

---

## TC-015-010: GET /health → 200 khi healthy

| **ID** | TC-015-010 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** HTTP 200, `{status: "ok"}`

---

## TC-015-011: Internal error → 500 không expose stack trace

| **ID** | TC-015-011 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** KV inject lỗi

**Kết quả mong đợi:** HTTP 500, body `{error: "internal server error"}` (không có stack)

---

---

# TC-016: Session Replay

**Design ref:** [TD-016](../designs/TD-016-session-replay.md)

---

## TC-016-001: Import JSONL với valid transcript

| **ID** | TC-016-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** JSONL file 10 lines hợp lệ

**Kết quả mong đợi:**
- 10 observations trong KV
- Session `source = "replay"`
- Response: `{success: true, importedCount: 10}`

---

## TC-016-002: Malformed JSONL — skip invalid, count valid

| **ID** | TC-016-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** 8 valid + 2 invalid JSON lines

**Kết quả mong đợi:**
- `importedCount = 8`
- `skippedCount = 2` (hoặc warning về 2 lines)

---

## TC-016-003: Import idempotent — không duplicate khi gọi 2 lần

| **ID** | TC-016-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi (lần 2):**
- `importedCount = 0`, `skippedCount = 10` (tất cả đã tồn tại)

---

## TC-016-004: Observation data được preserved đầy đủ

| **ID** | TC-016-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** `toolName`, `toolInput`, `toolOutput`, `agentId` đều preserved sau import

---

## TC-016-005: Viewer server trả về session list

| **ID** | TC-016-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 3 sessions trong KV, Viewer server running

**Kết quả mong đợi:** `GET /api/sessions` → HTTP 200, JSON array với 3 sessions

---

---

# TC-017: Memory Slots

**Design ref:** [TD-017](../designs/TD-017-memory-slots.md)

---

## TC-017-001: Slot write và read thành công

| **ID** | TC-017-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** name = `"current-task"`, content = `"Implementing auth middleware"`

**Kết quả mong đợi:**
- Write: `{success: true}`
- Read: `{success: true, slot: {name: "current-task", content: "Implementing auth middleware", createdAt, updatedAt}}`

---

## TC-017-002: Slot overwrite khi cùng name

| **ID** | TC-017-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Write `{name: "goal", content: "Old content"}`
2. Write `{name: "goal", content: "New content"}`
3. Read `{name: "goal"}`

**Kết quả mong đợi:**
- Chỉ 1 slot với name "goal"
- `slot.content = "New content"`
- `slot.updatedAt > slot.createdAt`

---

## TC-017-003: Read slot không tồn tại → null/not found

| **ID** | TC-017-003 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Kết quả mong đợi:** `{success: true, slot: null}` hoặc `{success: false, error: "not found"}`

---

## TC-017-004: Delete slot

| **ID** | TC-017-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** `{success: true, deleted: 1}`, read sau delete → not found

---

## TC-017-005: Session slot isolation

| **ID** | TC-017-005 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Slot A trong session-1, Slot B trong session-2

**Kết quả mong đợi:** `slot-list({sessionId: "session-1"})` chỉ trả về Slot A

---

## TC-017-006: Slot name validation — invalid chars

| **ID** | TC-017-006 | **Priority** | 🟠 P1 | **Type** | Unit |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** name = `"my slot!"` (space và !)

**Kết quả mong đợi:** `{success: false, error: "invalid slot name"}`

---

## TC-017-007: Empty content bị từ chối

| **ID** | TC-017-007 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** content = `""`

**Kết quả mong đợi:** `{success: false, error: "content is required"}`

---

---

# TC-018: Provider System

**Design ref:** [TD-018](../designs/TD-018-provider-system.md)

---

## TC-018-001: EMBEDDING_PROVIDER=none → zero vector không crash

| **ID** | TC-018-001 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Setup:** `EMBEDDING_PROVIDER = none`

**Kết quả mong đợi:**
- `embed("some text")` trả về Float32Array
- Tất cả values = 0
- `length = 384` (default dim)
- Không throw exception

---

## TC-018-002: AGENTMEMORY_AUTO_COMPRESS=false → 0 LLM calls

| **ID** | TC-018-002 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_AUTO_COMPRESS = false`

**Các bước:**
1. Observe 5 hooks
2. Đếm số HTTP calls đến Anthropic API

**Kết quả mong đợi:** 0 calls đến LLM API

---

## TC-018-003: Embedding output là Float32Array đúng dimension

| **ID** | TC-018-003 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- `typeof output.constructor.name = "Float32Array"`
- `output.length = 384` (provider default)

---

## TC-018-004: Embedding provider fail → fallback gracefully

| **ID** | TC-018-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Mock API trả về 503

**Kết quả mong đợi:** Không crash, fallback zero vector, warning logged

---

## TC-018-005: Unknown EMBEDDING_PROVIDER → informative error khi startup

| **ID** | TC-018-005 | **Priority** | 🟡 P2 | **Type** | Unit |
|---|---|---|---|---|---|

**Setup:** `EMBEDDING_PROVIDER = unknown_xyz`

**Kết quả mong đợi:** Throw Error với message rõ ràng về provider không hỗ trợ

---

---

# TC-019: Health & Diagnostics

**Design ref:** [TD-019](../designs/TD-019-health-diagnostics.md)

---

## TC-019-001: GET /health → 200 khi healthy

| **ID** | TC-019-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Server running, KV accessible

**Kết quả mong đợi:**
- HTTP 200
- `{status: "ok", uptime: <number>, version: "..."}`

---

## TC-019-002: GET /health → 503 khi KV không accessible

| **ID** | TC-019-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** KV inject lỗi (simulate disconnect)

**Kết quả mong đợi:** HTTP 503, `{status: "error", error: "..."}`

---

## TC-019-003: Health response không expose sensitive info

| **ID** | TC-019-003 | **Priority** | 🔴 P0 | **Type** | Security |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Response KHÔNG chứa:
- API keys, secrets
- File system paths
- Stack traces
- Environment variable values

---

## TC-019-004: GET /status trả về aggregate metrics

| **ID** | TC-019-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 50 sessions, 500 obs, 20 memories

**Kết quả mong đợi:**
- HTTP 200
- Body có: `totalSessions`, `totalObservations`, `totalMemories`, `graphNodeCount`, `heapUsedMB`

---

## TC-019-005: mem::diagnose báo cáo dimension mismatch

| **ID** | TC-019-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Vector index có mixed dimensions (384 và 768)

**Kết quả mong đợi:**
- `{vectorIndex: {mismatches: [{obsId: "...", dim: 768}], consistent: false}}`

---

---

# TC-020: Security

**Design ref:** [TD-020](../designs/TD-020-security.md)

---

## TC-020-001: Valid Bearer token → HTTP 200

| **ID** | TC-020-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET = "valid-secret-key-16chars"`

**Request:** `GET /status`, `Authorization: Bearer valid-secret-key-16chars`

**Kết quả mong đợi:** HTTP 200

---

## TC-020-002: Wrong token → HTTP 401

| **ID** | TC-020-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Request:** `Authorization: Bearer wrong-key`

**Kết quả mong đợi:** HTTP 401, `{error: "unauthorized"}`

---

## TC-020-003: No secret set → local mode (no auth)

| **ID** | TC-020-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `AGENTMEMORY_SECRET` không set

**Kết quả mong đợi:** Request không có auth được chấp nhận (200)

---

## TC-020-004: Private data không xuất hiện trong KV sau observation

| **ID** | TC-020-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** Hook với toolOutput = `"ANTHROPIC_API_KEY=sk-ant-api03-FAKEKEY_FOR_TEST"`

**Kết quả mong đợi:**
- KV observation KHÔNG chứa `sk-ant-api03-`
- Chỉ có `[REDACTED_SECRET]` tại vị trí đó

---

## TC-020-005: Private data không xuất hiện trong recall response

| **ID** | TC-020-005 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Observation đã được stored với redacted content

**Kết quả mong đợi:** Recall response không chứa un-redacted secrets

---

## TC-020-006: Path traversal sessionId bị từ chối

| **ID** | TC-020-006 | **Priority** | 🔴 P0 | **Type** | Unit |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** `sessionId = "../../../etc/passwd"`

**Kết quả mong đợi:**
- `{success: false, error: "invalid sessionId"}`
- Không tạo KV entry với path-like key

---

## TC-020-007: Timing-safe comparison trong auth

| **ID** | TC-020-007 | **Priority** | 🟠 P1 | **Type** | Unit |
|---|---|---|---|---|---|

**Verification method:** Code review — confirm auth comparison dùng `crypto.timingSafeEqual()` hoặc tương đương, KHÔNG dùng `===` để compare secrets

**Kết quả mong đợi:** Source code audit xác nhận timing-safe implementation

---

---

# TC-021: Performance

**Design ref:** [TD-021](../designs/TD-021-performance.md)

---

## TC-021-001: BM25 search p50 ≤ 14ms (1000 docs)

| **ID** | TC-021-001 | **Priority** | 🔴 P0 | **Type** | Performance |
|---|---|---|---|---|---|

**Setup:**
- Index: 1000 docs (10-20 concepts mỗi doc, diverse vocabulary)
- 100 queries với 10 diverse terms, mỗi term x 10 iterations
- Warm-up: 5 queries trước khi bắt đầu đo
- Isolated environment (không background processes)

**Các bước:**
1. Index 1000 docs
2. Warm-up 5 queries
3. Run 100 queries, ghi latency mỗi query
4. Tính p50 và p95

**Kết quả mong đợi:**
- `p50 ≤ 14ms`
- `p95 ≤ 100ms`
- `errors = 0`

---

## TC-021-002: Observe pipeline p50 ≤ 50ms (no LLM/embedding)

| **ID** | TC-021-002 | **Priority** | 🔴 P0 | **Type** | Performance |
|---|---|---|---|---|---|

**Setup:**
- MockKV (in-memory)
- `EMBEDDING_PROVIDER = none`
- `AGENTMEMORY_AUTO_COMPRESS = false`
- 50 observe calls, warm-up 5 trước

**Kết quả mong đợi:** `p50 ≤ 50ms`, `p95 ≤ 200ms`

---

## TC-021-003: Recall p50 ≤ 50ms (1000 obs, BM25-only)

| **ID** | TC-021-003 | **Priority** | 🔴 P0 | **Type** | Performance |
|---|---|---|---|---|---|

**Setup:** 1000 obs indexed, 50 recall queries

**Kết quả mong đợi:** `p50 ≤ 50ms`

---

## TC-021-004: Memory usage ổn định sau 1000 observe calls

| **ID** | TC-021-004 | **Priority** | 🟠 P1 | **Type** | Performance |
|---|---|---|---|---|---|

**Các bước:**
1. Ghi heap memory (baseline)
2. Run 1000 observe calls
3. Force GC
4. Ghi heap memory (after)

**Kết quả mong đợi:** `heapAfter - heapBefore < 50MB`

---

## TC-021-005: BM25 index throughput 1000 docs < 1000ms

| **ID** | TC-021-005 | **Priority** | 🟠 P1 | **Type** | Performance |
|---|---|---|---|---|---|

**Các bước:**
1. Chuẩn bị 1000 docs
2. Bắt đầu timer
3. Add tất cả 1000 docs
4. Dừng timer

**Kết quả mong đợi:** `totalTime < 1000ms`

---

---

# TC-022: Deployment

**Design ref:** [TD-022](../designs/TD-022-deployment.md)

---

## TC-022-001: `agentmemory --help` exit 0 với usage info

| **ID** | TC-022-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Chạy `agentmemory --help`
2. Kiểm tra exit code và stdout

**Kết quả mong đợi:**
- Exit code = 0
- Stdout chứa usage information (usage, options, commands)

---

## TC-022-002: Server khởi động và respond /health

| **ID** | TC-022-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Start server process với mặc định config
2. Wait for ready signal (hoặc timeout 5s)
3. `GET /health`
4. Kill process sau test

**Kết quả mong đợi:** HTTP 200 từ /health

---

## TC-022-003: SIGTERM → graceful shutdown, exit 0

| **ID** | TC-022-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Start server
2. Send `SIGTERM` signal
3. Wait for process exit
4. Kiểm tra exit code

**Kết quả mong đợi:** Exit code = 0, không corrupt KV

---

## TC-022-004: iii-engine không tìm thấy → informative error

| **ID** | TC-022-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** iii-engine binary không có trong PATH (hoặc wrong version)

**Kết quả mong đợi:**
- Non-zero exit code
- Stderr message đề cập đến "iii-engine" (hướng dẫn cài đặt)

---

## TC-022-005: Port conflict → informative error

| **ID** | TC-022-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Start một process chiếm port trước, sau đó start agentmemory với cùng port

**Kết quả mong đợi:**
- Non-zero exit code
- Stderr có "EADDRINUSE" hoặc "port in use"

---

---

# TC-023: Export & Import

**Design ref:** [TD-023](../designs/TD-023-export-import.md)

---

## TC-023-001: Export toàn bộ KV state sang JSON

| **ID** | TC-023-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 3 sessions, 30 obs, 10 memories, 5 graph nodes

**Các bước:**
1. Gọi `mem::export({format: "json"})`
2. Parse JSON output
3. Kiểm tra structure và data counts

**Kết quả mong đợi:**
- JSON valid (parseable)
- `json.version` tồn tại
- `json.exportedAt` là ISO timestamp
- `json.data.sessions.length = 3`
- `json.data.observations.length = 30`
- `json.data.memories.length = 10`

---

## TC-023-002: Import từ valid export JSON → restore đầy đủ

| **ID** | TC-023-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Clean KV, sử dụng export JSON từ TC-023-001

**Kết quả mong đợi:**
- `{success: true, importedSessions: 3, importedObservations: 30, importedMemories: 10}`
- Data tồn tại trong KV sau import

---

## TC-023-003: Import idempotent — gọi 2 lần không tạo duplicates

| **ID** | TC-023-003 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Import lần 1 → KV có 3 sessions, 30 obs
2. Import lần 2 với cùng data
3. Đếm entries trong KV

**Kết quả mong đợi:**
- KV vẫn có 3 sessions, 30 obs (không tăng)
- Response lần 2: `importedCount = 0, skippedCount = N`

---

## TC-023-004: Import corrupt JSON → fail gracefully

| **ID** | TC-023-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** JSON string bị truncate `'{"sessions": [{"id": "se'` (malformed)

**Kết quả mong đợi:**
- `{success: false, error: "invalid JSON"}`
- KV không thay đổi (không partial import)

---

## TC-023-005: Export không chứa expired memories

| **ID** | TC-023-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- Memory A: `forgetAfter = [hôm qua]` (expired)
- Memory B: không có `forgetAfter`

**Kết quả mong đợi:**
- Export JSON có Memory B
- Export JSON KHÔNG có Memory A

---

## TC-023-006: Import không overwrite data mới hơn

| **ID** | TC-023-006 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- KV: Memory M1 với `updatedAt = 2026-06-10T14:00:00Z` (mới hơn)
- Import data: Memory M1 với `updatedAt = 2026-06-09T14:00:00Z` (cũ hơn)

**Kết quả mong đợi:** M1 trong KV giữ nguyên `updatedAt = 2026-06-10...` (import bị skip)

---

## TC-023-007: Migration backfill missing fields

| **ID** | TC-023-007 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** KV có 5 memories ở format cũ (thiếu `isLatest` field)

**Các bước:**
1. Gọi `mem::migrate`
2. Đọc tất cả memories từ KV

**Kết quả mong đợi:**
- Tất cả 5 memories có `isLatest = true`
- Không mất data khác

---

## TC-023-008: Migration idempotent — chạy 2 lần an toàn

| **ID** | TC-023-008 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Sau lần 2, data không bị corrupt, không có duplicate fields
