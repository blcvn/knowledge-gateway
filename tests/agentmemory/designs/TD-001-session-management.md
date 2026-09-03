# TD-001: Session Management Test Design

**Liên kết Requirements:** [TR-001-session-management.md](../requirements/TR-001-session-management.md)  
**Source:** `references/agentmemory/src/functions/observe.ts`  
**Test file:** `tests/agentmemory/specs/session-management.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Module quản lý session xử lý vòng đời của một phiên làm việc agent — từ khi tạo đến khi kết thúc. Session được tạo tự động khi nhận hook đầu tiên từ một sessionId mới.

**Các điểm kiểm thử:**
- Tạo session (tường minh qua `session_start` và ẩn qua hook khác)
- Cập nhật `observationCount` và `firstPrompt`
- Giới hạn số observations (`MAX_OBS_PER_SESSION`)
- Validation payload đầu vào

---

## 2. Chiến lược kiểm thử

| Layer | Phương pháp |
|---|---|
| Unit | MockKV + MockSdk, kiểm thử từng nhánh validation |
| Integration | Gửi nhiều hook events theo thứ tự, xác nhận KV state |
| Boundary | Test tại đúng giá trị giới hạn (MAX_OBS_PER_SESSION) |

---

## 3. Test Cases

### Group A: Tạo Session

#### TC-001 — Tạo session từ hook `session_start`
**Requirement:** TR-001-SES-001 | **Type:** integration | 🔴 P0

**Given:** KV trống, không có session nào tồn tại  
**When:** Hook `session_start` được gửi với `sessionId="sess_abc"`, `project="my-project"`, `cwd="/Users/dev/my-project"`  
**Then:**
- KV `mem:sessions` chứa entry với key `sess_abc`
- Session có `project="my-project"`, `cwd="/Users/dev/my-project"`, `status="active"`
- Session có `startedAt` hợp lệ (ISO timestamp)
- `observationCount = 1`

**Test Data:**
- sessionId: `sess_abc`
- hookType: `session_start`
- project: `my-project`
- cwd: `/Users/dev/my-project`

---

#### TC-002 — ID format của observation: prefix `obs_`, timestamp-based, unique
**Requirement:** TR-001-SES-002 | **Type:** unit | 🔴 P0

**Given:** `generateId("obs")` được gọi  
**When:** Gọi 1000 lần liên tiếp  
**Then:**
- Tất cả IDs có pattern `obs_<base36_timestamp>_<hex_random>`
- Không có 2 IDs trùng nhau

**Kỹ thuật:** Statistical uniqueness test (birthday problem)

---

#### TC-003 — Session fields đầy đủ sau tạo ẩn (implicit)
**Requirement:** TR-001-SES-003 | **Type:** integration | 🔴 P0

**Given:** Không có session nào, hook `post_tool_use` đến trước bất kỳ `session_start`  
**When:** Hook chứa `project` và `cwd` hợp lệ  
**Then:**
- Session được tạo tự động với `id`, `project`, `cwd`, `status="active"`
- `startedAt` = timestamp của hook
- `observationCount = 1`

**Điều kiện kích hoạt implicit creation:** Cả `project` và `cwd` đều phải có trong payload

---

#### TC-004 — Không tạo session implicit khi thiếu `project` hoặc `cwd`
**Requirement:** TR-001-SES-009 | **Type:** integration | 🟠 P1

**Given:** Không có session, hook `post_tool_use` không có `project`  
**When:** Hook được xử lý  
**Then:**
- Observation được ghi vào KV (nếu session đã tồn tại) hoặc bị bỏ qua session creation
- Không crash
- Không tạo session với data không đầy đủ

---

### Group B: firstPrompt

#### TC-005 — `firstPrompt` được capture từ hook `prompt_submit` đầu tiên
**Requirement:** TR-001-SES-006 | **Type:** integration | 🟠 P1

**Given:** Session `sess_fp` đã tồn tại (từ `session_start`), chưa có `firstPrompt`  
**When:** Hook `prompt_submit` với `data.prompt = "Build me an auth system"` được gửi  
**Then:**
- Session `firstPrompt = "Build me an auth system"`
- Chỉ 200 ký tự đầu được lưu nếu prompt dài hơn

**Test Data:** 
- Prompt ngắn: `"Build me an auth system"` → preserved as-is
- Prompt dài: chuỗi 300 chars → chỉ 200 chars đầu được lưu
- Prompt rỗng: `""` → `firstPrompt` không được set

---

#### TC-006 — `firstPrompt` KHÔNG bị ghi đè bởi prompt thứ 2
**Requirement:** TR-001-SES-006 | **Type:** integration | 🟠 P1

**Given:** Session đã có `firstPrompt = "First prompt"`  
**When:** Hook `prompt_submit` thứ 2 với `prompt = "Second prompt"` được gửi  
**Then:** `firstPrompt` vẫn là `"First prompt"` (bất biến sau lần set đầu)

---

### Group C: Observation Count

#### TC-007 — `observationCount` tăng chính xác sau mỗi observation
**Requirement:** TR-001-SES-015 | **Type:** integration | 🔴 P0

**Given:** Session với `observationCount = 0`  
**When:** 5 hook events được gửi tuần tự  
**Then:** `observationCount = 5`

**Kỹ thuật:** Sequential state transition test

---

#### TC-008 — `observationCount` chính xác khi concurrent hooks
**Requirement:** TR-001-SES-015 | **Type:** integration | 🔴 P0

**Given:** Session tồn tại  
**When:** 10 hooks gửi đồng thời (Promise.all)  
**Then:** `observationCount = 10` (không mất update do keyed-mutex)

**Lý giải:** `withKeyedLock("obs:sessionId")` ngăn race condition. Test này xác nhận mutex hoạt động.

---

### Group D: Observation Limit

#### TC-009 — Hook bị từ chối khi đạt `MAX_OBS_PER_SESSION`
**Requirement:** TR-001-SES-007 | **Type:** integration | 🔴 P0

**Given:** Session với đúng `MAX_OBS_PER_SESSION` observations (ví dụ: 3 nếu set max=3)  
**When:** Hook thứ `MAX+1` được gửi  
**Then:**
- Trả về `{success: false, error: "...limit..."}` 
- KV không thay đổi (không có observation mới)

**Boundary values:** Test tại `max-1` (pass), `max` (pass), `max+1` (fail)

---

#### TC-010 — `MAX_OBS_PER_SESSION` mặc định là 500
**Requirement:** TR-001-SES-008 | **Type:** unit | 🟠 P1

**Given:** Biến môi trường `MAX_OBS_PER_SESSION` không được set  
**When:** `getMaxObservationsPerSession()` được gọi  
**Then:** Trả về `500`

---

#### TC-011 — `MAX_OBS_PER_SESSION` có thể cấu hình qua env var
**Type:** unit | 🟡 P2

**Given:** `MAX_OBS_PER_SESSION=100`  
**When:** `getMaxObservationsPerSession()` gọi  
**Then:** Trả về `100`

---

### Group E: Validation

#### TC-012 — Từ chối payload thiếu `sessionId`
**Requirement:** TR-001-SES-011 | **Type:** unit | 🔴 P0

**Given:** Payload không có `sessionId`  
**When:** `mem::observe` được trigger  
**Then:** `{success: false, error: "...sessionId...required..."}` (không crash)

---

#### TC-013 — Từ chối payload thiếu `hookType`
**Type:** unit | 🔴 P0

**Given:** Payload có `sessionId` nhưng không có `hookType`  
**When:** `mem::observe` trigger  
**Then:** `{success: false}` với error message

---

#### TC-014 — Từ chối payload thiếu `timestamp`
**Type:** unit | 🔴 P0

**Given:** Payload có `sessionId`, `hookType` nhưng không có `timestamp`  
**When:** `mem::observe` trigger  
**Then:** `{success: false}` với error message

---

#### TC-015 — `sessionId` phải là string (không phải number hay null)
**Type:** unit | 🟠 P1

**Given:** Payload có `sessionId: 12345` (number)  
**When:** `mem::observe` trigger  
**Then:** `{success: false}` — type guard validation

---

## 4. Test Data Matrix

| TC | sessionId | hookType | project/cwd | Expected |
|---|---|---|---|---|
| TC-001 | `sess_abc` | `session_start` | Có | Session created |
| TC-003 | `sess_implicit` | `post_tool_use` | Có | Implicit create |
| TC-004 | `sess_noproj` | `post_tool_use` | Không có | No session create |
| TC-009 | `sess_full` | `post_tool_use` | Có | Rejected (limit) |
| TC-012 | *(missing)* | `post_tool_use` | Không quan trọng | Error |

---

## 5. Coverage Notes

- `observe.ts` lines 46-60: validation block
- `observe.ts` lines 127-135: limit check
- `observe.ts` lines 244-274: implicit session creation
- `observe.ts` lines 232-241: firstPrompt capture
- `schema.ts`: `generateId()` function
