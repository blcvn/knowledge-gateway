# Change Request: CR-AM-002 — Memory Lifecycle Management

**CR ID:** CR-AM-002  
**Component:** `services/memory-service` [EXTEND EXISTING]  
**Priority:** Critical  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.3, SRS FR-MEM-001..005, FR-SLOTS-001..002  
**Spec:** `references/agentmemory/specs/services/memory-service/spec.md`

---

## 1. Mô tả

Bổ sung hệ thống **Memory Lifecycle** đầy đủ vào `services/memory-service` hiện tại của VNP Memory:
1. **Jaccard-based Versioning** — khi memory mới có similarity > 0.7 với memory cũ → supersede (thay thế) thay vì tạo duplicate.
2. **Strength & Decay** — mỗi memory có `strength` score, decay theo thời gian.
3. **TTL Auto-Forget** — `forgetAfter` timestamp, sweep định kỳ.
4. **Eviction Policy** — formula `importance × recency × frequency` để prune stale memories.
5. **Memory Slots** — named, editable working memory blocks với size limit.
6. **6 Memory Types** — `pattern`, `preference`, `architecture`, `bug`, `workflow`, `fact`.

---

## 2. Vấn đề hiện tại

`services/memory-service` hiện tại (Memobase engine) hỗ trợ:
- ✅ Basic ingest và query
- ❌ Không có Jaccard versioning (duplicate memories tích tụ)
- ❌ Không có `strength` field và decay
- ❌ Không có TTL `forgetAfter`
- ❌ Không có eviction scoring
- ❌ Không có Memory Slots (named editable blocks)
- ❌ Chỉ có 1 generic memory type (Memobase profiles), thiếu `pattern`, `bug`, `workflow`, `architecture`

---

## 3. Thay đổi đề xuất

### 3.1. [MODIFY] Domain Model

**File:** `services/memory-service/internal/domain/entity.go`

```go
// AgentMemory — long-term memory entry (agentmemory model)
type AgentMemory struct {
    ID                  string
    CreatedAt           time.Time
    UpdatedAt           time.Time
    Type                MemoryType  // pattern|preference|architecture|bug|workflow|fact
    Title               string
    Content             string
    Concepts            []string
    Files               []string
    SessionIDs          []string
    Strength            float64     // 0.0–1.0, decays over time
    Version             int
    ParentID            string      // previous version ID
    Supersedes          []string    // IDs of superseded memories
    RelatedIDs          []string
    SourceObservationIDs []string
    IsLatest            bool
    ForgetAfter         *time.Time  // TTL
    AgentID             string
    Project             string
    TenantID            string
}

type MemoryType string
const (
    MemTypePattern      MemoryType = "pattern"
    MemTypePreference   MemoryType = "preference"
    MemTypeArchitecture MemoryType = "architecture"
    MemTypeBug          MemoryType = "bug"
    MemTypeWorkflow     MemoryType = "workflow"
    MemTypeFact         MemoryType = "fact"
)

// MemorySlot — named editable working memory block
type MemorySlot struct {
    Label       string
    Content     string
    Scope       string      // "project" | "global"
    Description string
    SizeLimit   int         // chars, 0 = no limit
    Pinned      bool        // immune to eviction
    ReadOnly    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
    TenantID    string
    Project     string
}
```

### 3.2. [NEW] `internal/usecase/remember_agent.go` — Jaccard Versioning

```go
const JaccardThreshold = 0.7

type RememberAgentRequest struct {
    Type        MemoryType
    Title       string
    Content     string
    Concepts    []string
    Files       []string
    SessionID   string
    Strength    float64    // default 0.7
    ForgetAfter *time.Time
    Project     string
    TenantID    string
}

// RememberAgent flow:
// 1. Load existing memories of same Type + Project (isLatest=true only)
// 2. Compute Jaccard(m.Concepts, req.Concepts) for each
// 3. If any > 0.7 → mark as superseded (isLatest=false)
// 4. Create new memory with version = max(superseded versions) + 1
// 5. Write to DB + notify search index
// 6. Return: {memory_id, superseded_ids[], version}
```

**Jaccard Similarity:**
```go
func jaccardSimilarity(a, b []string) float64 {
    setA := toLowerSet(a)
    setB := toLowerSet(b)
    intersection := 0
    for k := range setA {
        if setB[k] { intersection++ }
    }
    union := len(setA) + len(setB) - intersection
    if union == 0 { return 0 }
    return float64(intersection) / float64(union)
}
```

### 3.3. [NEW] `internal/usecase/evict.go` — Eviction Policy

```go
// Eviction scoring formula:
// score = importance × recency × frequency
//   importance = memory.Strength
//   recency    = exp(-daysSinceUpdate / 30.0)   // 30-day half-life
//   frequency  = log1p(len(sessionIDs))           // session access count proxy

type EvictRequest struct {
    MaxMemories int
    Project     string
    DryRun      bool
}

// Sort by score ascending → evict lowest score first
// Pinned memories (strength >= 1.0) are immune
// Only isLatest memories are evicted (history preserved)
```

### 3.4. [NEW] `internal/usecase/auto_forget.go` — TTL Sweep

```go
// AutoForget: runs every 60 minutes via background goroutine
// Scans all memories where forgetAfter < now
// Cascade delete: DB + BM25 index + vector index + graph edges
// Publishes: agentmemory.memory.expired NATS event
```

### 3.5. [NEW] `internal/usecase/slots.go` — Memory Slots

```go
// WriteSlot: create or update named slot
//   - Mode "replace": overwrite content
//   - Mode "append": append content with newline separator
//   - SizeLimit enforcement: truncate if exceeded
//   - ReadOnly check: reject write if readOnly=true
// ReadSlot: get slot by scope + label
// DeleteSlot: remove slot
// ListSlots: paginated listing, filter by scope/project
```

### 3.6. [NEW] `internal/usecase/retention.go` — Retention Scoring

```go
// GetRetentionScore: compute score for a specific memory
// Returns: {score, recency_factor, frequency_factor, importance_factor, days_since_access, recommend_action}
// recommend_action: "keep" (>0.3) | "review" (0.1-0.3) | "evict" (<0.1)
```

### 3.7. New API Endpoints

**[MODIFY]** Gateway + Service — thêm routes mới:

```
POST /v1/memory/agent/remember              # Jaccard-versioned remember
GET  /v1/memory/agent/list                  # List (filter: type, project, isLatest)
GET  /v1/memory/agent/{id}                  # Get memory detail
DELETE /v1/memory/agent/{id}                # Governance delete + cascade
GET  /v1/memory/agent/{id}/retention        # Get retention score
POST /v1/memory/agent/evict                 # Run eviction policy
POST /v1/memory/agent/auto-forget           # TTL sweep

GET  /v1/memory/slots                       # List slots
GET  /v1/memory/slots/{scope}/{label}       # Get slot
POST /v1/memory/slots/{scope}/{label}       # Write/update slot
DELETE /v1/memory/slots/{scope}/{label}     # Delete slot
```

### 3.8. Strength Decay Scheduler

```go
// Background goroutine (every 2h) runs decay sweep:
// For each memory: strength × decay_factor
// decay_factor = exp(-hoursSinceLastAccess / (HALF_LIFE_DAYS * 24))
// If strength drops below threshold (0.05): schedule for eviction
```

### 3.9. NATS Events

| Subject | Direction | Payload |
|---|---|---|
| `agentmemory.memory.remembered` | Publish | `{memory_id, type, version, superseded[]}` |
| `agentmemory.memory.superseded` | Publish | `{old_id, new_id}` |
| `agentmemory.memory.expired` | Publish | `{memory_id, reason: "ttl"|"eviction"}` |

---

## 4. Database Changes

**[NEW]** PostgreSQL table: `agent_memories`
```sql
CREATE TABLE agent_memories (
    id              UUID PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    project         TEXT,
    type            TEXT NOT NULL,          -- pattern|preference|...
    title           TEXT NOT NULL,
    content         TEXT NOT NULL,
    concepts        TEXT[],
    files           TEXT[],
    session_ids     TEXT[],
    strength        FLOAT8 DEFAULT 0.7,
    version         INT DEFAULT 1,
    parent_id       UUID REFERENCES agent_memories(id),
    supersedes      UUID[],
    related_ids     UUID[],
    is_latest       BOOLEAN DEFAULT true,
    forget_after    TIMESTAMPTZ,
    agent_id        TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_agent_memories_tenant_type_latest ON agent_memories(tenant_id, type, is_latest);
CREATE INDEX idx_agent_memories_forget_after ON agent_memories(forget_after) WHERE forget_after IS NOT NULL;
```

**[NEW]** PostgreSQL table: `memory_slots`
```sql
CREATE TABLE memory_slots (
    tenant_id   TEXT NOT NULL,
    project     TEXT,
    scope       TEXT NOT NULL,   -- "project" | "global"
    label       TEXT NOT NULL,
    content     TEXT,
    description TEXT,
    size_limit  INT DEFAULT 0,
    pinned      BOOLEAN DEFAULT false,
    read_only   BOOLEAN DEFAULT false,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, scope, label)
);
```

---

## 5. Acceptance Criteria

- [x] `POST /v1/memory/agent/remember` với concepts `["jose", "jwt", "auth"]`: trả về `{version: 1}`.
- [x] Submit lần 2 với concepts `["jose", "jsonwebtoken", "auth"]` (Jaccard ~0.67): **không** supersede (< 0.7 threshold).
- [x] Submit lần 2 với concepts `["jose", "jwt", "auth", "middleware"]` (Jaccard > 0.7): trả về `{version: 2, superseded: [id1]}`.
- [x] Memory cũ trong DB có `is_latest=false` sau khi bị supersede.
- [x] Memory với `forget_after` trong quá khứ: sau auto-forget sweep → bị xóa khỏi DB + indexes.
- [x] `POST /v1/memory/agent/evict` với `max_memories: 10, dry_run: true`: trả về list IDs sẽ bị evict, DB không thay đổi.
- [x] Memory có `strength >= 1.0` (pinned): không bị evict.
- [x] `POST /v1/memory/slots/project/conventions` với `mode: "append"`: nội dung được nối vào existing slot.
- [x] `GET /v1/memory/agent/{id}/retention` trả về `{score, recommend_action: "keep"|"review"|"evict"}`.
