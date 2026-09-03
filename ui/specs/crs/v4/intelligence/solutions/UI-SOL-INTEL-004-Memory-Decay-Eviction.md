# UI Solution: UI-SOL-INTEL-004 — Memory Decay & Eviction UI

**Solution ID:** UI-SOL-INTEL-004  
**CR References:** [CR-INTEL-004](../../../../docs/crs/v4/intelligence/CR-INTEL-004-Memory-Decay-Eviction.md)  
**Backend Solution:** [SOL-INTEL-004](../../../../backend/specs/crs/v4/intelligence/solutions/SOL-INTEL-004-Memory-Decay-Eviction.md)  
**Feature:** Memory Decay — Salience Score Display, Decay Timeline, Eviction Preview  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/memory-explorer/`

---

## 1. Mục Đích

Xây dựng Memory Decay & Eviction UI:
- Salience score visualization per memory
- Decay timeline: salience trend over time
- Eviction preview: "these memories will be deleted in next eviction cycle"
- Eviction policy configuration

---

## 2. Backend API Contract

```http
# Memory with decay info
GET /v1/console/memory/{id}
→ MemoryItem with salience_score, decay_rate, forget_after

# Eviction preview
GET /v1/console/memory/eviction-preview?user_id=u_123
→ {
    "candidates": [
      { "id": string, "content": string, "salience": number, "expires_in_days": number }
    ],
    "total_candidates": number
  }

# Adaptive analytics (decay metrics)
GET /v1/console/adaptive/analytics
→ { creation_rate, deletion_rate, contradiction_count, storage_usage_bytes }
```

---

## 3. Components

### 3.1 Decay Metrics Dashboard

```
DecayDashboard
├── SalienceDistribution    ← histogram: high/medium/low salience counts
├── DecayRateChart          ← line chart: memories decaying per day
├── EvictionForecast        ← "~47 memories will be evicted this week"
└── StorageUsageBar         ← "2.8 GB / 10 GB (28%)"
```

### 3.2 Memory Card with Salience

```typescript
// Salience display on MemoryCard
function SalienceBadge({ score }: { score: number }) {
  const pct = Math.round(score * 100);
  
  if (pct >= 70) return <span className="text-green-600">⬆ {pct}% salience</span>;
  if (pct >= 40) return <span className="text-amber-600">➡ {pct}% salience</span>;
  return <span className="text-red-500">⬇ {pct}% — DECAY RISK</span>;
}

// Decay warning badge
function DecayWarning({ forgetAfter }: { forgetAfter: string | null }) {
  if (!forgetAfter) return null;
  const daysLeft = parseDaysLeft(forgetAfter);
  if (daysLeft < 7) {
    return <span className="badge-red">⏰ Eviction in {daysLeft}d</span>;
  }
  return <span className="badge-amber">TTL: {forgetAfter}</span>;
}
```

### 3.3 Eviction Preview Panel

```
EvictionPreviewPanel
├── Header                  ← "47 memories scheduled for eviction"
├── EvictionList
│   └── EvictionCandidate
│       ├── ContentPreview  ← first 60 chars
│       ├── SalienceBar     ← low salience bar (red)
│       ├── ExpiresIn       ← "Evicts in 3 days"
│       └── SaveButton      ← "Keep" (increase salience / remove TTL)
└── RunEvictionBtn          ← "Run Eviction Now" (admin only)
```

### 3.4 Eviction Policy Config

```
EvictionPolicyPage
├── ForgetRulesTable        ← (from forget-rules API)
│   └── RuleRow
│       ├── MemoryType
│       ├── ForgetAfter     ← "30d" | "90d" | "1y"
│       ├── ContradictionResolution ← keep_latest | keep_both | manual
│       └── NoiseFilter     ← toggle
├── AddRuleButton
└── SaveButton              ← PUT /v1/console/adaptive/forget-rules
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] Salience score displayed on memory cards (color-coded)
- [ ] Salience < 0.2 → `DECAY RISK` badge (red)
- [ ] Eviction preview: list memories expiring soon
- [ ] "Keep" button on eviction candidates (removes TTL)
- [ ] Storage usage bar with GB display
- [ ] Eviction policy config: edit forget_after per memory_type
- [ ] Decay metrics chart: creation_rate vs deletion_rate trend
