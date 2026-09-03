# TC-001: Session Management — Test Cases

**Test Design tham chiếu:** [TD-001](../designs/TD-001-session-management.md)  
**Requirements tham chiếu:** [TR-001](../requirements/TR-001-session-management.md)  
**Module:** Session Lifecycle, Observation Count, firstPrompt, Validation  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-001-001: Tạo session từ hook `session_start`

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-001 |
| **Tên** | Tạo session thành công từ hook session_start |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Design ref** | TD-001 / TC-001 |
| **Requirement** | TR-001-SES-001 |

**Điều kiện tiên quyết:**
- Hệ thống agentmemory đang chạy
- KV store trống (không có session nào tồn tại)
- Mock SDK đã được khởi tạo

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_abc123` |
| `hookType` | `session_start` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |
| `project` | `my-project` |
| `cwd` | `/Users/dev/my-project` |
| `data` | `{}` |

**Các bước thực hiện:**
1. Gửi hook payload đến function `mem::observe` qua MockSdk
2. Chờ function xử lý xong
3. Đọc KV store tại scope `mem:sessions`, key `sess_abc123`
4. Inspect kết quả trả về và KV state

**Kết quả mong đợi:**
- Response có field `observationId` (string, không rỗng)
- KV `mem:sessions["sess_abc123"]` tồn tại
- `session.project = "my-project"`
- `session.cwd = "/Users/dev/my-project"`
- `session.status = "active"`
- `session.observationCount = 1`
- `session.startedAt` là ISO 8601 timestamp hợp lệ

**Tiêu chí Pass:** Tất cả 7 điều kiện kết quả đều đúng.  
**Tiêu chí Fail:** Bất kỳ điều kiện nào sai, hoặc có exception.

---

## TC-001-002: Observation ID có đúng format và unique

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-002 |
| **Tên** | generateId tạo ID unique với đúng prefix format |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-001-SES-002 |

**Điều kiện tiên quyết:**
- Module `schema.ts` được import
- Môi trường Node.js

**Dữ liệu đầu vào:**
- Prefix: `"obs"`
- Số lần gọi: 1000

**Các bước thực hiện:**
1. Gọi `generateId("obs")` 1000 lần liên tiếp
2. Thu thập tất cả 1000 IDs vào một Set
3. Kiểm tra kích thước Set
4. Kiểm tra pattern của mỗi ID

**Kết quả mong đợi:**
- Set có đúng 1000 phần tử (không có duplicate)
- Mỗi ID khớp pattern: `obs_` + base36 characters + `_` + hex characters
- Không có 2 IDs giống nhau
- Phần timestamp trong ID tăng dần theo thời gian

**Tiêu chí Pass:** Set.size = 1000, mọi ID match regex `^obs_[0-9a-z]+_[0-9a-f]{12}$`

---

## TC-001-003: Session implicit creation từ hook `post_tool_use`

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-003 |
| **Tên** | Session được tạo ẩn (implicit) khi hook đầu tiên không phải session_start |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-001-SES-003 |

**Điều kiện tiên quyết:**
- KV store không có session `sess_implicit`
- Không có lệnh `session_start` nào được gửi trước

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_implicit` |
| `hookType` | `post_tool_use` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |
| `project` | `test-project` |
| `cwd` | `/Users/dev/test-project` |
| `data.tool_name` | `edit_file` |
| `data.tool_input` | `{path: "src/auth.ts"}` |

**Các bước thực hiện:**
1. Gửi hook `post_tool_use` với payload trên (không có prior session_start)
2. Đọc KV `mem:sessions["sess_implicit"]`
3. Kiểm tra response và session state

**Kết quả mong đợi:**
- Session được tạo tự động
- `session.id = "sess_implicit"`
- `session.project = "test-project"`
- `session.cwd = "/Users/dev/test-project"`
- `session.status = "active"`
- `session.startedAt` xấp xỉ timestamp của hook
- `session.observationCount = 1`

**Tiêu chí Pass:** Session tồn tại trong KV với tất cả fields đúng.

---

## TC-001-004: Không tạo session implicit khi thiếu `project`

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-004 |
| **Tên** | Không tạo session khi hook thiếu field project |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-001-SES-009 |

**Điều kiện tiên quyết:**
- KV store trống

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_noproj` |
| `hookType` | `post_tool_use` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |
| `project` | *(bị bỏ qua / undefined)* |
| `cwd` | `/Users/dev/test` |

**Các bước thực hiện:**
1. Gửi hook `post_tool_use` không có field `project`
2. Đọc KV `mem:sessions["sess_noproj"]`
3. Kiểm tra response

**Kết quả mong đợi:**
- Session `sess_noproj` KHÔNG tồn tại trong KV
- Không có exception / crash
- Response có thể là `{success: false}` hoặc bị bỏ qua

**Tiêu chí Pass:** KV không có `sess_noproj`, hệ thống không crash.

---

## TC-001-005: `firstPrompt` được capture từ hook `prompt_submit` đầu tiên

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-005 |
| **Tên** | firstPrompt được lưu từ hook prompt_submit đầu tiên |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-001-SES-006 |

**Điều kiện tiên quyết:**
- Session `sess_fp` đã tồn tại trong KV (đã qua TC-001-001)
- Session chưa có `firstPrompt`

**Dữ liệu đầu vào (Hook 1 — session_start):**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_fp` |
| `hookType` | `session_start` |
| `project` | `p` |
| `cwd` | `/p` |

**Dữ liệu đầu vào (Hook 2 — prompt_submit):**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_fp` |
| `hookType` | `prompt_submit` |
| `data.prompt` | `Build me an auth system` |

**Các bước thực hiện:**
1. Gửi hook `session_start` để tạo session
2. Gửi hook `prompt_submit` với prompt ngắn (< 200 chars)
3. Đọc session từ KV
4. Kiểm tra field `firstPrompt`

**Kết quả mong đợi:**
- `session.firstPrompt = "Build me an auth system"`

**Test Data bổ sung (Prompt dài):**
- Gửi prompt 300 ký tự → `firstPrompt` chỉ có 200 ký tự đầu
- Gửi prompt rỗng `""` → `firstPrompt` vẫn là `undefined`

**Tiêu chí Pass:** `firstPrompt` khớp với prompt ngắn; bị truncate với prompt dài.

---

## TC-001-006: `firstPrompt` không bị overwrite bởi prompt thứ 2

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-006 |
| **Tên** | firstPrompt là immutable sau khi được set lần đầu |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-001-SES-006 |

**Điều kiện tiên quyết:**
- Session `sess_immutable` đã có `firstPrompt = "First prompt"`

**Dữ liệu đầu vào:**

| Bước | hookType | prompt |
|---|---|---|
| 1 | `prompt_submit` | `"First prompt"` |
| 2 | `prompt_submit` | `"Second prompt — should NOT overwrite"` |

**Các bước thực hiện:**
1. Gửi hook `session_start` để tạo session
2. Gửi hook `prompt_submit` với `"First prompt"`
3. Verify `firstPrompt = "First prompt"`
4. Gửi hook `prompt_submit` thứ 2 với `"Second prompt — should NOT overwrite"`
5. Đọc lại session từ KV
6. Kiểm tra `firstPrompt`

**Kết quả mong đợi:**
- Sau bước 5: `session.firstPrompt` vẫn là `"First prompt"` (không đổi)

**Tiêu chí Pass:** `firstPrompt` = `"First prompt"` sau cả 2 prompt_submit.

---

## TC-001-007: `observationCount` tăng chính xác sau mỗi hook

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-007 |
| **Tên** | observationCount increment sau mỗi hook event tuần tự |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-001-SES-015 |

**Điều kiện tiên quyết:**
- Session `sess_cnt` tồn tại, `observationCount = 0`

**Dữ liệu đầu vào:**
- 5 hook events tuần tự, mỗi cái có `sessionId = "sess_cnt"`, `hookType = "post_tool_use"`
- Timestamp tăng dần: `T+0s, T+1s, T+2s, T+3s, T+4s`

**Các bước thực hiện:**
1. Gửi hook 1 → đọc session → `observationCount = 1`
2. Gửi hook 2 → đọc session → `observationCount = 2`
3. Gửi hook 3 → đọc session → `observationCount = 3`
4. Gửi hook 4 → đọc session → `observationCount = 4`
5. Gửi hook 5 → đọc session → `observationCount = 5`

**Kết quả mong đợi:**
- `observationCount` = `{số hooks đã gửi}` tại mỗi bước

**Tiêu chí Pass:** `observationCount = 5` sau 5 hooks.

---

## TC-001-008: `observationCount` chính xác với concurrent hooks (mutex test)

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-008 |
| **Tên** | observationCount không bị lost update khi hooks đến đồng thời |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-001-SES-015 |

**Điều kiện tiên quyết:**
- Session `sess_concurrent` tồn tại

**Dữ liệu đầu vào:**
- 10 hook payloads giống nhau ngoại trừ timestamp (mỗi cái lệch 1ms)
- Tất cả có `sessionId = "sess_concurrent"`

**Các bước thực hiện:**
1. Gửi 10 hooks đồng thời (parallel dispatch)
2. Chờ tất cả 10 requests hoàn thành
3. Đọc `observationCount` từ KV

**Kết quả mong đợi:**
- Tất cả 10 requests trả về response với `observationId`
- `session.observationCount = 10` (không mất update)
- KV có đúng 10 observations trong `mem:obs:sess_concurrent`

**Tiêu chí Pass:** `observationCount = 10` và 10 observation entries trong KV.

---

## TC-001-009: Hook bị từ chối khi đạt MAX_OBS_PER_SESSION — boundary test

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-009 |
| **Tên** | Hook thứ MAX+1 bị từ chối, các hook trước đó pass |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-001-SES-007 |

**Điều kiện tiên quyết:**
- `MAX_OBS_PER_SESSION = 3` (test config)
- Session `sess_limit` tồn tại với `observationCount = 0`

**Dữ liệu đầu vào:**

| Hook # | sessionId | hookType | Expected |
|---|---|---|---|
| 1 | `sess_limit` | `post_tool_use` | ✅ Accept |
| 2 | `sess_limit` | `post_tool_use` | ✅ Accept |
| 3 | `sess_limit` | `post_tool_use` | ✅ Accept (last allowed) |
| 4 | `sess_limit` | `post_tool_use` | ❌ Reject |

**Các bước thực hiện:**
1. Gửi hook #1 → verify accepted (`observationId` present)
2. Gửi hook #2 → verify accepted
3. Gửi hook #3 → verify accepted (đây là hook thứ MAX = 3)
4. Gửi hook #4 → verify REJECTED
5. Đọc `observationCount` từ KV

**Kết quả mong đợi (Hook #4):**
- Response có `success: false`
- Response có `error` message chứa từ "limit" hoặc "maximum"
- KV không thay đổi sau hook #4 (vẫn là 3 observations)
- `session.observationCount` vẫn là `3`

**Tiêu chí Pass:** Hook #4 bị từ chối, KV count = 3 (không tăng lên 4).

---

## TC-001-010: MAX_OBS_PER_SESSION mặc định là 500

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-010 |
| **Tên** | Giá trị mặc định của MAX_OBS_PER_SESSION là 500 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-001-SES-008 |

**Điều kiện tiên quyết:**
- Biến môi trường `MAX_OBS_PER_SESSION` KHÔNG được set

**Các bước thực hiện:**
1. Đảm bảo `process.env.MAX_OBS_PER_SESSION` = undefined
2. Gọi hàm `getMaxObservationsPerSession()`
3. Kiểm tra giá trị trả về

**Kết quả mong đợi:**
- Trả về `500`

**Tiêu chí Pass:** Return value = `500`.

---

## TC-001-011: MAX_OBS_PER_SESSION có thể cấu hình qua env var

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-011 |
| **Tên** | MAX_OBS_PER_SESSION được đọc từ environment variable |
| **Loại** | Unit |
| **Ưu tiên** | 🟡 P2 |

**Điều kiện tiên quyết:**
- `MAX_OBS_PER_SESSION = "100"` được set trong env

**Các bước thực hiện:**
1. Set `process.env.MAX_OBS_PER_SESSION = "100"`
2. Gọi `getMaxObservationsPerSession()`
3. Kiểm tra giá trị trả về

**Kết quả mong đợi:**
- Trả về `100` (number, không phải string "100")

**Tiêu chí Pass:** Return value = `100`.

---

## TC-001-012: Payload thiếu `sessionId` bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-012 |
| **Tên** | mem::observe trả về error khi thiếu sessionId |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-001-SES-011 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | *(bị bỏ qua)* |
| `hookType` | `post_tool_use` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |

**Các bước thực hiện:**
1. Gọi `mem::observe` với payload không có `sessionId`
2. Kiểm tra response

**Kết quả mong đợi:**
- `response.success = false`
- `response.error` là string không rỗng
- Error message đề cập đến "sessionId" hoặc "required"
- Hệ thống không crash

**Tiêu chí Pass:** Response có `success: false` với error message có nghĩa.

---

## TC-001-013: Payload thiếu `hookType` bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-013 |
| **Tên** | mem::observe trả về error khi thiếu hookType |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_valid` |
| `hookType` | *(bị bỏ qua)* |
| `timestamp` | `2026-06-10T14:00:00.000Z` |

**Các bước thực hiện:**
1. Gọi `mem::observe` với payload không có `hookType`
2. Kiểm tra response

**Kết quả mong đợi:**
- `response.success = false`
- `response.error` không rỗng

**Tiêu chí Pass:** Response `{success: false}` với error.

---

## TC-001-014: Payload thiếu `timestamp` bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-014 |
| **Tên** | mem::observe trả về error khi thiếu timestamp |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_valid` |
| `hookType` | `post_tool_use` |
| `timestamp` | *(bị bỏ qua)* |

**Các bước thực hiện:**
1. Gọi `mem::observe` với payload không có `timestamp`
2. Kiểm tra response

**Kết quả mong đợi:**
- `response.success = false`
- `response.error` không rỗng

**Tiêu chí Pass:** Response `{success: false}` với error.

---

## TC-001-015: `sessionId` là number bị từ chối (type guard)

| Trường | Giá trị |
|---|---|
| **ID** | TC-001-015 |
| **Tên** | sessionId phải là string, number bị từ chối |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `sessionId` | `12345` (number, không phải string) |
| `hookType` | `post_tool_use` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |

**Các bước thực hiện:**
1. Gọi `mem::observe` với `sessionId: 12345`
2. Kiểm tra response

**Kết quả mong đợi:**
- `response.success = false`
- Không tạo session với key `12345` trong KV

**Tiêu chí Pass:** Type validation được thực thi, `success: false`.

---

## Tổng kết Module TC-001

| TC ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-001-001 | Session từ session_start | 🔴 P0 | Integration |
| TC-001-002 | ID format & unique | 🔴 P0 | Unit |
| TC-001-003 | Implicit session creation | 🔴 P0 | Integration |
| TC-001-004 | No session khi thiếu project | 🟠 P1 | Integration |
| TC-001-005 | firstPrompt capture | 🟠 P1 | Integration |
| TC-001-006 | firstPrompt immutable | 🟠 P1 | Integration |
| TC-001-007 | observationCount sequential | 🔴 P0 | Integration |
| TC-001-008 | observationCount concurrent | 🔴 P0 | Integration |
| TC-001-009 | Limit boundary test | 🔴 P0 | Integration |
| TC-001-010 | Default MAX=500 | 🟠 P1 | Unit |
| TC-001-011 | MAX from env var | 🟡 P2 | Unit |
| TC-001-012 | Missing sessionId | 🔴 P0 | Unit |
| TC-001-013 | Missing hookType | 🔴 P0 | Unit |
| TC-001-014 | Missing timestamp | 🔴 P0 | Unit |
| TC-001-015 | sessionId type guard | 🟠 P1 | Unit |
