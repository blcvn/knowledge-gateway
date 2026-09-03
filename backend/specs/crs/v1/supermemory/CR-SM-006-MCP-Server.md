# Change Request: CR-SM-006 — MCP Server (Model Context Protocol)

**CR ID:** CR-SM-006  
**Component:** `services/mcp-service` [NEW SERVICE]  
**Priority:** High  
**Status:** In Progress
**Reference:** Supermemory PRD §3.6, SRS §2.7, specs/services/07-mcp-service.md  
**Compatibility:** Claude Desktop, Cursor, Windsurf, VS Code, Claude Code, OpenCode, Hermes

---

## 1. Mô tả

Xây dựng **MCP Service** — Model Context Protocol server giúp tích hợp VNP Memory với mọi AI client hỗ trợ MCP:

1. **MCP Tools**: `memory` (save/forget), `recall` (search + profile), `listProjects`, `whoAmI`.
2. **MCP Resources**: `supermemory://profile` (user profile data), `supermemory://projects`.
3. **MCP Prompt**: `context` (inject system context với user profile vào conversation).
4. **Session Management**: Persistent session qua Redis (keyed by user_id), auto-refresh container tags.
5. **MCP Client Info Capture**: Capture client name/version (`claude-desktop`, `cursor`, `vscode`).
6. **OAuth2 for MCP**: OAuth flow đầy đủ để MCP clients authenticate.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện có MCP cơ bản nhưng thiếu tools quan trọng như `recall`, `listProjects`, `whoAmI`.
- Thiếu MCP Resources và Prompts.
- Session management chưa đủ robust (không persist qua reconnect).
- Chưa capture client info → không biết users đang dùng Claude hay Cursor.

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] `services/mcp-service/` (Port: via Gateway `/mcp`, gRPC: 9006)

### 3.2. Registered MCP Tools

| Tool | Input | Mô tả |
|------|-------|-------|
| `memory` | `{action: save\|forget, content, containerTag?}` | Lưu hoặc quên thông tin (max 200K chars) |
| `recall` | `{query, containerTag?, includeProfile?}` | Search memories + lấy user profile |
| `listProjects` | `{}` | Liệt kê projects/spaces (cache TTL 5 phút) |
| `whoAmI` | `{}` | Thông tin user đang đăng nhập + client info |

### 3.3. Registered MCP Resources

| Resource URI | Mô tả |
|-------------|-------|
| `supermemory://profile` | Static + Dynamic profile của user |
| `supermemory://projects` | Danh sách projects |

### 3.4. Registered MCP Prompts

| Prompt | Mô tả |
|--------|-------|
| `context` | Inject system context: profile + instructions để AI nhớ/quên |

**Context prompt example:**
```
<user_profile>
Long-term facts: User is a Go backend developer. Prefers clean architecture.
Current activities: Working on VNP Memory microservices project.
</user_profile>

When you learn new information about the user, save it using the memory tool.
When asked to forget something, use memory with action=forget.
```

### 3.5. Session Model

```go
type MCPSession struct {
    SessionID          string
    UserID             string
    OrgID              string
    APIKey             string
    Email              *string
    Name               *string
    RootContainerTag   *string     // From x-sm-project header
    CachedContainerTags []string   // TTL 5 phút
    ClientInfo         *MCPClientInfo
}

type MCPClientInfo struct {
    Name    string  // "claude-desktop" | "cursor" | "vscode"
    Version string
}
// Session stored in Redis, keyed by user_id (persist across reconnects)
```

### 3.6. `recall` Tool — Parallel Execution

```go
// Chạy đồng thời search + profile để giảm latency
var wg sync.WaitGroup
go func() { searchResult = searchService.MemorySearch(ctx, query) }()
go func() { profile = profileService.GetProfile(ctx, containerTag) }()
wg.Wait()
// Dedup: profile facts có priority cao hơn search results
response = mergeDedup(profile, searchResult)
```

### 3.7. Install Command (One-liner)

Hỗ trợ cài đặt 1 lệnh cho mỗi client:
```bash
npx -y install-mcp@latest vnp-memory --client claude
npx -y install-mcp@latest vnp-memory --client cursor
```

---

## 4. Acceptance Criteria

- [ ] Kết nối Claude Desktop tới VNP Memory MCP → thấy 4 tools: `memory`, `recall`, `listProjects`, `whoAmI`.
- [ ] Gọi `memory` với `action=save` → data xuất hiện trong Document Service.
- [ ] Gọi `recall` → nhận cả profile (static/dynamic) VÀ search results trong 1 response.
- [ ] Gọi `/context` prompt → AI nhận system message với đầy đủ user context.
- [ ] Disconnect rồi reconnect MCP → session được restore (không phải login lại).
- [ ] `whoAmI` trả về đúng client name (ví dụ `"claude-desktop"`).
