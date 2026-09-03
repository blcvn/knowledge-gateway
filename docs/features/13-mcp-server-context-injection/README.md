# Feature 13 — MCP Server & Context Injection

> **Loại:** Integration | **Priority:** High | **Status:** Implemented (CR-AM-008, CR-GR-006, CR-MB-006)

## Mô tả

MCP Server (Model Context Protocol) là interface chính để AI Assistants (Claude Code, Claude Desktop, AutoGen...) tương tác với VNP Memory. MCP expose memory operations dưới dạng **tools** mà AI có thể gọi trực tiếp trong conversation.

**Context Injection** là cơ chế tự động inject relevant memory context vào LLM prompt trước khi gửi — giảm token cost và tăng context quality.

---

## Business Logic

### MCP Protocol

MCP (Model Context Protocol, version `2024-11-05`) là standard protocol do Anthropic phát triển cho AI tool use:
- Transport: **SSE** (Server-Sent Events) và **HTTP Streamable**
- Protocol: **JSON-RPC 2.0**
- Methods: `initialize`, `tools/list`, `tools/call`, `ping`

VNP Memory expose MCP server trên port `:8082`.

### 37+ MCP Tools

VNP Memory expose 37+ tools chia theo domains:

**Memory Tools** (core):
- `memory_store` — Lưu memory (auto-route by type)
- `memory_recall` — Cross-engine recall
- `memory_search` — Search specific engine
- `memory_timeline` — Temporal events
- `memory_profile` — User profile context
- `memory_forget` — Delete memory

**Graph Tools**:
- `graph_query` — Query Graphiti temporal graph

**OpenViking Tools** (filesystem):
- `ov_read_file` — Read file từ VikingFS
- `ov_write_file` — Write file to VikingFS
- `ov_search` — Search với tiered context
- `ov_list_dir` — List directory
- `ov_grep` — Grep in files
- `ov_tree` — Directory tree
- `ov_session_commit` — Commit working session
- `ov_ingest` — Ingest external resources
- `ov_delete` — Delete file

**AgentMemory Tools** (observe + memory + orchestration):
- `observe_*` — Session observation
- `agent_remember` — Store typed agent memory
- `agent_recall` — Recall with hybrid search
- `agent_slots` — Read/write memory slots
- `orchestrate_lease` — Request distributed lease
- `orchestrate_signal` — Send inter-agent signal
- ...và nhiều tools khác

### Context Injection

Context Injection là middleware layer tự động inject memory vào LLM call:

1. **Pre-call Hook**: Trước khi AI gọi LLM, context injector query memory để tìm relevant context.
2. **Token Budget**: Configurable token budget (e.g., 1000 tokens). Injector chọn context có priority cao nhất trong budget.
3. **Sources**: Context có thể đến từ:
   - Memobase profile (user preferences/facts)
   - OpenViking tiered context (project files)
   - Supermemory adaptive KG (relevant knowledge)
4. **Post-call Hook**: Sau LLM response, injector có thể capture response để store vào memory.

### Agent Scoping

Mỗi MCP session có **scope** xác định:
- `isolated`: Memory chỉ visible với agent session này.
- `shared`: Memory shared across agents trong same tenant.
- `project`: Memory scoped tới project namespace.

---

## Dataflow

### MCP Connection & Tool Call

```
AI Assistant (Claude Code / Claude Desktop)
        │
        ├── 1. Connect: GET /mcp/sse  (SSE transport)
        │         OR:  GET /sse
        │
        ├── 2. Initialize:
        │         POST /mcp/message {"method": "initialize", "params": {...}}
        │         ← {"result": {"serverInfo": {...}, "capabilities": {...}}}
        │
        ├── 3. List Tools:
        │         POST /mcp/message {"method": "tools/list"}
        │         ← {"result": {"tools": [37+ tool definitions]}}
        │
        └── 4. Call Tool:
                  POST /mcp/message {
                    "method": "tools/call",
                    "params": {
                      "name": "memory_store",
                      "arguments": {"content": "...", "type": "fact"}
                    }
                  }
                          │
                          ▼
                  vnp-gateway (MCP server)
                          │
                          ├── Route to: memory.Store → engine
                          └── Return result → SSE stream → AI
```

### Context Injection Flow

```
AI Agent about to call LLM
        │
        ├── Context Injector intercepts (pre-hook)
        │
        ▼
Context Assembly
        │
        ├── Query 1: Memobase profile (< 100ms)
        │         └── User preferences, facts, goals
        │
        ├── Query 2: OpenViking current context (tiered L0/L1)
        │         └── Relevant project files, session state
        │
        ├── Query 3: Supermemory relevant memories
        │         └── Knowledge about current topic
        │
        ├── Token Budget Management
        │         ├── Rank context by relevance score
        │         └── Fill budget: highest priority first
        │
        └── Inject into system prompt:
                  "[MEMORY CONTEXT]
                   User: prefers TypeScript, uses VSCode
                   Project: uses Clean Architecture
                   Known issue: API X has rate limit 100/min
                   [/MEMORY CONTEXT]"
                          │
                          ▼
                  LLM Call (enriched prompt)
                          │
                          ▼
                  Post-hook: capture response → store to memory
```

---

## MCP Endpoints

| Method | Path | Transport | Mô tả |
|--------|------|-----------|-------|
| `GET` | `/mcp/sse` | SSE | SSE transport connection |
| `GET` | `/sse` | SSE | Alternate SSE endpoint |
| `POST` | `/mcp/message` | HTTP | JSON-RPC 2.0 messages |
| `POST` | `/message` | HTTP | Alternate HTTP endpoint |

---

## Services

| Service | Vai trò |
|---------|---------|
| `gateway/adapter/mcp/server.go` | MCP server implementation |
| `sm-mcp` | Supermemory-specific MCP tools |
| All engine services | Backend for each MCP tool |

---

## Business Value

### Pain Points được giải quyết

- **PP-P6-03 (Manual context injection)**
- **PP-P1-06 (Context tốn token)**

### Actors hưởng lợi

P1 AI Agent Developer, P5 IDE Plugin User, P6 Framework Integrator

### Giải pháp tham chiếu

- [S2 — Unified Memory API](../../bussiness/solutions/S2-unified-api.md)
- [S6 — Smart Context Assembly](../../bussiness/solutions/S6-context-efficiency.md)

### ROI / Kết quả đo được

> 37+ tools (vs 16 ban đầu) | Token budget management | 3 agent scopes

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
