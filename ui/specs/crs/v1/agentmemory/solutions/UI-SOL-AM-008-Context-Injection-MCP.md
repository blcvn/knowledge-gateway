# UI Solution: UI-SOL-AM-008 — Context Injection & MCP Management

**Solution ID:** UI-SOL-AM-008  
**CR References:** [CR-AM-008](../../../../docs/crs/v1/agentmemory/CR-AM-008-Context-Injection-MCP.md)  
**Backend Solution:** [SOL-008-Context-Injection-MCP.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-008-Context-Injection-MCP.md)  
**Feature:** Context Injection — Agent Integration & MCP Tool Browser  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/api-sdk/` + `ui/src/pages/observability/`

---

## 1. Mục Đích

Xây dựng UI cho MCP Server management và Context Injection:
- MCP Tools Browser: xem tất cả 37+ tools với schemas
- Connection Tester: test MCP connection (SSE/HTTP Streamable)
- Context Trace Viewer: xem context được inject vào mỗi LLM call
- Token Budget visualization

---

## 2. Backend API Alignment

### API Endpoints

| Method | Path | Mô tả |
|--------|------|--------|
| `GET` | `/mcp/tools` | List all 37+ MCP tools với schemas |
| `GET` | `/v1/traces/{trace_id}` | Context breakdown trace |
| `GET` | `/v1/traces` | List recent traces |

### TypeScript Types

```typescript
// ui/src/types/mcp.ts

interface MCPTool {
  name:        string;         // "memory_store", "ov_grep", ...
  description: string;
  category:    'memory' | 'filesystem' | 'profile' | 'agent' | 'graph' | 'admin';
  input_schema: JSONSchema;    // JSON Schema for tool args
  transport:   ('sse' | 'http')[];
}

interface ContextTrace {
  trace_id:          string;
  request: {
    query:   string;
    user_id: string;
  };
  context_breakdown: Array<{
    engine:         string;
    tier?:          'L0' | 'L1' | 'L2';
    files?:         string[];
    facts?:         string[];
    profile?:       string;
    tokens:         number;
    retrieval_ms:   number;
  }>;
  total_tokens:      number;
  total_retrieval_ms: number;
  llm_prompt_preview: string;
  created_at:        string;
}
```

---

## 3. Components Architecture

### 3.1 MCP Tools Browser

```
MCPToolsBrowserPage
├── ToolSearch              ← search by name/description
├── CategoryTabs            ← Memory | Filesystem | Profile | Agent | Graph | Admin
├── ToolsGrid               ← card grid
│   └── ToolCard
│       ├── ToolName        ← "memory_recall"
│       ├── CategoryBadge
│       ├── Description     ← first 80 chars
│       ├── TransportBadges ← SSE | HTTP tags
│       └── SchemaButton    ← expand to show JSON Schema
└── ConnectionPanel (right sidebar)
    ├── TransportSelector   ← SSE | HTTP Streamable
    ├── EndpointDisplay     ← /mcp/sse or /mcp/message
    ├── CopyConfigButton    ← copy Claude Code MCP config
    └── TestConnectionBtn   ← ping and show response
```

### 3.2 MCP Config Copy (Claude Code format)

```typescript
// Generated config for Claude Code .claude/mcp.json
const mcpConfig = {
  mcpServers: {
    "vnp-memory": {
      command: "sse",
      url: `${apiBaseUrl}/mcp/sse`,
      headers: { "Authorization": `Bearer ${apiKey}` }
    }
  }
};
// Button: "Copy for Claude Code" → copies JSON to clipboard
```

### 3.3 Context Trace Viewer

```
ContextTracePage
├── TracesList (left)       ← recent traces, click to select
│   └── TraceItem           ← trace_id, query preview, timestamp
└── TraceDetail (right)     ← selected trace breakdown
    ├── QueryHeader         ← original query + user_id
    ├── EngineBreakdown     ← per-engine contribution
    │   └── EngineSection
    │       ├── EngineName + TierBadge (L0/L1/L2)
    │       ├── ContentList ← files or facts
    │       └── TokenBar    ← tokens used by this engine
    ├── TokenBudgetChart    ← donut chart: budget used per engine
    ├── TotalStats          ← total_tokens, total_retrieval_ms
    └── PromptPreview       ← collapsible assembled LLM prompt
```

### 3.4 Token Budget Donut Chart

```
          Total: 370 tokens
         ┌────────────┐
    OpenViking   openviking: 340 (91.9%)
     91.9%  │╲   graphiti:   18  (4.9%)
            │ ╲  memobase:   12  (3.2%)
    graphiti│  ╲
     4.9%   └───
    memobase 3.2%
```

---

## 4. React Query Hooks

```typescript
// ui/src/api/hooks/useMCP.ts

export function useMCPTools() {
  return useQuery({
    queryKey: ['mcp', 'tools'],
    queryFn:  () => fetch('/mcp/tools').then(r => r.json()),
    staleTime: 5 * 60_000,    // tools rarely change
  });
}

export function useContextTraces(userId?: string) {
  return useQuery({
    queryKey: ['traces', { userId }],
    queryFn:  () => observabilityApi.listTraces({ user_id: userId, limit: 20 }),
    refetchInterval: 10_000,
  });
}

export function useContextTrace(traceId: string) {
  return useQuery({
    queryKey: ['traces', traceId],
    queryFn:  () => observabilityApi.getTrace(traceId),
    enabled: !!traceId,
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] MCP Tools Browser hiển thị tất cả 37+ tools với category filter
- [ ] Tool schema expandable (JSON Schema rendered nicely)
- [ ] "Copy for Claude Code" generates valid MCP config JSON
- [ ] Connection tester: ping `/mcp/tools` và hiển thị response time
- [ ] Context Trace Viewer: breakdown per-engine với token counts
- [ ] Token budget donut chart đúng tỷ lệ
- [ ] LLM prompt preview collapsible và syntax-highlighted
- [ ] Traces list auto-refresh mỗi 10s
