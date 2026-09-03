# UI Solution: UI-SOL-AM-002 — Memory Lifecycle Management

**Solution ID:** UI-SOL-AM-002  
**CR References:** [CR-AM-002](../../../../docs/crs/v1/agentmemory/CR-AM-002-Memory-Lifecycle.md)  
**Backend Solution:** [SOL-002-Memory-Lifecycle.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-002-Memory-Lifecycle.md)  
**Feature:** Memory Lifecycle — Create, Update, Decay, Eviction  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/memory-explorer/` + `ui/src/pages/adaptive/`

---

## 1. Mục Đích

Xây dựng UI cho Memory Lifecycle cho phép admin/developer:
- Xem danh sách memories với trạng thái lifecycle (active, decaying, evicted)
- Xem version chain của memory (is_latest, parent_id, relation_type)
- Quản lý forget rules và decay policy
- Monitor memory growth và eviction metrics

---

## 2. Backend API Alignment

### API Endpoints Sử Dụng

| Method | Path | Mô tả |
|--------|------|--------|
| `POST` | `/v1/console/memory/search` | Search memories với lifecycle filters |
| `GET` | `/v1/console/memory/{id}` | Memory detail với provenance |
| `GET` | `/v1/console/memory/{id}/versions` | Version chain |
| `GET` | `/v1/console/adaptive/memories` | Adaptive memories list |
| `GET` | `/v1/console/adaptive/memories/{id}/versions` | Adaptive version chain |
| `GET` | `/v1/console/adaptive/analytics` | Lifecycle analytics |
| `GET` | `/v1/console/adaptive/forget-rules` | Auto-forget rules |
| `PUT` | `/v1/console/adaptive/forget-rules` | Update forget rules |

### TypeScript Types

```typescript
// ui/src/types/memory.ts (extend existing)

interface MemoryLifecycleState {
  id:             string;
  is_latest:      boolean;
  parent_id?:     string;
  root_id?:       string;
  relation_type?: 'updates' | 'extends' | 'derives';
  forget_after?:  string;     // TTL: "30d", "90d"
  salience_score: number;     // 0.0 - 1.0
  decay_rate?:    number;     // decay per day
  created_at:     string;
  updated_at:     string;
}

interface LifecycleAnalytics {
  creation_rate:        number;  // memories/hour
  deletion_rate:        number;
  contradiction_count:  number;
  storage_usage_bytes:  number;
  avg_version_depth:    number;
  eviction_count_24h:   number;
}
```

---

## 3. Components Architecture

### 3.1 Memory Explorer — Lifecycle View

```
MemoryExplorerPage
├── SearchBar               ← query input + mode selector
├── LifecycleFilter         ← is_latest toggle, decay filter
├── MemoryResultGrid        ← card grid hoặc table view
│   ├── MemoryCard
│   │   ├── EngineTypeBadge ← graphiti/cognee/zep/...
│   │   ├── VersionBadge    ← "v3 (latest)" / "v1 (superseded)"
│   │   ├── SalienceBar     ← progress bar 0-100%
│   │   ├── TTLBadge        ← "expires in 23 days"
│   │   └── ActionButtons   ← View Detail / View Versions
│   └── Pagination
└── LifecycleAnalyticsPanel ← right sidebar
```

### 3.2 Version Chain Drawer

```
VersionChainDrawer (slide-in from right)
├── VersionTimeline         ← vertical chain oldest → newest
│   ├── VersionNode
│   │   ├── VersionNumber   ← "v1", "v2 (latest)"
│   │   ├── RelationBadge   ← "updates" | "extends" | "derives"
│   │   ├── ContentPreview  ← first 100 chars
│   │   └── DiffButton      ← show unified diff
│   └── ConnectorLine
└── DiffViewer              ← unified diff display
```

**React Query Hook:**
```typescript
export function useMemoryVersions(memoryId: string) {
  return useQuery({
    queryKey: ['memory', memoryId, 'versions'],
    queryFn: () => memoryApi.getVersions(encodeURIComponent(memoryId)),
    staleTime: 60_000,     // versions ít thay đổi
  });
}
```

### 3.3 Forget Rules Editor

```
ForgetRulesPage
├── RulesTable              ← current rules list
│   └── RuleRow             ← type, forget_after, contradiction_resolution
├── AddRuleForm             ← add/edit rule
└── SaveButton              ← PUT /v1/console/adaptive/forget-rules (bulk)
```

---

## 4. UI Design Details

### Salience Score Visualization
```
[████████░░] 80%  — Active (green)
[████░░░░░░] 40%  — Decaying (yellow)  
[█░░░░░░░░░] 10%  — Near eviction (red)
```

### Version Chain Visual
```
v1 ──extends──▶ v2 ──updates──▶ v3 (LATEST)
 │
 └── (superseded 30 days ago)
```

### Memory Status Badges
| Status | Badge |
|--------|-------|
| `is_latest=true` | `LATEST` (green) |
| `is_latest=false` | `SUPERSEDED` (gray) |
| `forget_after=30d` | `⏰ 23d left` (amber) |
| `salience < 0.2` | `DECAY RISK` (red) |

---

## 5. State Management

```typescript
// ui/src/api/hooks/useMemoryLifecycle.ts
export function useLifecycleAnalytics() {
  return useQuery({
    queryKey: ['adaptive', 'analytics'],
    queryFn:  () => adaptiveApi.getAnalytics(),
    refetchInterval: 30_000,
  });
}

export function useForgetRules() {
  return useQuery({
    queryKey: ['adaptive', 'forget-rules'],
    queryFn:  () => adaptiveApi.getForgetRules(),
  });
}

export function useUpdateForgetRules() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rules: ForgetRule[]) => adaptiveApi.updateForgetRules(rules),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'forget-rules'] }),
  });
}
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Memory list hiển thị `is_latest` badge và salience score
- [ ] Version chain drawer hiển thị đúng thứ tự và relation_type
- [ ] Diff viewer hiển thị unified diff giữa các versions
- [ ] Forget rules editor cho phép thêm/sửa/xóa rules
- [ ] Analytics panel cập nhật mỗi 30s
- [ ] Filter by `is_latest`, `engine`, `memory_type`, `forget_after` hoạt động
- [ ] TTL countdown hiển thị dạng relative (ví dụ: "expires in 23 days")
