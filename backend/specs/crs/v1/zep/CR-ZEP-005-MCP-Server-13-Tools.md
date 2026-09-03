# Change Request: CR-ZEP-005 — MCP Server (13 Read-Only Tools)

**CR ID:** CR-ZEP-005  
**Component:** `services/gateway` / `mcp-service` [UPGRADE]  
**Priority:** High  
**Status:** In Progress
**Reference:** Zep PRD §6.2 F7, SRS §5.2, specs/services/01-gateway.md  
**Compatibility:** Claude Desktop (stdio), Claude Code (HTTP Streamable), Cline, Cursor

---

## 1. Mô tả

Nâng cấp MCP Server của VNP Memory lên **13 read-only tools** đầy đủ theo chuẩn Zep, với dual transport:

1. **13 MCP Tools**: Đầy đủ từ `search_graph` đến `get_episode_mentions`.
2. **Dual Transport**: stdio (Claude Desktop/Cline) + HTTP Streamable 2025-03-26 spec (Claude Code).
3. **Read-Only By Design**: MCP chỉ read — write operations yêu cầu SDK/REST API.
4. **Graph Exploration Tools**: Traverse graph, inspect nodes/edges/episodes.
5. **Docker Deployment**: Support `docker run -e ZEP_API_KEY=... zep-mcp-server:latest`.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện có MCP server nhưng chưa đủ 13 tools của Zep spec.
- Chưa có `search_graph` với đầy đủ parameters (scope, reranker, filters, ratings).
- Thiếu graph exploration tools: `get_user_nodes`, `get_user_edges`, `get_episodes`, `get_node_edges`.
- Chưa support `get_episode_mentions` (biết nodes/edges nào xuất hiện trong episode).

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] Gateway MCP Server

### 3.2. Complete 13 MCP Tools

**Core Search (Primary context retrieval):**

| Tool | Input Parameters | Output |
|------|----------------|--------|
| `search_graph` | `user_id`, `query`, `scope`, `limit`, `reranker`, `node_labels[]`, `edge_types[]`, `min_fact_rating`, `mmr_lambda`, `center_node_uuid` | JSON search results với temporal facts |
| `get_user_context` | `thread_id`, `template_id?` | Pre-formatted context string cho LLM |
| `get_user` | `user_id` | User metadata JSON |
| `list_threads` | `user_id` | Danh sách threads của user |

**Graph Query (Graph exploration):**

| Tool | Input Parameters | Output |
|------|----------------|--------|
| `get_user_nodes` | `user_id`, `limit?` | All entity nodes trong user's graph |
| `get_user_edges` | `user_id`, `limit?` | All temporal edges (facts) trong user's graph |
| `get_episodes` | `user_id`, `lastn?` | Recent episodes (temporal events) |

**Detail Retrieval (Specific entity lookup):**

| Tool | Input Parameters | Output |
|------|----------------|--------|
| `get_thread_messages` | `thread_id`, `limit?` | Messages của một thread |
| `get_node` | `node_uuid` | Chi tiết một entity node |
| `get_edge` | `edge_uuid` | Chi tiết một temporal edge (fact) |
| `get_episode` | `episode_uuid` | Chi tiết một episode |
| `get_node_edges` | `node_uuid` | Tất cả edges kết nối với node |
| `get_episode_mentions` | `episode_uuid` | Nodes và edges được mention trong episode |

### 3.3. Transport Modes

```
Mode 1 — stdio (Claude Desktop, Cline):
  $ zep-mcp-server --stdio
  Claude Desktop config:
    {"command": "/path/to/zep-mcp-server", "args": ["--stdio"], "env": {"ZEP_API_KEY": "..."}}

Mode 2 — HTTP Streamable (Claude Code, HTTP MCP clients):
  $ zep-mcp-server --port 8080
  # MCP Streamable HTTP spec 2025-03-26
  claude mcp add --transport http zep http://localhost:8080
```

### 3.4. Tool: `search_graph` (Flagship)

```go
type SearchGraphInput struct {
    UserID         string   // required
    Query          string   // required
    Scope          string   // "edges"|"nodes"|"episodes"|"all"
    Limit          int      // default: 10
    Reranker       string   // "rrf"|"mmr"|"cross_encoder"|"node_distance"|"episode_mentions"
    NodeLabels     []string // filter by node types
    EdgeTypes      []string // filter by edge relationship types
    MinFactRating  float64  // minimum quality threshold
    MmrLambda      float64  // MMR diversity parameter
    CenterNodeUUID string   // for node_distance reranker
}
```

### 3.5. Configuration

```yaml
# Environment variables
ZEP_API_KEY: required   # API key for VNP Memory
LOG_LEVEL: info         # debug|info|warn|error
PORT: 8080              # HTTP mode only
```

### 3.6. Docker Deployment

```yaml
# docker-compose.yml
services:
  vnp-memory-mcp:
    image: vnp-memory/mcp-server:latest
    ports: ["8080:8080"]
    environment:
      - VNP_MEMORY_API_KEY=${VNP_MEMORY_API_KEY}
      - LOG_LEVEL=info
```

---

## 4. Acceptance Criteria

- [ ] Claude Desktop kết nối MCP (stdio mode) → thấy đủ 13 tools trong tool list.
- [ ] Claude Code kết nối MCP (HTTP mode) → `search_graph` trả về temporal facts với valid_at/invalid_at.
- [ ] `get_user_nodes` → trả về classified entities (User, Preference, Organization...).
- [ ] `get_episode_mentions` → trả về đúng nodes và edges được mention trong episode.
- [ ] MCP server từ chối mọi write operations qua MCP (0 write tools).
- [ ] `get_user_context` trả về pre-formatted string ready for injection vào system prompt.
- [ ] Docker container start trong < 5 giây, health check đi qua.
