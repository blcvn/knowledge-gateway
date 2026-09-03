# TD-022: Deployment Test Design

**Liên kết Requirements:** [TR-022-deployment.md](../requirements/TR-022-deployment.md)  
**Test file:** `tests/agentmemory/specs/deployment.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Deployment tests verify quá trình cài đặt, khởi động, và vận hành hệ thống trong môi trường thực tế.

---

## 2. Chiến lược kiểm thử

| Layer | Phương pháp |
|---|---|
| Unit | Config parsing, env var validation |
| Integration | Start server process, verify health |
| E2E | Full lifecycle: install → start → use → stop |

---

## 3. Test Cases

### Group A: CLI và Configuration

#### TC-001 — `agentmemory --help` trả về usage info không crash
**Requirement:** TR-022-DEP-001 | **Type:** integration | 🔴 P0

**Given:** Binary được build  
**When:** `agentmemory --help` gọi  
**Then:**
- Exit code 0
- Output chứa usage info
- Không có unhandled error

---

#### TC-002 — `agentmemory --version` trả về đúng version string
**Type:** integration | 🔴 P0

**Given:** Package built với version từ `package.json`  
**When:** `agentmemory --version`  
**Then:** Output match pattern `\d+\.\d+\.\d+`

---

#### TC-003 — Server khởi động thành công và respond healthy
**Requirement:** TR-022-DEP-002 | **Type:** integration | 🔴 P0

**Given:** Không có `AGENTMEMORY_SECRET`, không có external deps  
**When:**
1. Start server process
2. Wait cho ready signal
3. `GET /health`

**Then:** Status 200, `{status: "ok"}`

---

#### TC-004 — Port conflict: server fail gracefully với clear error message
**Type:** integration | 🟠 P1

**Given:** Port 3721 đang được dùng bởi process khác  
**When:** Start agentmemory trên port 3721  
**Then:**
- Exit code non-zero
- Stderr có message về "port in use" hoặc "EADDRINUSE"

---

### Group B: Environment Variables

#### TC-005 — Tất cả required env vars được validate khi startup
**Requirement:** TR-022-DEP-003 | **Type:** unit | 🔴 P0

**Given:** `AGENTMEMORY_SECRET` có value nhưng < 16 chars  
**When:** Server startup  
**Then:** Fail fast với error "SECRET too short (min 16 chars)"

---

#### TC-006 — `TOKEN_BUDGET` invalid value → use default
**Type:** unit | 🟡 P2

**Given:** `TOKEN_BUDGET=not-a-number`  
**When:** Server startup  
**Then:** Server starts (không crash), dùng default TOKEN_BUDGET=8000

---

#### TC-007 — Missing optional env vars không block startup
**Requirement:** TR-022-DEP-004 | **Type:** unit | 🔴 P0

**Given:** Chỉ có mandatory env vars được set, không có optionals  
**When:** Server startup  
**Then:** Server khởi động thành công

---

### Group C: Graceful Shutdown

#### TC-008 — SIGTERM: server gracefully shutdown
**Requirement:** TR-022-DEP-005 | **Type:** integration | 🔴 P0

**Given:** Server running với active connections  
**When:** `kill -TERM <pid>` gửi  
**Then:**
- In-flight requests được complete (hoặc timeout 5s)
- Server exits với code 0
- Không có data corruption trong KV

---

#### TC-009 — SIGINT (Ctrl+C): graceful shutdown
**Type:** integration | 🟠 P1

**Given:** Server running  
**When:** SIGINT signal gửi  
**Then:** Graceful exit, code 0

---

### Group D: iii-engine Integration

#### TC-010 — iii-engine v0.11.2 binary compatible
**Requirement:** TR-022-DEP-006 | **Type:** integration | 🔴 P0

**Given:** iii-engine v0.11.2 binary trong PATH  
**When:** Server start  
**Then:**
- Không có "version mismatch" error
- KV operations hoạt động (ping test)

---

#### TC-011 — iii-engine not found: informative error
**Type:** integration | 🔴 P0

**Given:** iii-engine binary không có trong PATH  
**When:** Server start  
**Then:**
- Exit code non-zero
- Error message: "iii-engine not found — install with iii-install"

---

### Group E: npm Install

#### TC-012 — `npm install` không fail với peer dependency warnings
**Type:** integration | 🟠 P1

**Given:** Clean node_modules  
**When:** `npm install` chạy  
**Then:**
- Exit code 0
- Không có missing peer deps errors (warnings OK)

---

## 4. Notes

- Server startup tests cần manage child processes với proper cleanup
- Platform matrix: test trên macOS ARM64, Linux x64 (không cần Windows)
- `iii-engine` version được pin trong `package.json` → verify khớp
