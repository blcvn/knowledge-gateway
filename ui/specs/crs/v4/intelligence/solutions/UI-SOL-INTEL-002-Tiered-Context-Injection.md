# UI Solution: UI-SOL-INTEL-002 — Tiered Context Injection UI

**Solution ID:** UI-SOL-INTEL-002  
**CR References:** [CR-INTEL-002](../../../../docs/crs/v4/intelligence/CR-INTEL-002-Tiered-Context-Injection.md)  
**Backend Solution:** [SOL-INTEL-002](../../../../backend/specs/crs/v4/intelligence/solutions/SOL-INTEL-002-Tiered-Context-Injection.md)  
**Feature:** Tiered Context — L0/L1/L2 Tier Badges, Token Budget Display  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/memory-explorer/ContextInspector.tsx`

---

## 1. Mục Đích

Xây dựng UI cho Tiered Context Injection:
- Context query panel với tier selector và token budget
- L0/L1/L2 tier badges per result
- Token budget visualization (used/remaining)
- Semantic grep results
- Context assembly preview

---

## 2. Backend API Contract

```http
POST /v1/ov/context
{
  "query":        string,
  "scope":        "project" | "session" | "global",
  "tier":         "auto" | "L0" | "L1" | "L2",
  "token_budget": number    // e.g., 4096
}
→ {
    "results": [
      { "path": string, "tier": "L0"|"L1"|"L2", "content": string, "tokens": number }
    ],
    "total_tokens": number,
    "budget_used":  string    // "9.4%"
  }

POST /v1/ov/grep
{ "pattern": string, "scope": "project" }
→ { "matches": [{ "path": string, "line": number, "content": string }] }
```

---

## 3. Components Architecture

### 3.1 Context Inspector Panel

```
ContextInspectorPage
├── QueryInput              ← "auth middleware JWT"
├── ControlRow
│   ├── ScopeSelect         ← project | session | global
│   ├── TierSelect          ← Auto | L0 | L1 | L2
│   └── BudgetInput         ← token budget (default 4096)
├── TokenBudgetBar          ← before search: "Budget: 4096 tokens"
│                             after search:  "Used: 385 / 4096 (9.4%)"
└── ResultsList
    └── ContextResultCard
        ├── FilePath        ← "gateway/auth.go"
        ├── TierBadge       ← [L0] / [L1] / [L2] (colored)
        ├── TokenCount      ← "340 tokens"
        ├── ContentPreview  ← collapsible content
        └── UpgradeButton   ← "Load L2 (full file)" → re-query with L2
```

### 3.2 Tier Badge Styling

```typescript
const TIER_STYLES = {
  L0: 'bg-green-100 text-green-800 border-green-300',   // headlines (cheapest)
  L1: 'bg-blue-100 text-blue-800 border-blue-300',      // summaries (medium)
  L2: 'bg-purple-100 text-purple-800 border-purple-300', // full (expensive)
};

const TIER_LABELS = {
  L0: 'L0 Headlines',
  L1: 'L1 Summary',
  L2: 'L2 Full',
};

function TierBadge({ tier }: { tier: 'L0' | 'L1' | 'L2' }) {
  return (
    <span className={`text-xs px-2 py-0.5 rounded border font-mono ${TIER_STYLES[tier]}`}>
      {tier}
    </span>
  );
}
```

### 3.3 Token Budget Visualization

```typescript
// Before query: show budget setting
// After query: show actual usage

function TokenBudgetBar({ budget, used }: { budget: number; used?: number }) {
  const pct = used ? Math.round((used / budget) * 100) : 0;
  
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-sm">
        <span>Token Budget</span>
        <span>{used ? `${used} / ${budget} (${pct}%)` : `${budget} available`}</span>
      </div>
      {used && (
        <div className="h-2 bg-gray-100 rounded overflow-hidden">
          <div
            className={`h-full rounded ${pct > 90 ? 'bg-red-500' : pct > 70 ? 'bg-amber-500' : 'bg-green-500'}`}
            style={{ width: `${pct}%` }}
          />
        </div>
      )}
    </div>
  );
}
```

### 3.4 Semantic Grep Panel

```
SemanticGrepPanel
├── PatternInput            ← "JWT" or "auth" or regex
├── ScopeSelect             ← project | session | global
└── GrepResults
    └── GrepMatch
        ├── FilePath        ← "gateway/auth.go"
        ├── LineNumber      ← ":42"
        └── LineContent     ← "...JWT middleware, exports..."
              (with pattern highlighted in bold)
```

---

## 4. React Query Hooks

```typescript
// ui/src/api/hooks/useContextInjection.ts

export function useContextQuery() {
  return useMutation({
    mutationFn: (req: ContextQueryRequest) => ovApi.queryContext(req),
  });
}

export function useSemanticGrep() {
  return useMutation({
    mutationFn: (req: { pattern: string; scope: string }) => ovApi.grep(req),
  });
}
```

---

## 5. Cost Savings Display

```
Cost Comparison (shown after query):
────────────────────────────────────
Without Tiering:  ~8,000 tokens  ($0.50/call)
With L0 Headlines:   385 tokens  ($0.02/call) ✅ 96% savings
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Tier selector: Auto / L0 / L1 / L2
- [ ] Token budget input: 512-8192, default 4096
- [ ] Results show L0/L1/L2 badge per file
- [ ] Token budget bar: shows % used after query
- [ ] "Load L2" button upgrades individual file to full content
- [ ] Semantic grep: pattern highlighted in results
- [ ] Cost savings comparison: shown after first query
- [ ] Budget exceeded warning: red bar + toast if `budget_used > 90%`
