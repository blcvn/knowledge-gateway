# User Requirements Document (URD)

## OpenViking — Context Database for AI Agents

| Field           | Value                    |
| --------------- | ------------------------ |
| **Product**     | OpenViking               |
| **Version**     | 0.1.x (Alpha)            |
| **Last Updated**| 2026-05-07               |

---

## 1. User Personas

### 1.1 Persona A — AI Agent Developer

| Attribute        | Chi tiết                                                         |
|------------------|------------------------------------------------------------------|
| **Vai trò**      | Software Engineer / AI Engineer                                  |
| **Mục tiêu**     | Xây dựng Agent có bộ nhớ dài hạn, truy xuất context chính xác   |
| **Pain Points**  | Phải tự quản lý vector DB, prompt stuffing, context fragmentation|
| **Kỹ năng**      | Python, REST API, LLM integration, Docker                       |
| **Tần suất**     | Hàng ngày — tích hợp OpenViking vào pipeline Agent              |

**User Stories**:
- US-A1: Tôi muốn thêm tài liệu dự án vào OpenViking bằng một lệnh duy nhất để Agent có thể truy vấn.
- US-A2: Tôi muốn Agent tự động trích xuất long-term memory từ conversation để thông minh hơn qua mỗi phiên.
- US-A3: Tôi muốn tìm kiếm context theo semantic query và nhận kết quả xếp hạng với relevance score.
- US-A4: Tôi muốn quản lý resources, memories, và skills qua API RESTful thống nhất.
- US-A5: Tôi muốn sử dụng Python SDK (sync/async) để tích hợp nhanh vào ứng dụng.

### 1.2 Persona B — IDE Plugin User (Claude Code / OpenCode / Codex)

| Attribute        | Chi tiết                                                         |
|------------------|------------------------------------------------------------------|
| **Vai trò**      | Software Developer using AI coding assistants                   |
| **Mục tiêu**     | AI coding assistant nhớ context dự án giữa các phiên            |
| **Pain Points**  | AI quên context sau mỗi session, phải lặp lại giải thích       |
| **Kỹ năng**      | Coding, IDE usage, basic CLI                                     |
| **Tần suất**     | Hàng ngày — AI assistant sử dụng OpenViking như persistent memory|

**User Stories**:
- US-B1: Tôi muốn AI assistant nhớ coding preferences và project structure giữa các phiên làm việc.
- US-B2: Tôi muốn AI có thể tìm kiếm trong codebase đã index mà không cần copy-paste toàn bộ code.
- US-B3: Tôi muốn nói "remember this" và AI sẽ lưu thông tin quan trọng vào long-term memory.
- US-B4: Tôi muốn AI assistant sử dụng MCP protocol để tương tác tự nhiên với OpenViking.

### 1.3 Persona C — Platform Administrator

| Attribute        | Chi tiết                                                         |
|------------------|------------------------------------------------------------------|
| **Vai trò**      | DevOps / Platform Engineer                                       |
| **Mục tiêu**     | Triển khai và vận hành OpenViking cho nhiều teams/users           |
| **Pain Points**  | Multi-tenancy, security, monitoring, scaling                     |
| **Kỹ năng**      | Docker, K8s, monitoring, security                                |
| **Tần suất**     | Hàng tuần — quản lý infrastructure, review metrics               |

**User Stories**:
- US-C1: Tôi muốn tạo và quản lý accounts/users với RBAC (ROOT/ADMIN/USER).
- US-C2: Tôi muốn bật encryption at-rest để bảo vệ dữ liệu nhạy cảm.
- US-C3: Tôi muốn monitor hệ thống qua Prometheus metrics và Grafana dashboards.
- US-C4: Tôi muốn triển khai multi-worker hoặc trên Kubernetes với Helm chart.
- US-C5: Tôi muốn cấu hình namespace isolation để đảm bảo dữ liệu giữa các team không bị trộn lẫn.

### 1.4 Persona D — VikingBot End User

| Attribute        | Chi tiết                                                         |
|------------------|------------------------------------------------------------------|
| **Vai trò**      | Non-technical user tương tác qua chat interface                  |
| **Mục tiêu**     | Hỏi đáp, quản lý thông tin cá nhân qua chatbot AI               |
| **Pain Points**  | Cần interface đơn giản, không muốn dùng CLI/API                  |
| **Kỹ năng**      | Basic chat, messaging apps                                       |
| **Tần suất**     | Hàng ngày — chat qua Telegram/WeChat/Feishu/Slack               |

**User Stories**:
- US-D1: Tôi muốn chat với AI và nó nhớ những gì tôi đã nói trong các phiên trước.
- US-D2: Tôi muốn Agent giúp tôi tìm kiếm thông tin trong tài liệu đã upload.
- US-D3: Tôi muốn sử dụng bot qua Telegram, Feishu, hoặc web console.

---

## 2. Interaction Models

### 2.1 CLI Interaction (Persona A, B)

```
                      ┌──────────────┐
   Developer ────────►│   ov CLI     │
                      │  (Rust)      │
                      └──────┬───────┘
                             │ HTTP
                      ┌──────▼───────┐
                      │ OpenViking   │
                      │ Server :1933 │
                      └──────────────┘
```

**Core CLI Commands**:

| Command                          | Mô tả                                       |
|----------------------------------|----------------------------------------------|
| `ov status`                      | Kiểm tra server health                       |
| `ov add-resource <path/url>`     | Thêm resource (local path hoặc URL)          |
| `ov ls <viking-uri>`            | Liệt kê nội dung directory                   |
| `ov tree <viking-uri> -L N`     | Hiển thị cây thư mục                         |
| `ov find <query>`               | Tìm kiếm semantic                            |
| `ov grep <pattern> --uri <uri>` | Tìm kiếm regex trong content                 |
| `ov read <viking-uri>`          | Đọc nội dung file                             |
| `ov chat`                       | Interactive chat với VikingBot                |

### 2.2 Python SDK Interaction (Persona A)

```python
# Sync client
from openviking import OpenViking

client = OpenViking(url="http://localhost:1933")

# Add resource
client.add_resource("https://github.com/org/repo")

# Search context
results = client.find("authentication flow", limit=5)

# Session management
session = client.create_session()
session.add_message("user", "How does the auth work?")
session.commit()
```

```python
# Async client
from openviking import AsyncOpenViking

async def main():
    client = AsyncOpenViking(url="http://localhost:1933")
    results = await client.find("database schema", limit=10)
```

### 2.3 MCP Interaction (Persona B)

```
   AI IDE (Claude/Codex) ───── MCP Protocol ─────► /mcp endpoint
                                                    │
                                          ┌─────────┴──────────┐
                                          │  9 MCP Tools       │
                                          │  search, read,     │
                                          │  list, store,      │
                                          │  add_resource,     │
                                          │  grep, glob,       │
                                          │  forget, health    │
                                          └────────────────────┘
```

**MCP Tool Interaction Flow**:

1. IDE gửi `search` query → OpenViking trả về ranked results
2. IDE gọi `read` với URI → Nhận full content
3. User nói "remember this" → IDE gọi `store` → Memory extraction
4. IDE muốn index repo → Gọi `add_resource` với git URL

### 2.4 REST API Interaction (Persona A, C)

```
   Application ────► HTTP REST API (:1933)
                     │
                     ├── /api/v1/resources/*     # Resource management
                     ├── /api/v1/filesystem/*     # File operations
                     ├── /api/v1/content/*        # Content read/write
                     ├── /api/v1/search/*         # Semantic search
                     ├── /api/v1/sessions/*       # Session lifecycle
                     ├── /api/v1/admin/*          # Account/user CRUD
                     ├── /api/v1/observer/*       # Retrieval stats
                     ├── /api/v1/privacy-configs/* # Privacy settings
                     ├── /api/v1/tasks/*          # Task tracking
                     └── /metrics                 # Prometheus metrics
```

### 2.5 Chat Interaction (Persona D)

```
   User ──── Telegram/Feishu/Slack ────► VikingBot Gateway (:18790)
                                         │
                                         ├── Agent Tool Execution
                                         ├── OpenViking Context Retrieval
                                         └── Memory Persistence
```

---

## 3. Standard Operating Procedures (SOPs)

### SOP-01: Initial Setup & Configuration

| Step | Action                                    | Chi tiết                                           |
|------|-------------------------------------------|----------------------------------------------------|
| 1    | Install OpenViking                        | `pip install openviking --upgrade`                 |
| 2    | Initialize configuration                  | `openviking-server init` (interactive wizard)      |
| 3    | Validate setup                            | `openviking-server doctor`                         |
| 4    | Start server                              | `openviking-server`                                |
| 5    | Verify                                    | `ov status`                                        |

**Configuration file**: `~/.openviking/ov.conf` (JSON)

**Required settings**:
- `storage.workspace` — đường dẫn lưu trữ dữ liệu
- `embedding.dense.*` — Embedding model config (provider, API key, model name)
- `vlm.*` — VLM model config (provider, API key, model name)

### SOP-02: Resource Ingestion

| Step | Action                                    | Chi tiết                                           |
|------|-------------------------------------------|----------------------------------------------------|
| 1    | Add resource                              | `ov add-resource <URL_or_path> [--wait]`           |
| 2    | Check processing status                   | `ov ls viking://resources/`                        |
| 3    | Browse structure                          | `ov tree viking://resources/<name> -L 3`           |
| 4    | Search content                            | `ov find "your query"`                              |
| 5    | Read specific file                        | `ov read viking://resources/<name>/path/to/file`   |

**Supported input types**:
- Git repository URL → Auto-clone, tree-sitter parse, embed
- HTTP URL → Scrape, convert to markdown, embed
- Local directory → Recursive scan, detect file types, embed
- Single file (PDF/DOCX/PPTX/XLSX/EPUB) → Parse, chunk, embed

### SOP-03: Session & Memory Management

| Step | Action                                    | Chi tiết                                           |
|------|-------------------------------------------|----------------------------------------------------|
| 1    | Create session                            | `POST /api/v1/sessions` hoặc SDK `create_session()`|
| 2    | Add messages                              | `POST /api/v1/sessions/{id}/messages`              |
| 3    | Record context usage                      | `POST /api/v1/sessions/{id}/used`                  |
| 4    | Commit (trigger memory extraction)        | `POST /api/v1/sessions/{id}/commit`                |
| 5    | View extracted memories                   | `ov ls viking://user/memories/`                    |
| 6    | Search memories                           | `ov find "query" --type memory`                    |

**Commit Behavior**:
- `keep_recent_count=0` → Archive toàn bộ messages
- `keep_recent_count=N` → Giữ N messages gần nhất, archive phần còn lại
- Phase 2 chạy background: Working Memory v2 + memory extraction

### SOP-04: Multi-tenant Administration

| Step | Action                                    | Chi tiết                                           |
|------|-------------------------------------------|----------------------------------------------------|
| 1    | Configure auth mode                       | Set `server.auth_mode` + `server.root_api_key`     |
| 2    | Create account                            | `POST /api/v1/admin/accounts`                      |
| 3    | Create user keys                          | `POST /api/v1/admin/accounts/{id}/users/{uid}/keys`|
| 4    | Set namespace policy                      | Configure isolation policies per account           |
| 5    | Monitor                                   | Prometheus `/metrics` + Grafana dashboards         |

**Auth Modes**:
- `dev` — No auth, localhost only, ROOT by default
- `api_key` — Root key + per-user keys, full RBAC
- `trusted` — Trust gateway headers, optional root key

### SOP-05: Encryption Configuration

| Step | Action                                    | Chi tiết                                           |
|------|-------------------------------------------|----------------------------------------------------|
| 1    | Choose KMS provider                       | Local file, HashiCorp Vault, hoặc Volcengine KMS   |
| 2    | Enable encryption                         | Set `encryption.enabled = true` in ov.conf         |
| 3    | Configure provider                        | Provider-specific settings (key path, Vault URL)   |
| 4    | (Optional) Enable API key hashing         | `encryption.api_key_hashing.enabled = true`        |
| 5    | Restart server                            | `openviking-server`                                |

### SOP-06: IDE Plugin Setup (Claude Code)

| Step | Action                                    | Chi tiết                                           |
|------|-------------------------------------------|----------------------------------------------------|
| 1    | Start OpenViking server                   | `openviking-server`                                |
| 2    | Install plugin                            | Follow `examples/claude-code-memory-plugin` guide  |
| 3    | Configure MCP connection                  | Point to `http://localhost:1933/mcp`               |
| 4    | Set identity headers                      | `X-OpenViking-Account`, `X-OpenViking-User`        |
| 5    | Test                                      | Say "search for X" in IDE → should trigger MCP tool|

---

## 4. Non-Functional Requirements

### 4.1 Performance

| Requirement                  | Target                      |
|-----------------------------|-----------------------------|
| API response time (p50)     | < 100ms (filesystem ops)    |
| Semantic search latency     | < 500ms (with rerank)       |
| Resource ingestion (small)  | < 30s per file              |
| Session commit Phase 1      | < 1s (lock-protected)       |
| Concurrent sessions         | ≥ 1,000                     |
| Embedding throughput        | ≥ 100 requests/s            |

### 4.2 Reliability

| Requirement                  | Target                      |
|-----------------------------|-----------------------------|
| API uptime                  | ≥ 99.9%                     |
| Data durability             | WAL + redo-log protection   |
| Graceful shutdown           | Clean resource release      |
| Lock manager                | Distributed filesystem locks|
| Error recovery              | Phase 2 redo-log replay     |

### 4.3 Scalability

| Requirement                  | Target                      |
|-----------------------------|-----------------------------|
| Multi-worker                | Uvicorn `--workers N`       |
| Storage scalability         | RAGFS-based, disk-bound     |
| Embedding concurrency       | Configurable `max_concurrent`|
| VLM concurrency             | Configurable `max_concurrent`|

### 4.4 Security

| Requirement                  | Implementation               |
|-----------------------------|------------------------------|
| Authentication              | API Key / Trusted gateway    |
| Authorization               | Role-based (ROOT/ADMIN/USER) |
| Encryption at-rest          | AES-256-GCM envelope        |
| Key management              | Local / Vault / Cloud KMS    |
| Namespace isolation         | URI-based account/user scope |
| Network binding             | Dev mode restricted to localhost|
| API key protection          | Optional Argon2id hashing    |

### 4.5 Usability

| Requirement                  | Implementation               |
|-----------------------------|------------------------------|
| Setup wizard                | `openviking-server init`     |
| Health check                | `openviking-server doctor`   |
| Error messages              | Structured JSON with codes   |
| Documentation               | Multi-language (EN/CN/JA)    |
| Examples                    | Quick start, plugins, K8s    |

---

## 5. Acceptance Criteria

### 5.1 Core Functionality

- [ ] Server starts successfully with `openviking-server`
- [ ] `ov status` returns healthy state
- [ ] Resource ingestion processes git repos and URLs
- [ ] `ov find` returns semantically relevant results
- [ ] Session commit extracts memories to user/agent directories
- [ ] MCP endpoint responds to all 9 tools
- [ ] Multi-tenant isolation prevents cross-account data access

### 5.2 Integration

- [ ] Claude Code plugin connects via MCP and executes tools
- [ ] Python SDK (sync/async) performs CRUD operations
- [ ] Rust CLI performs all filesystem operations
- [ ] Docker container starts with health check passing
- [ ] Prometheus metrics endpoint exposes operational data

### 5.3 Security

- [ ] Dev mode rejects non-localhost connections
- [ ] API key authentication validates before data access
- [ ] Encryption at-rest encrypts/decrypts files correctly
- [ ] RBAC prevents unauthorized role escalation
- [ ] Namespace isolation blocks cross-user data access
