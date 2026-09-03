# TASK-SM-010 — services/mcp-server: MCP Tools Upgrade (Supermemory Set)

**Task ID:** TASK-SM-010  
**Wave:** 4 (Integrations)  
**Solution:** [SOL-SM-006](../solutions/SOL-SM-006-MCP-Server.md)  
**Depends on:** TASK-SM-007 (search), TASK-SM-008 (profile), TASK-SM-005 (documents)  
**Ước tính:** 4h  
**Priority:** High — Claude Desktop / Cursor integration

---

## Mục tiêu

Nâng cấp MCP server trong `gateway/adapter/mcp/` với Supermemory toolset:
- Tất cả tools đều support auth via `sm_` API key
- OAuth2 integration (từ TASK-SM-003)
- Tools: memory_profile, search, add_memory, forget_memory, list_documents, get_document_status

---

## Công việc cụ thể

### 1. Audit & Map Existing MCP Tools

Kiểm tra `gateway/adapter/mcp/` hiện có gì, map với Supermemory tool set:

| MCP Tool | Handler | Status |
|----------|---------|--------|
| `memory_profile` | ProfileService.GetProfile | Exists (upgrade) |
| `search` | SearchService.HybridSearch | Exists (upgrade) |
| `add_memory` | MemoryService.CreateMemory | New |
| `forget_memory` | MemoryService.Forget | New |
| `list_documents` | DocumentService.ListDocuments | New |
| `get_document_status` | DocumentService.GetDocument | New |
| `search_memories_v4` | SearchService.MemorySearchV4 | New |
| `get_profile_with_search` | ProfileService.ProfileSearch | New |

### 2. Implement/Upgrade Tool Handlers

**`gateway/adapter/mcp/tools/`**

#### `memory_profile.go` (UPGRADE)
```go
// Tool: memory_profile
// Input: {spaceId?: string, includeSystemPrompt?: bool}
// Returns: UserProfile + optional formatted system prompt
// NEW: includeSystemPrompt → call ToSystemPrompt()
```

#### `search.go` (UPGRADE)
```go
// Tool: search
// Input: {query, spaceId, limit, mode, rewriteQuery, rerank, filters}
// NEW params: mode (hybrid|memories-only|documents-only), filters JSONB
// Returns: SearchResult[] with score + type (chunk|memory|document)
```

#### `add_memory.go` (NEW)
```go
// Tool: add_memory
// Input: {content, spaceId, isStatic?, forgetAfter?}
// Calls: MemoryService.CreateMemory directly (bypass document pipeline)
// Returns: {memoryId, createdAt}
```

#### `forget_memory.go` (NEW)
```go
// Tool: forget_memory
// Input: {content?, memoryId?} (at least one required)
// Calls: MemoryService.Forget 3-phase algorithm
// Returns: {forgotten: true, count: N}
// Requires: memory:forget permission
```

#### `list_documents.go` (NEW)
```go
// Tool: list_documents
// Input: {spaceId, status?, limit?, offset?}
// Returns: Document[] with status field (queued|extracting|...|done|failed)
```

#### `get_document_status.go` (NEW)
```go
// Tool: get_document_status
// Input: {documentId}
// Returns: {id, status, chunkCount, tokenCount, processingMeta}
// Used by AI to poll async ingestion progress
```

#### `search_memories_v4.go` (NEW)
```go
// Tool: search_memories_v4
// Input: {query, spaceId, limit}
// Returns: {memory, score}[] with explicit similarity scores
```

#### `get_profile_with_search.go` (NEW)
```go
// Tool: get_profile_with_search
// Input: {spaceId, query?, limit?}
// Calls: ProfileService.ProfileSearch (profile + search in 1 call)
// Returns: {profile: {static, dynamic}, searchResults: []}
```

### 3. OAuth2 MCP Authentication

**`gateway/adapter/mcp/auth.go`**

```go
// Support 2 auth modes:
// 1. API key: "Authorization: Bearer sm_xxx" → ValidateAPIKey
// 2. OAuth2: "Authorization: Bearer {jwt}" → ValidateJWT (from OAuth2 flow)
// Return AuthContext{OrgID, UserID, Role}
```

**`gateway/adapter/mcp/middleware.go`**:
- Extract auth from request headers
- Inject AuthContext into MCP call context
- Apply RBAC: forget_memory requires memory:forget permission

### 4. MCP Server Transport

```go
// Dual transport (same as ZEP-014 pattern):
// --stdio: Claude Desktop / Cline
// --port N: HTTP Streamable 2025-03-26 spec
// Start: cmd/mcp-server/main.go
```

### 5. Tests

- `TestMemoryProfileTool_IncludeSystemPrompt`: tool call with includeSystemPrompt=true → response has systemPrompt
- `TestSearchTool_HybridMode`: mode=hybrid → both chunks and memories in results
- `TestSearchTool_FilterParam`: filters JSONB → passed to search service
- `TestAddMemoryTool_Success`: content + spaceId → memory created
- `TestForgetMemoryTool_RequiresPermission`: viewer role → 403
- `TestGetDocumentStatusTool_AsyncPoll`: status=extracting → status field in response
- `TestOAuth2Auth_ValidJWT`: valid JWT → OrgID extracted
- `TestAPIKeyAuth_sm_Prefix`: sm_xxx → OrgID extracted

---

## Acceptance Criteria

- [ ] `go build ./gateway/...` không lỗi
- [ ] MCP server lists 8 tools (không ít hơn, không nhiều hơn)
- [ ] memory_profile tool → response has `static`, `dynamic`, `memoryCount`
- [ ] search tool mode=memories-only → no chunk results
- [ ] add_memory → memory exists in database
- [ ] forget_memory with viewer API key → 403 Forbidden
- [ ] get_document_status → real-time pipeline status
- [ ] `go test ./gateway/adapter/mcp/...` pass

---

## Files tạo/sửa

```
gateway/adapter/mcp/
├── auth.go                           (MODIFY: add OAuth2 JWT support)
├── middleware.go                     (NEW: RBAC enforcement)
├── server.go                         (MODIFY: register new tools)
└── tools/
    ├── memory_profile.go             (MODIFY: add includeSystemPrompt)
    ├── search.go                     (MODIFY: add mode + filters)
    ├── add_memory.go                 (NEW)
    ├── forget_memory.go              (NEW)
    ├── list_documents.go             (NEW)
    ├── get_document_status.go        (NEW)
    ├── search_memories_v4.go         (NEW)
    ├── get_profile_with_search.go    (NEW)
    └── tools_test.go                 (NEW)

cmd/mcp-server/main.go                (NEW or MODIFY)
```

## Sau khi hoàn thành

Chạy: `go build ./... && go test ./gateway/adapter/mcp/...`
