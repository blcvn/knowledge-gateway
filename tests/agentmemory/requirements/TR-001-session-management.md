# TR-001: Session Management Test Requirements

**Module:** Session Lifecycle  
**Nguồn:** SRS §3.1 (FR-SESSION-001..004), URD §3.1, Architecture §4.1, TDD §2.1  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho vòng đời Session: khởi tạo, kết thúc, giới hạn observation, và abandonment.

**Các function liên quan:**
- `mem::observe` (xử lý `session_start` hook)
- `mem::summarize` (trigger khi session kết thúc)
- REST: `GET /sessions`, `GET /sessions/{id}`, `POST /session/start`

---

## TR-001-SES-001 — Tạo session từ hook `session_start`
🔴 P0 | `[INT]` | **FR-SESSION-001**

**Given:** Không có session nào đang tồn tại với `sessionId`  
**When:** Hook `session_start` được gửi với payload:
```json
{
  "hookType": "session_start",
  "sessionId": "sess_abc123",
  "project": "my-project",
  "cwd": "/Users/dev/my-project",
  "timestamp": "2026-06-10T14:00:00Z",
  "data": { "model": "claude-opus-4" }
}
```
**Then:**
- Session được tạo trong KV với `status: "active"`
- `id` khớp với `sess_abc123`
- `project` = "my-project"
- `cwd` = "/Users/dev/my-project"
- `startedAt` = "2026-06-10T14:00:00Z"
- `observationCount` = 1
- `model` = "claude-opus-4"

**Traceability:** FR-SESSION-001, SRS §6.1

---

## TR-001-SES-002 — ID session là unique UUID
🟠 P1 | `[UNIT]` | **FR-SESSION-001**

**Given:** 1000 session được tạo đồng thời  
**When:** Mỗi session có `sessionId` được generate  
**Then:**
- Không có 2 session nào có cùng ID
- Format ID: `sess_<nanoid>` (URL-safe, 21 ký tự)

**Traceability:** FR-SESSION-001, TDD §10.1

---

## TR-001-SES-003 — Session tracking fields đầy đủ
🟠 P1 | `[INT]` | **FR-SESSION-001**

**Given:** Session vừa được tạo qua `session_start`  
**When:** `GET /sessions/{id}` được gọi  
**Then:** Response chứa đầy đủ fields:
```typescript
{
  id: string,          // UUID
  project: string,     // required
  cwd: string,         // required
  startedAt: string,   // ISO timestamp
  status: "active",
  observationCount: number,
  model?: string,
  tags?: string[],
  firstPrompt?: string,
  agentId?: string
}
```

**Traceability:** FR-SESSION-001, SRS §6.1

---

## TR-001-SES-004 — Session kết thúc qua hook `stop`
🔴 P0 | `[INT]` | **FR-SESSION-002**

**Given:** Session `sess_abc123` đang ở status `active`  
**When:** Hook `stop` được gửi cho session đó  
**Then:**
- `status` thay đổi thành `"completed"`
- `endedAt` được set với ISO timestamp
- Session summary được trigger tự động (async)

**Traceability:** FR-SESSION-002, SRS §3.1

---

## TR-001-SES-005 — Session kết thúc qua hook `session_end`
🔴 P0 | `[INT]` | **FR-SESSION-002**

**Given:** Session `sess_abc123` đang ở status `active`  
**When:** Hook `session_end` được gửi  
**Then:**
- `status` thay đổi thành `"completed"`
- Behavior giống TR-001-SES-004

**Traceability:** FR-SESSION-002

---

## TR-001-SES-006 — `firstPrompt` được capture
🟠 P1 | `[INT]` | **FR-SESSION-001**

**Given:** Session mới vừa tạo  
**When:** Hook `prompt_submit` đầu tiên với `userPrompt = "Build me an auth system"`  
**Then:**
- `session.firstPrompt` = "Build me an auth system"
- Các `prompt_submit` tiếp theo KHÔNG ghi đè `firstPrompt`

**Traceability:** FR-SESSION-001, SRS §6.1

---

## TR-001-SES-007 — Giới hạn observations per session
🔴 P0 | `[INT]` | **FR-SESSION-003**

**Given:** `MAX_OBS_PER_SESSION = 5`  
**When:** 6 observations được gửi vào cùng một session  
**Then:**
- Observations 1-5 được accept và lưu
- Observation thứ 6 nhận response warning (không hard reject)
- `observationCount` = 5 (không tăng thêm sau giới hạn)

**Traceability:** FR-SESSION-003, SRS §3.1

---

## TR-001-SES-008 — Default MAX_OBS_PER_SESSION = 500
🟡 P2 | `[UNIT]` | **FR-SESSION-003**

**Given:** Không có `MAX_OBS_PER_SESSION` env var  
**When:** Config được load  
**Then:** `maxObs = 500`

**Traceability:** FR-SESSION-003, SRS §9.3

---

## TR-001-SES-009 — Session implicit creation
🟠 P1 | `[INT]` | **FR-SESSION-001**

**Given:** Session với `sessionId` chưa tồn tại trong KV  
**When:** Hook `post_tool_use` được gửi (không phải `session_start`)  
**Then:**
- Session được tự động tạo với `status: "active"`
- `project` và `cwd` lấy từ payload (nếu có)
- Không throw error

**Traceability:** SRS §3.1, TDD §2.1 step [13]

---

## TR-001-SES-010 — List sessions với pagination
🟡 P2 | `[INT]` | **FR-SESSION-001**

**Given:** 50 sessions tồn tại trong KV  
**When:** `GET /sessions?limit=10&offset=0`  
**Then:**
- Trả về đúng 10 sessions
- Sessions được sắp xếp theo `startedAt` DESC (newest first)
- Response có metadata: `total`, `limit`, `offset`

**Traceability:** SRS §7.1

---

## TR-001-SES-011 — Filter sessions theo project
🟡 P2 | `[INT]`

**Given:** 20 sessions, 10 cho project "A", 10 cho project "B"  
**When:** `GET /sessions?project=A`  
**Then:** Chỉ trả về 10 sessions của project "A"

**Traceability:** UR-015, SRS §7.1

---

## TR-001-SES-012 — Filter sessions theo status
🟡 P2 | `[INT]`

**Given:** Mix of active, completed, abandoned sessions  
**When:** `GET /sessions?status=completed`  
**Then:** Chỉ trả về sessions với `status = "completed"`

**Traceability:** SRS §7.1

---

## TR-001-SES-013 — Session summary triggered on completion
🟠 P1 | `[INT]` | **FR-SESSION-002**

**Given:** Session với 10 compressed observations  
**When:** Hook `stop` được gửi  
**Then:**
- `mem::summarize` được trigger (async)
- `SessionSummary` được tạo trong KV với:
  - `sessionId`
  - `title` (1 câu)
  - `narrative` (2-3 đoạn)
  - `keyDecisions[]`
  - `filesModified[]`
  - `concepts[]`
  - `observationCount`

**Traceability:** FR-SESSION-002, FR-COMPRESS-003, TDD §6.2

---

## TR-001-SES-014 — Session detail bao gồm observations
🟡 P2 | `[INT]`

**Given:** Session với 5 observations đã được compress  
**When:** `GET /sessions/{id}?include=observations`  
**Then:** Response bao gồm danh sách `CompressedObservation[]`

**Traceability:** SRS §7.1 `memory_session_detail`

---

## TR-001-SES-015 — ObservationCount tăng chính xác
🔴 P0 | `[INT]` | **FR-SESSION-001**

**Given:** Session với `observationCount = 5`  
**When:** 3 observations mới được gửi (không trùng lặp)  
**Then:** `observationCount = 8`

**Traceability:** FR-SESSION-001, TDD §2.1 step [13]
