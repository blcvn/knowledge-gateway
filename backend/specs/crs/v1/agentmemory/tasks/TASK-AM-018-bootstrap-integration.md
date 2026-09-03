# TASK-AM-018 — Bootstrap Integration (Monolith Wire-Up)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-018 |
| **Wave** | 3 (Orchestration) |
| **Component** | `apps/memory/` |
| **Status** | ✅ Done |
| **Solution Ref** | All SOLs — bootstrap sections |
| **Priority** | High |
| **Depends On** | TASK-AM-015, TASK-AM-017 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** bootstrap integration: services start from compose  
---

## Context

Wire tất cả AgentMemory services vào monolith InProcessRegistry. Update config.yaml với các env vars mới. Đảm bảo các goroutine lifecycle được managed đúng.

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `apps/memory/internal/bootstrap/app.go` |
| MODIFY | `apps/memory/configs/config.yaml` |
| CREATE | `apps/memory/internal/config/agentmemory.go` |

---

## Implementation

### MODIFY `apps/memory/internal/bootstrap/app.go`

```go
// Thêm vào func Bootstrap() theo thứ tự dependency:

func Bootstrap(ctx context.Context, cfg *config.Config) error {
    // ... existing init (DB, NATS, etc.) ...

    // [NEW] Wave 1: Foundation services
    InitPrivacyPackage()                           // pkg/privacy (no-op, package init)
    InitObserveSearch(reg, db, nc, cfg)            // am-search (#37)
    InitObserveService(reg, db, nc, cfg)           // am-observe (#36)
    InitAgentMemoryLifecycle(reg, db, nc, cfg)     // extend memory-service with AgentMemory

    // [NEW] Wave 2: Integration
    InitConsolidationPipeline(reg, db, nc, cfg)    // consolidation in memory-service
    // Note: MCP tools auto-register via tool_registry.go

    // [NEW] Wave 3: Orchestration
    InitOrchestration(reg, db, nc, cfg)            // am-orchestration (#38)

    // [NEW] Wave 4: Governance (extends existing services)
    InitGovernanceAudit(reg, db, nc, cfg)
    InitHealthMonitor(db, nc, cfg)

    // Register SSE handler for gateway
    gateway.RegisterSSEHandler(observeSSEBroker)

    return nil
}
```

### MODIFY `apps/memory/configs/config.yaml`

```yaml
# ─── Existing config (unchanged) ──────────────────────────────────────────

# ─── [NEW] AgentMemory configuration ──────────────────────────────────────
agentmemory:
  # Observe service
  max_obs_per_session: 500          # AGENTMEMORY_MAX_OBS_PER_SESSION
  dedup_ttl_secs: 30                # AGENTMEMORY_DEDUP_TTL
  inject_context: false             # AGENTMEMORY_INJECT_CONTEXT=true to enable
  token_budget: 2000                # AGENTMEMORY_TOKEN_BUDGET

  # Memory lifecycle
  strength_default: 0.7             # default strength for new memories
  half_life_days: 30                # AGENTMEMORY_HALF_LIFE_DAYS
  max_memories_project: 1000        # AGENTMEMORY_MAX_MEMORIES_PROJECT

  # Agent scope
  agent_scope: "shared"             # AGENTMEMORY_AGENT_SCOPE: "shared"|"isolated"

  # Consolidation
  consolidation_interval_hours: 2   # AGENTMEMORY_CONSOLIDATION_INTERVAL
  min_procedure_frequency: 3        # minimum repeated workflows to extract
  lesson_half_life_days: 90         # AGENTMEMORY_LESSON_HALF_LIFE

  # Auto-compress
  auto_compress: false              # AGENTMEMORY_AUTO_COMPRESS=true to enable LLM compression

search:
  embedding_provider: "none"        # AGENTMEMORY_EMBEDDING_PROVIDER: "none"|"bifrost"
  embedding_model: "text-embedding-3-small"
  embed_dims: 384                   # AGENTMEMORY_EMBED_DIMS
  data_dir: "${HOME}/.agentmemory/indexes"  # AGENTMEMORY_DATA_DIR

bifrost:
  url: ""                           # AGENTMEMORY_BIFROST_URL (for LLM + embeddings)

# ─── Snapshot ─────────────────────────────────────────────────────────────
snapshot:
  enabled: false                    # AGENTMEMORY_SNAPSHOT_ENABLED
  data_dir: "${HOME}/.agentmemory"
```

### `apps/memory/internal/config/agentmemory.go`

```go
package config

import (
    "os"
    "strconv"
    "time"
)

type AgentMemoryConfig struct {
    // Observe
    MaxObsPerSession int
    DedupTTL         time.Duration
    InjectContext    bool
    TokenBudget      int

    // Memory lifecycle
    StrengthDefault    float64
    HalfLifeDays       int
    MaxMemoriesProject int
    AgentScope         string

    // Consolidation
    ConsolidationIntervalHours int
    MinProcedureFrequency      int
    LessonHalfLifeDays         int
    AutoCompress               bool
}

func LoadAgentMemoryConfig() AgentMemoryConfig {
    return AgentMemoryConfig{
        MaxObsPerSession:           getEnvInt("AGENTMEMORY_MAX_OBS_PER_SESSION", 500),
        DedupTTL:                   time.Duration(getEnvInt("AGENTMEMORY_DEDUP_TTL", 30)) * time.Second,
        InjectContext:               getEnvBool("AGENTMEMORY_INJECT_CONTEXT", false),
        TokenBudget:                 getEnvInt("AGENTMEMORY_TOKEN_BUDGET", 2000),
        StrengthDefault:             0.7,
        HalfLifeDays:                getEnvInt("AGENTMEMORY_HALF_LIFE_DAYS", 30),
        MaxMemoriesProject:          getEnvInt("AGENTMEMORY_MAX_MEMORIES_PROJECT", 1000),
        AgentScope:                  getEnvStr("AGENTMEMORY_AGENT_SCOPE", "shared"),
        ConsolidationIntervalHours:  getEnvInt("AGENTMEMORY_CONSOLIDATION_INTERVAL", 2),
        MinProcedureFrequency:       getEnvInt("AGENTMEMORY_MIN_PROCEDURE_FREQ", 3),
        LessonHalfLifeDays:          getEnvInt("AGENTMEMORY_LESSON_HALF_LIFE", 90),
        AutoCompress:                getEnvBool("AGENTMEMORY_AUTO_COMPRESS", false),
    }
}

func getEnvInt(key string, def int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil { return n }
    }
    return def
}

func getEnvBool(key string, def bool) bool {
    if v := os.Getenv(key); v != "" {
        return v == "true" || v == "1" || v == "yes"
    }
    return def
}

func getEnvStr(key, def string) string {
    if v := os.Getenv(key); v != "" { return v }
    return def
}
```

### Environment Variables Reference

```bash
# Observe Service
AGENTMEMORY_MAX_OBS_PER_SESSION=500   # default: 500
AGENTMEMORY_DEDUP_TTL=30              # seconds, default: 30
AGENTMEMORY_INJECT_CONTEXT=false      # bool, default: false
AGENTMEMORY_TOKEN_BUDGET=2000         # tokens, default: 2000
AGENTMEMORY_AGENT_SCOPE=shared        # "shared" | "isolated"

# Memory Lifecycle
AGENTMEMORY_HALF_LIFE_DAYS=30         # memory decay half-life
AGENTMEMORY_MAX_MEMORIES_PROJECT=1000 # eviction threshold
AGENTMEMORY_AUTO_COMPRESS=false       # enable LLM compression

# Search
AGENTMEMORY_EMBEDDING_PROVIDER=none   # "none" | "bifrost"
AGENTMEMORY_EMBED_DIMS=384
AGENTMEMORY_DATA_DIR=~/.agentmemory/indexes

# Bifrost (LLM)
AGENTMEMORY_BIFROST_URL=              # http://localhost:9000

# Consolidation
AGENTMEMORY_CONSOLIDATION_INTERVAL=2  # hours
AGENTMEMORY_LESSON_HALF_LIFE=90       # days

# Snapshot
AGENTMEMORY_SNAPSHOT_ENABLED=false
```

---

## Verification

```bash
# Build the monolith
cd apps/memory
go build ./...

# Start with defaults (no external deps)
AGENTMEMORY_INJECT_CONTEXT=false \
  go run ./cmd/memory/main.go

# Health check
curl -s http://localhost:8080/v1/health | jq .

# Verify all 3 new services registered
curl -s http://localhost:8080/v1/observe/search/stats | jq .
curl -s http://localhost:8080/v1/observe/sessions | jq .
curl -s http://localhost:8080/v1/orchestration/actions | jq .
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| Monolith builds without errors | ✅ |
| `am-observe` registered → `POST /v1/observe` responds | ✅ |
| `am-search` registered → `GET /v1/observe/search/stats` responds | ✅ |
| `am-orchestration` registered → `GET /v1/orchestration/actions` responds | ✅ |
| Config loaded from env vars | ✅ |
| Consolidation goroutine starts | ✅ |
| Decay scheduler goroutine starts | ✅ |
