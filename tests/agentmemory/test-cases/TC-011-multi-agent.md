# TC-011: Multi-Agent — Test Cases

**Test Design tham chiếu:** [TD-011](../designs/TD-011-multi-agent.md)  
**Requirements tham chiếu:** [TR-011](../requirements/TR-011-multi-agent.md)  
**Module:** Lease System, Signal Bus, agentId, agentScope  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-011-001: Acquire lease thành công khi resource free

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-011-AGT-001 |

**Điều kiện tiên quyết:** KV `mem:leases` không có entry cho resource `"shared-state-123"`

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `resource` | `shared-state-123` |
| `agentId` | `agent-A` |
| `ttl` | `30` (giây) |

**Các bước thực hiện:**
1. Gọi `acquire-lease({resource: "shared-state-123", agentId: "agent-A", ttl: 30})`
2. Kiểm tra response
3. Đọc KV `mem:leases["shared-state-123"]`

**Kết quả mong đợi:**
- `{success: true, leaseId: "..."}` — leaseId là string không rỗng
- KV `mem:leases["shared-state-123"]` tồn tại
- `lease.holder = "agent-A"`
- `lease.expiresAt` là timestamp trong tương lai (khoảng T + 30s)

**Tiêu chí Pass:** `success = true` và `leaseId` có giá trị.

---

## TC-011-002: Acquire bị từ chối khi resource đã có lease

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-011-AGT-002 |

**Điều kiện tiên quyết:** Agent A đang giữ lease cho `"shared-state-123"` (từ TC-011-001)

**Các bước thực hiện:**
1. Agent B gọi `acquire-lease({resource: "shared-state-123", agentId: "agent-B", ttl: 30})`
2. Kiểm tra response

**Kết quả mong đợi:**
- `{success: false, error: "...locked..."}` (hoặc tương đương)
- KV `mem:leases["shared-state-123"]` vẫn có `holder = "agent-A"` (không bị overwrite)

---

## TC-011-003: Lease hết TTL → resource released tự động

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-003 |
| **Loại** | Unit (fake timers) |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-011-AGT-003 |

**Điều kiện tiên quyết:** Fake timers được enable

**Các bước thực hiện:**
1. Agent A acquire lease với `ttl = 5` giây
2. Advance fake time by **4 giây** → Agent B cố acquire → phải bị từ chối
3. Advance fake time thêm **2 giây** (total = 6s, past TTL)
4. Agent B cố acquire lại

**Kết quả mong đợi:**
- Bước 2: Agent B nhận `success: false`
- Bước 4: Agent B nhận `success: true` (TTL đã expire, resource available)

---

## TC-011-004: Release lease trước TTL → resource available ngay

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-011-AGT-004 |

**Các bước thực hiện:**
1. Agent A acquire lease → nhận `leaseId`
2. Agent A gọi `release-lease({leaseId})`
3. Verify response: `{success: true}`
4. Agent B cố acquire ngay lập tức

**Kết quả mong đợi:** Agent B acquire thành công ngay (không cần chờ TTL)

---

## TC-011-005: Signal delivery đến đúng teamId

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-011-AGT-007 |

**Setup:**
- Team `"team-A"`: Agents A và B
- Team `"team-B"`: Agent C

**Dữ liệu đầu vào (signal payload từ Agent A):**

| Trường | Giá trị |
|---|---|
| `type` | `file-changed` |
| `file` | `auth.ts` |
| `teamId` | `team-A` |

**Các bước thực hiện:**
1. Agent B subscribe `{signalType: "file-changed", agentId: "agent-B"}`
2. Agent C subscribe `{signalType: "file-changed", agentId: "agent-C"}`
3. Agent A publish signal với `teamId = "team-A"`
4. Kiểm tra signals pending cho B và C

**Kết quả mong đợi:**
- Agent B nhận được signal với `file = "auth.ts"`
- Agent C KHÔNG nhận được signal

---

## TC-011-006: Observation có agentId từ AGENT_ID env var

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-011-AGT-010 |

**Điều kiện tiên quyết:** `AGENT_ID = cursor-agent-1` trong môi trường

**Dữ liệu đầu vào:**
- Hook `post_tool_use` không có `agentId` trong payload

**Kết quả mong đợi:** `observation.agentId = "cursor-agent-1"` trong KV

---

## TC-011-007: agentScope="own" — chỉ trả về obs của agent đó

| Trường | Giá trị |
|---|---|
| **ID** | TC-011-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-011-AGT-011 |

**Setup:**
- 3 obs từ Agent A (`agentId = "agent-A"`)
- 2 obs từ Agent B (`agentId = "agent-B"`)
- Tất cả trong cùng session `sess_multi`

**Dữ liệu đầu vào:**
- `recall({sessionId: "sess_multi", agentScope: "own", agentId: "agent-A"})`

**Kết quả mong đợi:**
- Chỉ 3 obs của agent-A trong results
- Obs của agent-B không xuất hiện

---

## Tổng kết TC-011

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-011-001 | Acquire lease free | 🔴 P0 | Integration |
| TC-011-002 | Acquire bị từ chối | 🔴 P0 | Integration |
| TC-011-003 | TTL expire | 🔴 P0 | Unit |
| TC-011-004 | Release early | 🟠 P1 | Integration |
| TC-011-005 | Signal delivery | 🟠 P1 | Integration |
| TC-011-006 | agentId from env | 🔴 P0 | Integration |
| TC-011-007 | agentScope=own | 🟠 P1 | Integration |
