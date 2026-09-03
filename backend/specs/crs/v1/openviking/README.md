# Change Requests — OpenViking Feature Parity

**Project:** VNP Memory  
**Domain:** OpenViking — Context Database cho AI Agents (ByteDance/Volcengine)  
**Path:** `specs/crs/v1/openviking/`  
**Updated:** 2026-06-16  
**Status:** Proposed

> Các Change Requests này được tạo từ phân tích đối chiếu giữa VNP Memory hiện tại và tài liệu OpenViking (`PRD v0.1.x`, `SRS`, `URD`, `specs/services/00-09`).
>
> **OpenViking Core Innovation:** Áp dụng **"Filesystem Paradigm"** — giao thức `viking://` — để tổ chức thống nhất memories, resources và skills của Agent, thay thế mô hình lưu trữ vector phẳng truyền thống.
>
> **Key Result (benchmark):** +49% task completion, 83-91% token cost reduction so với vanilla RAG.

---

## Danh sách Change Requests

| CR ID | Tên | Loại | Priority | Status |
|---|---|---|---|---|
| [CR-OV-001](./CR-OV-001-Gateway-Service.md) | **Unified Gateway** — REST (17 routes), MCP (9 tools), WebDAV, 3-mode Auth, RBAC (ROOT/ADMIN/USER/BOT), Rate Limit, Circuit Breaker | 🆕 New Service | Critical | Proposed |
| [CR-OV-002](./CR-OV-002-Filesystem-Service.md) | **Filesystem Service** — VikingFS Go-native, L0/L1/L2 tiered read, grep/glob parallel, Relations, Privacy Config, PathLock, transparent encryption, NATS events | 🆕 New Service | Critical | Proposed |
| [CR-OV-003](./CR-OV-003-Search-Service.md) | **Search Service** — HierarchicalRetriever 6 bước, score propagation (α=0.7), hotness blending (α=0.1), convergence detection (3 rounds), 5 reranker providers, event-driven index sync | 🆕 New Service | High | Proposed |
| [CR-OV-004](./CR-OV-004-Session-Service.md) | **Session Service** — Two-Phase Commit (lock-protected Phase 1, background Phase 2), Working Memory v2 (7 sections), 8 memory categories, redo log crash recovery | 🆕 New Service | High | Proposed |
| [CR-OV-005](./CR-OV-005-Resource-Service.md) | **Resource Service** — Ingestion pipeline (git/HTTP/local/doc), parser registry (50+ ext, tree-sitter), L0/L1 VLM generation, watch/auto-refresh, task tracking | 🆕 New Service | High | Proposed |
| [CR-OV-006](./CR-OV-006-Crypto-Admin-Services.md) | **Crypto & Admin** — OVE1 envelope encryption (AES-256-GCM, key hierarchy), KMS adapters (Local/Vault/Cloud), account/user/key CRUD, health aggregation, cascade events | 🆕 New Services | High | Proposed |
| [CR-OV-007](./CR-OV-007-Shared-Infrastructure.md) | **Shared `pkg/`** — viking domain types, adapter interfaces (12 embedding providers), Go VikingFS engine, middleware stack, resilience patterns, OTel observability, ov CLI, Python SDK | 🆕 New Packages | High | Proposed |

---

## Kiến trúc Golang Microservices mục tiêu

```
External Consumers:
  ov CLI (Rust/Go) · Python SDK (sync/async) · MCP Clients (Claude/OpenCode)
  WebDAV (file manager) · VikingBot (Telegram/Feishu/Slack)
                                    │
           ┌─────────────────────────▼──────────────────────────┐
           │             openviking-gateway                       │
           │  REST (17 routes) · MCP (9 tools) · WebDAV          │
           │  Auth (DEV/API_KEY/TRUSTED) · RateLimit · CORS      │
           │  Circuit Breaker · Request Validation                │
           │  Port: 8080 (REST) · 8081 (gRPC) · 8082 (MCP)      │
           └──┬──────┬──────┬──────┬──────┬──────┬──────────────┘
              │      │      │      │      │      │  gRPC (internal)
     ┌────────┘      │      │      │      │      └────────────┐
     ▼               ▼      ▼      ▼      ▼                   ▼
┌─────────┐ ┌──────────┐ ┌──────┐ ┌────────┐ ┌────────┐ ┌────────┐
│   fs    │ │  search  │ │sess. │ │resource│ │ crypto │ │ admin  │
│ :9011   │ │  :9012   │ │:9013 │ │ :9014  │ │ :9015  │ │ :9030  │
│         │ │          │ │      │ │        │ │        │ │        │
│VikingFS │ │Hierarchi.│ │2-Ph. │ │Ingest  │ │OVE1    │ │Account │
│L0/L1/L2 │ │Retrieval │ │Commit│ │Pipeline│ │Encrypt.│ │API Key │
│grep/glob│ │6-step    │ │WM v2 │ │50+     │ │KMS     │ │Health  │
│PathLock │ │ScoreProp.│ │8 Cat.│ │parsers │ │Rotate  │ │Stats   │
│Encrypt  │ │Hotness   │ │Redo  │ │Watch   │ │        │ │        │
└────┬────┘ └──────────┘ └──────┘ └────────┘ └────────┘ └────────┘
     │                      │         │
     └──────────────────────▼─────────▼──────────────────────┐
                    NATS JetStream (Async Event Bus)           │
           ┌─────────────────────────────────────────────────┐ │
           │   ov.content.written → Search indexes           │ │
           │   ov.session.committed → Search updates hotness │ │
           │   ov.resource.ingested → Search reindexes       │ │
           │   ov.crypto.key.rotated → FS re-wraps           │ │
           │   admin.account.* → All services cascade        │ │
           └─────────────────────────────────────────────────┘ │
                                                                │
        ┌───────────────────────────────────────────────────────┘
        │         SHARED INFRASTRUCTURE (pkg/)
        │  VikingFS · VectorDB (Qdrant/Weaviate) · Redis
        │  NATS JetStream · Bifrost (LLM/Embed gateway)
        │  OTel Collector · KMS (Local/Vault/Cloud)
        └───────────────────────────────────────────────────────
```

---

## 5 Tính năng Cốt lõi Độc đáo của OpenViking

| # | Feature | Mô tả | CR |
|---|---------|-------|---|
| 1 | **Viking URI** | `viking://` protocol thống nhất memories, resources, skills, sessions dưới một cây thư mục. Không còn data rác không xác định vị trí | CR-OV-002 |
| 2 | **Tiered Context (L0/L1/L2)** | Token budget control: Abstract (~100 tokens) → Overview (~2K tokens) → Full Content. Load on demand, không cần dump toàn bộ context | CR-OV-002 |
| 3 | **Hierarchical Search** | 6-step recursive retrieval theo cấu trúc thư mục, score propagation α=0.7, hotness blending α=0.1, convergence detection sau 3 rounds ổn định | CR-OV-003 |
| 4 | **Working Memory v2** | Session commit tự động nén lịch sử thành 7-section structured Markdown + extract 8 loại memories via VLM. Two-Phase Commit đảm bảo crash safety | CR-OV-004 |
| 5 | **OVE1 Transparent Encryption** | Per-file AES-256-GCM envelope encryption, hoàn toàn trong suốt với caller. Key rotation chỉ re-wrap File Keys, không re-encrypt content | CR-OV-006 |

---

## Performance Targets (từ URD)

| Metric | Target |
|--------|--------|
| API response (p50, filesystem ops) | < 100ms |
| Semantic search latency (with rerank) | < 500ms |
| Session commit Phase 1 | < 1s |
| Concurrent sessions per instance | ≥ 1,000 |
| Token cost reduction vs RAG | ≥ 80% |
| Task completion vs RAG | ≥ 40% improvement |

---

## Infrastructure mới cần thêm

> [!IMPORTANT]
> OpenViking cần thêm các components này vào infrastructure của VNP Memory:

| Component | Mục đích |
|-----------|---------|
| **Qdrant / Weaviate** | Vector database cho hierarchical search |
| **Bifrost gateway** | Unified LLM/Embed proxy (12+ providers) |
| **NATS JetStream** | Async event bus (thay thế hoặc bổ sung) |
| **tree-sitter** | Go bindings cho code parsing (smacker/go-tree-sitter) |

---

## Lộ trình triển khai

| Wave | CR | Mô tả |
|------|-----|-------|
| **Wave 1 — Foundation** | CR-007 | `pkg/` shared types, adapters, vikingfs, middleware, observability |
| **Wave 2 — Security** | CR-006 | Crypto (OVE1 encryption) + Admin (account/key management) |
| **Wave 3 — Storage** | CR-002 | Filesystem service (VikingFS, L0/L1/L2, grep/glob, PathLock) |
| **Wave 4 — Search** | CR-003 | Search service (HierarchicalRetriever, score propagation, hotness) |
| **Wave 5 — Context** | CR-004, CR-005 | Session (WM v2, memory extraction) + Resource (ingestion pipeline) |
| **Wave 6 — Gateway** | CR-001 | Unified gateway (REST 17 routes, MCP 9 tools, WebDAV, auth) |
