# Product Requirements Document (PRD)

## OpenViking — Context Database for AI Agents

| Field           | Value                                     |
| --------------- | ----------------------------------------- |
| **Product**     | OpenViking                                |
| **Version**     | 0.1.x (Alpha)                             |
| **Author**      | ByteDance / Volcengine                    |
| **License**     | AGPL-3.0 (Main) · Apache 2.0 (CLI/Examples) |
| **Last Updated**| 2026-05-07                                |

---

## 1. Executive Summary

**OpenViking** là một **Context Database** mã nguồn mở được thiết kế chuyên biệt cho AI Agents. Hệ thống giải quyết 5 thách thức cốt lõi trong phát triển Agent: phân mảnh context, nhu cầu context tăng vọt, hiệu quả truy xuất kém, context không quan sát được, và hạn chế trong lặp lại bộ nhớ.

OpenViking đổi mới bằng cách áp dụng **"paradigm hệ thống tệp"** (filesystem paradigm) — giao thức `viking://` — để tổ chức thống nhất memories, resources và skills của Agent, thay thế mô hình lưu trữ vector phẳng truyền thống.

---

## 2. Problem Statement

### 2.1 Thách thức trong phát triển Agent

| # | Thách thức                  | Mô tả                                                                 |
|---|----------------------------|------------------------------------------------------------------------|
| 1 | **Fragmented Context**     | Memories nằm trong code, resources trong vector DB, skills phân tán    |
| 2 | **Surging Context Demand** | Agent chạy lâu tạo context liên tục; cắt/nén gây mất thông tin        |
| 3 | **Poor Retrieval**         | RAG truyền thống sử dụng flat storage, thiếu global view               |
| 4 | **Unobservable Context**   | Chuỗi truy xuất ngầm như hộp đen, khó debug                           |
| 5 | **Limited Memory Iteration** | Bộ nhớ chỉ ghi nhận tương tác, thiếu task memory của Agent            |

### 2.2 Target Users

- **AI Agent Developers** — xây dựng Agent thông minh có context management
- **LLM Application Teams** — tích hợp long-term memory vào sản phẩm AI
- **Enterprise AI Platform** — triển khai multi-tenant context infrastructure
- **AI IDE Plugins** — Claude Code, OpenCode, Codex context plugins

---

## 3. Product Vision

> **"Data in, Context out."** — Biến dữ liệu thô thành context có cấu trúc phân tầng, cho phép AI Agent truy xuất chính xác theo nhu cầu.

### 3.1 Giải pháp 5 trụ cột

| Trụ cột                           | Giải quyết        | Cơ chế                                            |
|-----------------------------------|--------------------|---------------------------------------------------|
| Filesystem Management Paradigm    | Fragmentation      | `viking://` URI, cấu trúc thư mục ảo              |
| Tiered Context Loading (L0/L1/L2) | Token Consumption   | Abstract → Overview → Detail, load on demand       |
| Directory Recursive Retrieval     | Retrieval Quality   | Hierarchical search + semantic + rerank            |
| Visualized Retrieval Trajectory   | Observability       | Quỹ đạo truy xuất có thể quan sát và tối ưu       |
| Automatic Session Management      | Memory Iteration    | Auto-compress, extract long-term memory            |

---

## 4. Core Features

### 4.1 Virtual Filesystem (VikingFS)

**Mô tả**: Hệ thống tệp ảo thống nhất, tổ chức mọi loại context dưới giao thức `viking://`.

**Cấu trúc namespace**:

```
viking://
├── resources/          # Project docs, repos, web pages
├── user/               # User preferences, habits, memories
│   └── {user_id}/
│       ├── memories/
│       └── privacy/
├── agent/              # Agent skills, instructions, task memories
│   └── {agent_id}/
│       ├── skills/
│       ├── memories/
│       └── instructions/
└── session/            # Active session data
    └── {session_id}/
```

**Thao tác cốt lõi**: `ls`, `tree`, `read`, `write`, `mkdir`, `rm`, `find`, `grep`, `glob`, `stat`, `mv`, `cp`

### 4.2 Three-Tier Context System (L0/L1/L2)

| Layer | Tên       | Token Budget | Mục đích                                    |
|-------|-----------|--------------|---------------------------------------------|
| L0    | Abstract  | ~100 tokens  | Quick relevance check, one-sentence summary |
| L1    | Overview  | ~2K tokens   | Core info + usage scenarios for planning    |
| L2    | Detail    | Full content | Deep reading khi cần thiết                  |

**File mapping**: `.abstract.md` (L0), `.overview.md` (L1), raw content (L2)

### 4.3 Hierarchical Retrieval Engine

**Pipeline 5 bước**:

1. **Intent Analysis** → Phân tích query thành nhiều retrieval conditions
2. **Global Vector Search** → Dense + Sparse vector, định vị high-score directories
3. **Directory Recursive Search** → Drill-down vào subdirectories, score propagation
4. **Rerank** → Optional model-based reranking (Volcengine, OpenAI, Cohere, Jina, etc.)
5. **Result Aggregation** → Hotness score blending, convergence detection

**Thuật toán nổi bật**:
- Score Propagation: `final_score = α × child_score + (1-α) × parent_score`
- Hotness Boost: `blended = (1-α_hot) × semantic + α_hot × hotness`
- Convergence: Dừng sau 3 rounds topK không đổi

### 4.4 Session Management & Memory Extraction

**Session Lifecycle**:

1. **Create** → Session directory + messages.jsonl
2. **Add Messages** → Append JSONL, track participants, token accounting
3. **Commit (2-Phase)**:
   - **Phase 1** (Lock-protected): Archive messages, write retained tail
   - **Phase 2** (Background): Memory extraction, Working Memory v2 update
4. **Working Memory v2** → 7-section structured document (Session Title, Current State, Task & Goals, Key Facts, Files & Context, Errors, Open Issues)

**Memory Categories**: profile, preferences, entities, events, cases, patterns, tools, skills

### 4.5 Resource Ingestion Pipeline

**Supported Sources**:
- Git repositories (clone + tree-sitter parsing)
- HTTP/HTTPS URLs (web scraping + markdown conversion)
- Local files/directories (via CLI)
- Documents: PDF, DOCX, PPTX, XLSX, EPUB

**Processing Pipeline**:
1. Source detection & download
2. File parsing (VLM-assisted for images/complex docs)
3. Tree building (directory structure)
4. Chunking & embedding (dense + sparse vectors)
5. L0/L1 summary generation (via VLM)
6. Vector index upsert

**Watch Resource**: Auto-refresh resources on schedule

### 4.6 VikingBot — Agent Framework

**Mô tả**: AI Agent framework tích hợp sẵn, chạy như companion service.

**Features**:
- Multi-channel bot (Telegram, Feishu/Lark, DingTalk, Slack, QQ, Discord)
- Tool-use agent with skill execution
- Sandbox integration (code execution)
- FUSE mount (filesystem mount)
- Web Console UI (Gradio-based)
- MCP server integration

### 4.7 Security & Multi-tenancy

| Feature                | Chi tiết                                                          |
|------------------------|-------------------------------------------------------------------|
| **Authentication**     | 3 modes: Dev (localhost only), API Key, Trusted (gateway-backed)  |
| **RBAC**               | 4 roles: ROOT → ADMIN → USER → BOT                               |
| **Namespace Isolation** | Account → User → Agent scoping, configurable isolation policies  |
| **Encryption**         | Envelope encryption: AES-256-GCM per file, multi-provider KMS    |
| **API Key Management** | Root key + per-account/user keys, optional Argon2id hashing       |
| **Privacy Controls**   | User privacy config service with version history                  |

### 4.8 MCP (Model Context Protocol) Endpoint

**9 Tools** exposed qua Streamable HTTP tại `/mcp`:

| Tool           | Mô tả                                     |
|----------------|--------------------------------------------|
| `search`       | Semantic search (memories, resources, skills)|
| `read`         | Read full content from URI(s)               |
| `list`         | Directory listing                           |
| `store`        | Store messages → memory extraction          |
| `add_resource` | Add remote resource (HTTP/git)              |
| `grep`         | Regex pattern matching in files             |
| `glob`         | Filename glob matching                      |
| `forget`       | Delete URI permanently                      |
| `health`       | Server health check                         |

---

## 5. Technical Architecture

### 5.1 Technology Stack

| Layer            | Technology                                                      |
|------------------|-----------------------------------------------------------------|
| **Language**     | Python 3.10+ (core), Rust (CLI + RAGFS), C++ (vector engine)   |
| **API Framework**| FastAPI + Uvicorn (multi-worker)                                |
| **Storage**      | RAGFS (custom filesystem), embedded vector DB                   |
| **Embedding**    | 12+ providers: OpenAI, Volcengine, Gemini, Jina, Cohere, etc.  |
| **VLM**          | OpenAI, Volcengine, Codex, Kimi, GLM                           |
| **Observability**| OpenTelemetry (traces + logs), Prometheus metrics               |
| **Crypto**       | AES-256-GCM, Argon2id, multi-provider KMS                      |
| **Container**    | Docker (multi-stage), Kubernetes (Helm chart)                   |

### 5.2 Module Architecture

```
┌─────────────────────────────────────────────────────┐
│                  FastAPI HTTP Server                 │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────┐ │
│  │ Routers  │ │ Auth/RBAC │ │ MCP /mcp │ │WebDAV│ │
│  └──────────┘ └───────────┘ └──────────┘ └──────┘ │
├─────────────────────────────────────────────────────┤
│               Service Layer (core.py)               │
│  ┌────────────┐ ┌────────────┐ ┌──────────────┐   │
│  │FSService   │ │SearchService│ │SessionService│   │
│  │ResourceSvc │ │RelationSvc  │ │DebugService  │   │
│  │PackService │ │TaskTracker  │ │              │   │
│  └────────────┘ └────────────┘ └──────────────┘   │
├─────────────────────────────────────────────────────┤
│               Core Domain Layer                     │
│  ┌──────────┐ ┌──────────┐ ┌────────────────────┐ │
│  │Context   │ │Namespace │ │DirectoryInitializer│ │
│  │ContextType│ │URI Resolve│ │SkillLoader        │ │
│  │ContextLevel│ │Accessibility│                    │ │
│  └──────────┘ └──────────┘ └────────────────────┘ │
├─────────────────────────────────────────────────────┤
│               Infrastructure Layer                  │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌────────┐ │
│  │VikingFS  │ │VikingDB  │ │RAGFS   │ │Embedder│ │
│  │QueueMgr  │ │VectorIdx │ │LockMgr │ │VLM     │ │
│  │Encryptor │ │Observers │ │Rerank  │ │Parsers │ │
│  └──────────┘ └──────────┘ └────────┘ └────────┘ │
└─────────────────────────────────────────────────────┘
```

---

## 6. Deployment Models

| Model               | Mô tả                                    | Port        |
|---------------------|-------------------------------------------|-------------|
| **Standalone**       | `openviking-server` single process        | 1933 (API)  |
| **With Bot**         | `--with-bot` starts VikingBot companion    | 1933 + 18790|
| **Docker**           | `ghcr.io/volcengine/openviking:latest`    | 1933, 8020  |
| **Multi-worker**     | `--workers N` uvicorn multi-process       | 1933        |
| **Kubernetes**       | Helm chart (`examples/k8s-helm`)          | Configurable|
| **Cloud (ECS)**      | Volcengine ECS + veLinux deployment       | Configurable|

---

## 7. Integration Points

### 7.1 IDE Plugin Ecosystem

| Plugin                  | Giao thức    | Repository                              |
|-------------------------|--------------|-----------------------------------------|
| Claude Code Memory      | MCP / CLI    | `examples/claude-code-memory-plugin`    |
| OpenCode Memory         | MCP / CLI    | `examples/opencode-memory-plugin`       |
| Codex Memory            | MCP          | `examples/codex-memory-plugin`          |
| OpenClaw Context        | Plugin API   | `examples/openclaw-plugin`              |

### 7.2 Client Libraries

| Client              | Type   | Entry Point                       |
|----------------------|--------|-----------------------------------|
| `SyncOpenViking`     | Python | `openviking.sync_client`          |
| `AsyncOpenViking`    | Python | `openviking.async_client`         |
| `SyncHTTPClient`     | Python | `openviking_cli.client.sync_http` |
| `AsyncHTTPClient`    | Python | `openviking_cli.client.http`      |
| `ov` CLI             | Rust   | `crates/ov_cli`                   |

---

## 8. Performance Benchmarks

### 8.1 OpenClaw Context Plugin Results

| Experimental Group                          | Task Completion | Input Tokens    |
|---------------------------------------------|-----------------|-----------------|
| OpenClaw (memory-core)                      | 35.65%          | 24,611,530      |
| OpenClaw + LanceDB (-memory-core)           | 44.55%          | 51,574,530      |
| **OpenClaw + OpenViking (-memory-core)**     | **52.08%**      | **4,264,396**   |
| **OpenClaw + OpenViking (+memory-core)**     | **51.23%**      | **2,099,622**   |

**Key Results**:
- **+49% task completion** vs vanilla OpenClaw
- **83-91% reduction** in input token cost
- **+17% task completion** vs LanceDB with **92-96% token reduction**

---

## 9. Roadmap Indicators

| Area                  | Status      | Notes                                     |
|-----------------------|-------------|-------------------------------------------|
| Core VikingFS         | ✅ Stable    | Full CRUD, ACL, encryption                |
| Hierarchical Retrieval| ✅ Stable    | Dense + sparse + rerank                   |
| Session/Memory        | ✅ Stable    | WM v2, 2-phase commit                     |
| Multi-tenancy         | ✅ Stable    | Account/User/Agent isolation              |
| MCP Endpoint          | ✅ Stable    | 9 tools via Streamable HTTP               |
| VikingBot             | 🟡 Beta      | Multi-channel, sandbox, FUSE              |
| Encryption            | ✅ Stable    | Envelope AES-GCM, multi-KMS              |
| Observability         | ✅ Stable    | OTel traces/logs/metrics, Prometheus      |
| WebDAV                | 🟡 Beta      | File manager access                       |
| Multi-modal           | 🔵 Planned   | Image/video/audio context                 |

---

## 10. Success Metrics

| Metric                          | Target                          |
|---------------------------------|---------------------------------|
| Task Completion Rate (vs RAG)   | ≥ 40% improvement               |
| Token Cost Reduction            | ≥ 80% reduction                 |
| Retrieval Latency (p95)         | < 500ms                         |
| Memory Extraction Accuracy      | ≥ 85% relevance                 |
| API Uptime                      | ≥ 99.9%                         |
| Concurrent Sessions             | ≥ 1,000 per instance            |
