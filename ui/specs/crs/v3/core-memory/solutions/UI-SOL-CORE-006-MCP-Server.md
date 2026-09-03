# UI Solution: UI-SOL-CORE-006 — MCP Server Management UI

**Solution ID:** UI-SOL-CORE-006  
**CR References:** [CR-CORE-006](../../../../docs/crs/v3/core-memory/CR-CORE-006-MCP-Server.md)  
**Backend Solution:** [SOL-CORE-006](../../../../backend/specs/crs/v3/core-memory/solutions/SOL-CORE-006-MCP-Server.md)  
**Feature:** MCP Server — 37+ Tools, Dual Transport, Integration Guide  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/api-sdk/MCPIntegration.tsx`

---

## 1. Mục Đích

Xây dựng MCP Server Management UI:
- Tool browser với 37+ tools và JSON Schema
- Dual transport config (SSE vs HTTP Streamable)
- Connection tester
- Integration quickstart guides (Claude Code, LangChain, AutoGen)
- Token budget setting per scope

---

## 2. Backend API Contract

```http
GET /mcp/tools
→ {
    "tools": [
      {
        "name": "memory_store",
        "description": "Store memory to appropriate engine",
        "category": "memory",
        "inputSchema": { "type": "object", "properties": { ... } },
        "transport": ["sse", "http"]
      },
      // ... 36 more tools
    ],
    "total": 37
  }

GET  /mcp/sse       → SSE transport endpoint
POST /mcp/message   → HTTP Streamable transport endpoint
```

---

## 3. Components Architecture

### 3.1 MCP Integration Dashboard

```
MCPDashboardPage
├── ConnectionStatusCard    ← MCP server: online/offline
├── TransportTabs
│   ├── SSE Transport Tab
│   │   ├── EndpointUrl     ← GET /mcp/sse
│   │   ├── UsedBy          ← "Claude Code, legacy clients"
│   │   └── CopyEndpointBtn
│   └── HTTP Streamable Tab
│       ├── EndpointUrl     ← POST /mcp/message
│       ├── UsedBy          ← "New MCP clients, LangChain"
│       └── CopyEndpointBtn
├── QuickstartTabs
│   ├── Claude Code Tab     ← mcp.json config snippet
│   ├── LangChain Tab       ← Python code snippet
│   └── AutoGen Tab         ← config snippet
└── ToolsBrowser (below)
```

### 3.2 Integration Quickstart Configs

```typescript
// Claude Code: .claude/mcp.json
const claudeCodeConfig = {
  mcpServers: {
    "vnp-memory": {
      command: "sse",
      url: `${apiBaseUrl}/mcp/sse`,
      headers: {
        "Authorization": `Bearer ${apiKey}`,
        "X-Tenant-ID": tenantId
      }
    }
  }
};

// LangChain Python
const langchainSnippet = `
from langchain_mcp import MCPToolkit
toolkit = MCPToolkit(
    url="${apiBaseUrl}/mcp/message",
    headers={"Authorization": "Bearer ${apiKey}"}
)
tools = toolkit.get_tools()
`;

// Display as syntax-highlighted code blocks with copy button
```

### 3.3 Tools Browser

```
ToolsBrowser
├── SearchInput             ← search tool names
├── CategoryFilter          ← Memory | Filesystem | Profile | Agent | Graph | Admin
├── TransportFilter         ← SSE | HTTP | Both
└── ToolsGrid
    └── ToolCard
        ├── ToolName        ← "memory_recall"
        ├── CategoryBadge   ← color-coded
        ├── Description
        ├── TransportBadges ← SSE | HTTP
        └── SchemaSection (expandable)
            ├── InputSchema ← JSON Schema rendered as table
            └── ExampleCall ← example invocation JSON
```

### 3.4 Tool Category Colors

```typescript
const CATEGORY_COLORS = {
  memory:     'bg-blue-100 text-blue-800',
  filesystem: 'bg-green-100 text-green-800',
  profile:    'bg-orange-100 text-orange-800',
  agent:      'bg-purple-100 text-purple-800',
  graph:      'bg-indigo-100 text-indigo-800',
  admin:      'bg-red-100 text-red-800',
};

// Tool count per category (from backend response):
// Memory (6): memory_store, memory_recall, memory_forget, memory_timeline, memory_observe, memory_consolidate
// Filesystem (8): ov_read_file, ov_write_file, ov_grep, ov_search, ov_list_dir, ...
// Profile (5): profile_get_context, profile_get_user, profile_flush, ...
// Agent (7): observe_session_start, observe_session_end, observe_hook, ...
// Graph (5): graph_search, graph_add_episode, graph_get_entity, ...
// Admin (6): admin_forget_user, admin_list_tenants, admin_get_health, ...
```

### 3.5 Token Budget Settings

```
TokenBudgetPanel
├── ScopeSelector           ← project | session | global
└── BudgetSlider            ← 512 → 8192 tokens
    ├── L0 Suggestion       ← "< 200 tokens: headlines only"
    ├── L1 Suggestion       ← "< 2000 tokens: summaries"
    └── L2 Suggestion       ← "> 2000 tokens: full content"
```

---

## 4. React Query Hook

```typescript
export function useMCPTools() {
  return useQuery({
    queryKey: ['mcp', 'tools'],
    queryFn:  async () => {
      const res = await fetch('/mcp/tools', {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      });
      return res.json() as Promise<{ tools: MCPTool[]; total: number }>;
    },
    staleTime: 5 * 60_000,   // stable list
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] All 37+ tools displayed with category filter
- [ ] Tool JSON Schema displayed as readable table (not raw JSON)
- [ ] Claude Code config generated with current API key + tenant
- [ ] "Copy" button for each config snippet
- [ ] Connection tester: ping `/mcp/tools` → show `200 OK` + latency
- [ ] Transport toggle: SSE vs HTTP Streamable endpoints clear
- [ ] Token budget slider: 3 tier suggestions visible
- [ ] Tool search: filter by name in real-time
