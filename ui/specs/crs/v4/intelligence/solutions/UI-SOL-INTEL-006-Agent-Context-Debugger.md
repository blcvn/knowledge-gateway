# UI Solution: UI-SOL-INTEL-006 — Agent Context Debugger UI

**Solution ID:** UI-SOL-INTEL-006  
**CR References:** [CR-INTEL-006](../../../../docs/crs/v4/intelligence/CR-INTEL-006-Agent-Context-Debugger.md)  
**Backend Solution:** [SOL-INTEL-006](../../../../backend/specs/crs/v4/intelligence/solutions/SOL-INTEL-006-Agent-Context-Debugger.md)  
**Feature:** Agent Context Debugger — 3-Column Layout, Trace Breakdown, Token Chart  
**Priority:** 🟠 Medium  
**Frontend Component:** `ui/src/pages/observability/ContextDebugger.tsx`

---

## 1. Mục Đích

Xây dựng Agent Context Debugger UI:
- 3-column layout: Request | Pipeline | Final Prompt
- Per-engine context breakdown với tokens và latency
- Token budget donut chart
- LLM prompt preview (assembled context)
- Trace list với `X-Debug-Trace: true` results

---

## 2. Backend API Contract

```http
# Enable trace on recall
POST /v1/memory/recall
X-Debug-Trace: true
→ response includes "trace_id"

# Get trace detail
GET /v1/traces/{trace_id}
→ {
    "trace_id": string,
    "request": { "query": string, "user_id": string },
    "context_breakdown": [
      { "engine": string, "tier"?: string, "files"?: string[], "facts"?: string[],
        "profile"?: string, "tokens": number, "retrieval_ms": number }
    ],
    "total_tokens": number,
    "total_retrieval_ms": number,
    "llm_prompt_preview": string
  }

# List recent traces
GET /v1/traces?user_id=u_123&limit=20
→ TraceSpan[]
```

---

## 3. Components Architecture

### 3.1 3-Column Debugger Layout

```
ContextDebuggerPage (3 columns, equal width)
│
Column 1: Request           Column 2: Pipeline          Column 3: Final Prompt
─────────────────────       ─────────────────────       ─────────────────────
TracesList                  EngineBreakdown             AssembledPrompt
  └── TraceItem               ├── EngineSection (each)   ├── SystemPromptSection
       ├── query preview         │   ├── EngineName         │   (context injected)
       ├── user_id               │   ├── TierBadge          ├── UserQuerySection
       ├── timestamp             │   ├── ContentList        └── TokenCounter
       └── [SELECT]              │   ├── TokenBar
                                 │   └── LatencyMs
                                 ├── TotalRow
                                 │   ├── TotalTokens
                                 │   └── TotalLatency
                                 └── TokenDonutChart
```

### 3.2 Engine Section Component

```typescript
function EngineSection({ breakdown }: { breakdown: ContextBreakdown }) {
  return (
    <div className="border rounded-lg p-3 space-y-2">
      <div className="flex items-center gap-2">
        <span className={ENGINE_COLORS[breakdown.engine]}>{breakdown.engine}</span>
        {breakdown.tier && <TierBadge tier={breakdown.tier as 'L0' | 'L1' | 'L2'} />}
        <span className="ml-auto text-xs text-gray-500">{breakdown.retrieval_ms}ms</span>
      </div>
      
      {/* Files (OpenViking) */}
      {breakdown.files?.map(f => (
        <div key={f} className="text-xs font-mono text-gray-600 truncate">{f}</div>
      ))}
      
      {/* Facts (Graphiti) */}
      {breakdown.facts?.map(fact => (
        <div key={fact} className="text-xs text-gray-700 italic">"{fact}"</div>
      ))}
      
      {/* Profile (Memobase) */}
      {breakdown.profile && (
        <div className="text-xs text-gray-700">{breakdown.profile}</div>
      )}
      
      {/* Token bar */}
      <div className="text-xs font-semibold">{breakdown.tokens} tokens</div>
    </div>
  );
}
```

### 3.3 Token Donut Chart

```typescript
// Using Recharts PieChart
function TokenDonutChart({ breakdown }: { breakdown: ContextBreakdown[] }) {
  const data = breakdown.map(b => ({
    name:  b.engine,
    value: b.tokens,
    fill:  ENGINE_HEX_COLORS[b.engine],
  }));
  
  const total = breakdown.reduce((sum, b) => sum + b.tokens, 0);
  
  return (
    <PieChart width={180} height={180}>
      <Pie data={data} innerRadius={50} outerRadius={80} dataKey="value">
        {data.map((entry, i) => <Cell key={i} fill={entry.fill} />)}
      </Pie>
      <text x={90} y={90} textAnchor="middle" dominantBaseline="middle">
        {total}t
      </text>
      <Tooltip formatter={(v) => `${v} tokens`} />
    </PieChart>
  );
}
```

### 3.4 LLM Prompt Preview

```typescript
// Assembled prompt in mono font
// Sections clearly demarcated:
// [SYSTEM CONTEXT]
// <OpenViking L1: auth.go>
// ...file summary...
// <Graphiti: episodic facts>
// - Auth uses JWT RS256 (valid since 2026-01-01)
// <Memobase: user profile>
// Tên: Bình | Role: Backend Eng...
//
// [USER QUERY]
// auth middleware JWT

function LLMPromptPreview({ promptPreview }: { promptPreview: string }) {
  return (
    <pre className="text-xs font-mono bg-gray-900 text-gray-100 p-3 rounded 
                    overflow-auto max-h-96 whitespace-pre-wrap">
      {promptPreview}
    </pre>
  );
}
```

---

## 4. Enable Trace in Dev Panel

```typescript
// Dev panel: enable trace for next recall
const [traceEnabled, setTraceEnabled] = useState(false);

// When traceEnabled: add header X-Debug-Trace: true to next recall request
// After recall: show trace_id → fetch trace detail
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] 3-column layout (Request | Pipeline | Final Prompt)
- [ ] Trace list on left, select to load detail
- [ ] Engine sections: OpenViking shows files, Graphiti shows facts, Memobase shows profile
- [ ] Tier badges (L0/L1/L2) on OpenViking section
- [ ] Token donut chart: correct proportions per engine
- [ ] Total tokens + total retrieval_ms displayed
- [ ] LLM prompt preview: mono font, scrollable, max-height
- [ ] `X-Debug-Trace: true` toggle in dev panel
- [ ] Traces auto-expire: "7 days" note in UI (no manual delete needed)
