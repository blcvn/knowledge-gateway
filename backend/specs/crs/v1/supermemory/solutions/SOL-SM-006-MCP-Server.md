# Solution: SOL-SM-006 — MCP Server (Model Context Protocol)

**CR ID:** CR-SM-006  
**Solution ID:** SOL-SM-006  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp MCP Server hiện có trong VNP Memory (port `:8082`) để hỗ trợ đầy đủ 4 tools, 2 resources, 1 prompt, session persistence qua Redis, và OAuth2 authentication cho MCP clients. Tận dụng kiến trúc MCP SSE + HTTP Streamable đã có sẵn.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| MCP Server `:8082` | `gateway/adapter/mcp/` | Có: SSE + HTTP Streamable |
| 16 MCP tools | `gateway/adapter/mcp/` | Có: memory_store, memory_recall, ov_*, etc. |
| `sm-mcp` service | `apps/memory/internal/bootstrap/` | Có: sm-specific MCP tools |
| Redis | Infrastructure | Đã có cho cache/rate limit |

### Gap phân tích

- Thiếu tool `recall` (profile + search combo trong 1 call)
- Thiếu tool `listProjects` (container tags listing)
- Thiếu tool `whoAmI` (user info + client info)
- Thiếu MCP Resources (`supermemory://profile`, `supermemory://projects`)
- Thiếu MCP Prompt `context`
- Session không persist qua reconnect
- Chưa capture MCP client info (name/version)
- Chưa có OAuth2 for MCP

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service (Nâng cấp)

```
gateway/adapter/mcp/
├── server.go                 # MCP server registration (SSE + Streamable)
├── session.go                # MCPSession management (Redis)
├── auth.go                   # OAuth2 for MCP + API Key auth
├── tools/
│   ├── memory_tool.go        # NÂNG CẤP: save/forget, max 200K chars
│   ├── recall_tool.go        # MỚI: search + profile combo
│   ├── list_projects_tool.go # MỚI: liệt kê projects
│   └── whoami_tool.go        # MỚI: user + client info
├── resources/
│   ├── profile_resource.go   # MỚI: supermemory://profile
│   └── projects_resource.go  # MỚI: supermemory://projects
└── prompts/
    └── context_prompt.go     # MỚI: context system prompt
```

### 3.2. Session Model

```go
// gateway/adapter/mcp/session.go

package mcp

import (
    "encoding/json"
    "time"
)

type MCPSession struct {
    SessionID           string
    UserID              string
    OrgID               string
    APIKey              string
    Email               *string
    Name                *string
    RootContainerTag    *string      // From x-sm-project header
    CachedContainerTags []string     // TTL 5 phút
    ContainerTagsCachedAt *time.Time
    ClientInfo          *MCPClientInfo
    CreatedAt           time.Time
    LastActiveAt        time.Time
}

type MCPClientInfo struct {
    Name    string   // "claude-desktop" | "cursor" | "vscode" | "windsurf"
    Version string
}

// Redis session manager
type SessionManager struct {
    redis     RedisClient
    sessionTTL time.Duration // 24h default
}

func (sm *SessionManager) GetOrCreate(ctx context.Context, userID string) (*MCPSession, error) {
    key := fmt.Sprintf("mcp:session:%s", userID)
    data, err := sm.redis.Get(ctx, key).Bytes()

    if err == nil {
        // Restore existing session (persist qua reconnect)
        var session MCPSession
        if err := json.Unmarshal(data, &session); err == nil {
            session.LastActiveAt = time.Now()
            sm.save(ctx, &session)
            return &session, nil
        }
    }

    // Create new session
    session := &MCPSession{
        SessionID:    generateID(),
        UserID:       userID,
        CreatedAt:    time.Now(),
        LastActiveAt: time.Now(),
    }
    sm.save(ctx, session)
    return session, nil
}

func (sm *SessionManager) save(ctx context.Context, s *MCPSession) {
    key := fmt.Sprintf("mcp:session:%s", s.UserID)
    data, _ := json.Marshal(s)
    sm.redis.Set(ctx, key, data, sm.sessionTTL)
}

// Capture client info từ MCP Initialize request
func (sm *SessionManager) SetClientInfo(ctx context.Context, session *MCPSession, info MCPClientInfo) {
    session.ClientInfo = &info
    sm.save(ctx, session)
}
```

### 3.3. Tool Implementations

#### `memory` Tool (Nâng cấp)

```go
// gateway/adapter/mcp/tools/memory_tool.go

type MemoryToolInput struct {
    Action       string  `json:"action"`        // "save" | "forget"
    Content      string  `json:"content"`       // max 200,000 chars
    ContainerTag *string `json:"containerTag"`  // Optional space scoping
}

func (t *MemoryTool) Handle(ctx context.Context, session *MCPSession, input MemoryToolInput) (string, error) {
    containerTag := resolveContainerTag(session, input.ContainerTag)

    // Validate content length
    if len(input.Content) > 200_000 {
        return "", ErrContentTooLarge
    }

    switch input.Action {
    case "save":
        _, err := t.docClient.CreateDocument(ctx, CreateDocumentRequest{
            OrgID:         session.OrgID,
            UserID:        session.UserID,
            Content:       input.Content,
            Type:          "text",
            ContainerTags: []string{containerTag},
        })
        if err != nil { return "", err }
        return "Memory saved successfully.", nil

    case "forget":
        err := t.forgetClient.Forget(ctx, ForgetRequest{
            OrgID:   session.OrgID,
            SpaceID: containerTag,
            Content: input.Content,
        })
        if err != nil { return "", err }
        return "Memory forgotten.", nil

    default:
        return "", ErrInvalidAction
    }
}
```

#### `recall` Tool (Mới — Parallel Execution)

```go
// gateway/adapter/mcp/tools/recall_tool.go

type RecallToolInput struct {
    Query          string  `json:"query"`
    ContainerTag   *string `json:"containerTag"`
    IncludeProfile *bool   `json:"includeProfile"` // Default true
}

func (t *RecallTool) Handle(ctx context.Context, session *MCPSession, input RecallToolInput) (*RecallResponse, error) {
    containerTag := resolveContainerTag(session, input.ContainerTag)
    includeProfile := input.IncludeProfile == nil || *input.IncludeProfile

    var (
        searchResult *SearchResponse
        profile      *ProfileResponse
        searchErr    error
    )

    var wg sync.WaitGroup

    // G1: Search memories
    wg.Add(1)
    go func() {
        defer wg.Done()
        searchResult, searchErr = t.searchClient.MemorySearch(ctx, SearchRequest{
            Query:   input.Query,
            OrgID:   session.OrgID,
            SpaceID: containerTag,
            Limit:   10,
            Mode:    "memories-only",
        })
    }()

    // G2: Get profile (nếu yêu cầu)
    if includeProfile {
        wg.Add(1)
        go func() {
            defer wg.Done()
            profile, _ = t.profileClient.GetProfile(ctx, ProfileRequest{
                OrgID:   session.OrgID,
                SpaceID: containerTag,
            })
        }()
    }

    wg.Wait()

    if searchErr != nil { return nil, searchErr }

    // Merge + dedup: profile facts có priority cao hơn search results
    return mergeRecallResponse(profile, searchResult), nil
}
```

#### `listProjects` Tool (Mới)

```go
// gateway/adapter/mcp/tools/list_projects_tool.go

type ListProjectsToolInput struct{}

func (t *ListProjectsTool) Handle(ctx context.Context, session *MCPSession, _ ListProjectsToolInput) (*ListProjectsResponse, error) {
    // Check session cache (TTL 5 phút)
    if !isExpired(session.ContainerTagsCachedAt, 5*time.Minute) {
        return &ListProjectsResponse{Projects: session.CachedContainerTags}, nil
    }

    // Fetch từ Project Service
    projects, err := t.projectClient.ListProjects(ctx, ListProjectsRequest{OrgID: session.OrgID})
    if err != nil { return nil, err }

    // Update session cache
    tags := extractContainerTags(projects)
    session.CachedContainerTags = tags
    now := time.Now()
    session.ContainerTagsCachedAt = &now
    t.sessionMgr.save(ctx, session)

    return &ListProjectsResponse{Projects: tags}, nil
}
```

#### `whoAmI` Tool (Mới)

```go
// gateway/adapter/mcp/tools/whoami_tool.go

type WhoAmIToolInput struct{}

type WhoAmIResponse struct {
    UserID     string          `json:"userId"`
    OrgID      string          `json:"orgId"`
    Email      *string         `json:"email"`
    Name       *string         `json:"name"`
    ClientInfo *MCPClientInfo  `json:"clientInfo"`
    SessionID  string          `json:"sessionId"`
}

func (t *WhoAmITool) Handle(ctx context.Context, session *MCPSession, _ WhoAmIToolInput) (*WhoAmIResponse, error) {
    return &WhoAmIResponse{
        UserID:     session.UserID,
        OrgID:      session.OrgID,
        Email:      session.Email,
        Name:       session.Name,
        ClientInfo: session.ClientInfo,
        SessionID:  session.SessionID,
    }, nil
}
```

### 3.4. MCP Resources

```go
// gateway/adapter/mcp/resources/profile_resource.go

// URI: supermemory://profile
func (r *ProfileResource) Handle(ctx context.Context, session *MCPSession) (*ResourceContent, error) {
    containerTag := resolveDefaultContainerTag(session)
    profile, err := r.profileClient.GetProfile(ctx, ProfileRequest{
        OrgID:   session.OrgID,
        SpaceID: containerTag,
    })
    if err != nil { return nil, err }

    content := fmt.Sprintf("Static facts:\n%s\n\nDynamic context:\n%s",
        strings.Join(profile.Static, "\n"),
        strings.Join(profile.Dynamic, "\n"),
    )

    return &ResourceContent{
        URI:      "supermemory://profile",
        MimeType: "text/plain",
        Text:     content,
    }, nil
}

// URI: supermemory://projects
func (r *ProjectsResource) Handle(ctx context.Context, session *MCPSession) (*ResourceContent, error) {
    projects, _ := r.projectClient.ListProjects(ctx, ListProjectsRequest{OrgID: session.OrgID})
    content := formatProjectsList(projects)

    return &ResourceContent{
        URI:      "supermemory://projects",
        MimeType: "text/plain",
        Text:     content,
    }, nil
}
```

### 3.5. Context Prompt

```go
// gateway/adapter/mcp/prompts/context_prompt.go

func (p *ContextPrompt) Handle(ctx context.Context, session *MCPSession) (*PromptContent, error) {
    containerTag := resolveDefaultContainerTag(session)
    profile, _ := p.profileClient.GetProfile(ctx, ProfileRequest{
        OrgID:   session.OrgID,
        SpaceID: containerTag,
    })

    systemMessage := buildContextSystemMessage(profile)

    return &PromptContent{
        Name:        "context",
        Description: "Inject user profile context into the conversation",
        Messages: []PromptMessage{
            {
                Role:    "system",
                Content: systemMessage,
            },
        },
    }, nil
}

func buildContextSystemMessage(profile *ProfileResponse) string {
    return fmt.Sprintf(`<user_profile>
Long-term facts:
%s

Current activities:
%s
</user_profile>

When you learn new information about the user, save it using the memory tool with action=save.
When asked to forget something, use memory with action=forget.
Use the recall tool to search for relevant past context before answering questions.`,
        strings.Join(profile.Static, "\n"),
        strings.Join(profile.Dynamic, "\n"),
    )
}
```

### 3.6. OAuth2 for MCP

```go
// gateway/adapter/mcp/auth.go

// Tích hợp với Auth Service OAuth2 server (CR-SM-007)
// MCP clients authenticate qua standard OAuth2 Authorization Code flow

// Discovery endpoint (well-known)
// GET /.well-known/oauth-authorization-server → auth service metadata

// MCP server xác thực token từ Authorization header:
// Authorization: Bearer <access_token_from_oauth>

func (s *MCPServer) authenticateRequest(r *http.Request) (*MCPSession, error) {
    // 1. Try Bearer token (OAuth)
    if bearer := extractBearer(r); bearer != "" {
        userID, orgID, err := s.authClient.ValidateBearerToken(r.Context(), bearer)
        if err != nil { return nil, ErrUnauthorized }
        session, _ := s.sessionMgr.GetOrCreate(r.Context(), userID)
        session.OrgID = orgID
        return session, nil
    }

    // 2. Try API Key (sm_ prefix)
    if apiKey := extractAPIKey(r); apiKey != "" {
        userID, orgID, err := s.authClient.ValidateAPIKey(r.Context(), apiKey)
        if err != nil { return nil, ErrUnauthorized }
        session, _ := s.sessionMgr.GetOrCreate(r.Context(), userID)
        session.OrgID = orgID
        session.APIKey = apiKey
        return session, nil
    }

    return nil, ErrUnauthorized
}

// Capture client info từ MCP Initialize params
func (s *MCPServer) handleInitialize(ctx context.Context, session *MCPSession, params InitializeParams) {
    if params.ClientInfo.Name != "" {
        s.sessionMgr.SetClientInfo(ctx, session, MCPClientInfo{
            Name:    params.ClientInfo.Name,
            Version: params.ClientInfo.Version,
        })
    }
}
```

### 3.7. Install Command

```bash
# One-liner install cho mỗi client
# Tạo package @vnp-memory/install-mcp hoặc dùng npx install-mcp

npx -y install-mcp@latest vnp-memory --client claude
# → Thêm vào ~/Library/Application Support/Claude/claude_desktop_config.json:
# {
#   "mcpServers": {
#     "vnp-memory": {
#       "url": "https://api.vnpmemory.io/mcp",
#       "env": { "VNP_MEMORY_API_KEY": "..." }
#     }
#   }
# }

npx -y install-mcp@latest vnp-memory --client cursor
# → Thêm vào ~/.cursor/mcp.json
```

---

## 4. Registered MCP Capabilities Summary

| Capability | Type | Implementation |
|------------|------|----------------|
| `memory` | Tool | save/forget, max 200K chars |
| `recall` | Tool | Parallel search + profile |
| `listProjects` | Tool | Project list, cache 5 phút |
| `whoAmI` | Tool | User + client info từ session |
| `supermemory://profile` | Resource | Static + Dynamic profile text |
| `supermemory://projects` | Resource | Projects list |
| `context` | Prompt | System message với user context |

---

## 5. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Session model + Redis persistence | 1 ngày |
| **P2** | Nâng cấp `memory` tool (max 200K, action enum) | 1 ngày |
| **P3** | `recall` tool (parallel search + profile) | 2 ngày |
| **P4** | `listProjects` + `whoAmI` tools | 1 ngày |
| **P5** | MCP Resources (profile + projects) | 1 ngày |
| **P6** | Context prompt | 1 ngày |
| **P7** | OAuth2 for MCP integration | 2 ngày |
| **P8** | Client info capture (Initialize handler) | 1 ngày |
| **P9** | Tests + Compatibility (Claude, Cursor, VSCode) | 2 ngày |

**Tổng:** ~12 ngày (Wave 4)

---

## 6. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| Claude Desktop thấy 4 tools | Registered: memory, recall, listProjects, whoAmI |
| `memory` save → data trong Document Service | memory_tool.go → docClient.CreateDocument |
| `recall` → profile + search trong 1 resp | Parallel G1+G2 → mergeRecallResponse |
| `/context` prompt → system message đầy đủ | buildContextSystemMessage(profile) |
| Disconnect/reconnect → session restored | Redis keyed by user_id, TTL 24h |
| `whoAmI` trả về đúng client name | ClientInfo captured từ MCP Initialize |
