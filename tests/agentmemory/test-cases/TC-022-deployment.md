# TC-022: Deployment — Test Cases

**Test Design tham chiếu:** [TD-022](../designs/TD-022-deployment.md)  
**Requirements tham chiếu:** [TR-022](../requirements/TR-022-deployment.md)  
**Module:** CLI, Startup/Shutdown, Error Handling  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-022-001: `agentmemory --help` exit 0 với usage info

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-022-DEP-001 |

**Điều kiện tiên quyết:** `agentmemory` binary trong PATH

**Các bước thực hiện:**
1. Chạy `agentmemory --help`
2. Capture exit code và stdout

**Kết quả mong đợi:**
- Exit code = 0
- Stdout có usage information (chứa ít nhất một trong: `usage`, `options`, `commands`, `--help`)

---

## TC-022-002: Server khởi động và respond /health trong 5s

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-022-DEP-002 |

**Điều kiện tiên quyết:** iii-engine available trong PATH

**Các bước thực hiện:**
1. Start `agentmemory` server process (mặc định config)
2. Poll `GET /health` mỗi 500ms, timeout 5s
3. Verify response

**Kết quả mong đợi:**
- `GET /health` trả về HTTP 200 trong vòng 5 giây
- Không cần manual intervention

**Cleanup:** Kill server process sau test

---

## TC-022-003: SIGTERM → graceful shutdown, exit 0

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-022-DEP-003 |

**Điều kiện tiên quyết:** Server đang running và healthy

**Các bước thực hiện:**
1. Verify server healthy (GET /health → 200)
2. Send `SIGTERM` đến server process
3. Chờ process exit (timeout 10s)
4. Kiểm tra exit code
5. Đọc KV data để kiểm tra integrity

**Kết quả mong đợi:**
- Exit code = 0
- KV data không bị corrupt (không có partial writes)
- Process exit trong vòng 10 giây

---

## TC-022-004: SIGKILL → restart từ persistent KV state

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-022-DEP-004 |

**Các bước thực hiện:**
1. Create 5 memories qua API
2. Force kill process (SIGKILL)
3. Restart server
4. `GET /memories` → kiểm tra data

**Kết quả mong đợi:** 5 memories vẫn tồn tại sau restart (persistent KV)

---

## TC-022-005: iii-engine binary không tìm thấy → informative error, non-zero exit

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-022-DEP-005 |

**Setup:**
- Override PATH để iii-engine không accessible
- Hoặc rename binary tạm thời

**Các bước thực hiện:**
1. Start `agentmemory` với iii-engine missing
2. Capture exit code và stderr

**Kết quả mong đợi:**
- Exit code ≠ 0
- Stderr đề cập đến `"iii-engine"` hoặc `"KV"` hoặc `"binary"`

---

## TC-022-006: Port conflict → informative error, non-zero exit

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-022-DEP-006 |

**Setup:**
1. Start một process chiếm port (e.g., port 7332)
2. Start `agentmemory` với cùng port

**Kết quả mong đợi:**
- Exit code ≠ 0
- Stderr chứa `"EADDRINUSE"` hoặc `"port in use"` hoặc `"address already in use"`

---

## TC-022-007: `agentmemory --version` in đúng version string

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-022-DEP-007 |

**Các bước thực hiện:**
1. Chạy `agentmemory --version`
2. Capture stdout

**Kết quả mong đợi:**
- Exit code = 0
- Stdout chứa version string theo format semver (e.g., `1.2.3` hoặc `v1.2.3`)

---

## TC-022-008: Custom port via `--port` flag hoạt động

| Trường | Giá trị |
|---|---|
| **ID** | TC-022-008 |
| **Loại** | Integration |
| **Ưu tiên** | 🟡 P2 |
| **Requirement** | TR-022-DEP-008 |

**Các bước thực hiện:**
1. Start `agentmemory --port 8888`
2. `GET http://localhost:8888/health`

**Kết quả mong đợi:** HTTP 200 từ port 8888

---

## Tổng kết TC-022

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-022-001 | --help exit 0 | 🔴 P0 | Integration |
| TC-022-002 | Server startup < 5s | 🔴 P0 | Integration |
| TC-022-003 | SIGTERM → graceful exit 0 | 🔴 P0 | Integration |
| TC-022-004 | SIGKILL → data persistent | 🟠 P1 | Integration |
| TC-022-005 | iii-engine missing → error | 🔴 P0 | Integration |
| TC-022-006 | Port conflict → error | 🟠 P1 | Integration |
| TC-022-007 | --version semver | 🟠 P1 | Integration |
| TC-022-008 | Custom --port | 🟡 P2 | Integration |
