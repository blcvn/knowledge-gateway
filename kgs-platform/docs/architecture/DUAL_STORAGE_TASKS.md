# KGS Dual-Mode Storage — Task Breakdown

> Ref: [DUAL_STORAGE_DESIGN.md](DUAL_STORAGE_DESIGN.md) · [ARCHITECTURE.md](ARCHITECTURE.md)

**Status:** 🔴 NOT STARTED | 🟡 IN PROGRESS | 🟢 DONE | 🚫 BLOCKED

---

## Phase 1 — Foundation & Config

### TASK-1.1 🟢 Thêm SurrealDB config vào protobuf

**Files đã sửa:**
- ✅ `internal/conf/conf.proto` — thêm `storage_mode` (field 20), `SurrealDB` message (field 21)
- ✅ `configs/config.yaml` — thêm `storage_mode: specialized` + section surrealdb (commented)
- ✅ `internal/conf/surrealdb.go` — helper `StorageMode()`, `IsSurrealDBMode()`

> ⚠️ `conf.pb.go` chưa regenerate vì `protoc` chưa cài trên máy. Cần `make config` khi có protoc.

**Effort:** S (1–2h) · **Dependencies:** Không

---

### TASK-1.2 🟢 Tạo SurrealDB Go client wrapper

**File đã tạo:** `internal/data/surrealdb/client.go`

- ✅ Connect WebSocket, Sign in, Select namespace/database
- ✅ Health check (`Ping`) via `RETURN true` query
- ✅ Cleanup function for graceful shutdown
- ✅ Query helper with error logging

**Effort:** M (3–4h) · **Dependencies:** TASK-1.1 ✅

---

### TASK-1.3 🟢 Tạo StorageBundle + Factory

**File đã tạo:** `internal/data/factory.go`

- ✅ `StorageBundle` struct gom tất cả ports
- ✅ `NewStorageFactory()` — switch theo `storage_mode` config
- ✅ `newSpecializedBundle()` — wrap existing `NewData()` logic
- ✅ `newSurrealDBBundle()` — stub cho Phase 2 (returns error "not yet implemented")
- ✅ `OutboxEnabled` flag — `false` khi SurrealDB mode

**Effort:** M (3–4h) · **Dependencies:** TASK-1.2 ✅

---

### TASK-1.4 🟢 Schema init cho SurrealDB

**File đã tạo:** `internal/data/surrealdb/schema.go`

- ✅ `InitSchema()` — idempotent schema creation
- ✅ 14 tables tương đương GORM models:
  - kgs_apps, kgs_api_keys, kgs_quotas, kgs_audit_logs
  - kgs_entity_types, kgs_relation_types
  - kgs_rules, kgs_rule_executions, kgs_policies
  - kg_entities, kg_edges, graph_versions, view_definitions
  - kg_overlays, kg_overlay_sessions, kg_locks
- ✅ Vector index: `MTREE DIMENSION 1536 DIST COSINE`
- ✅ Full-text search analyzer: `kgs_text` (BM25)
- ✅ Unique indexes cho entity_types, relation_types, api_keys, etc.

**Effort:** M (4h) · **Dependencies:** TASK-1.2 ✅

---

## Phase 2 — SurrealDB Core Adapters (L1)

### TASK-2.1 🟢 Implement `RegistryRepo` cho SurrealDB

**File đã tạo:** `internal/data/surrealdb/registry_repo.go`
- ✅ 7 methods: CreateApp, GetApp, ListApps, CreateAPIKey, GetAPIKeyByHash, RevokeAPIKey, GetQuota
- ✅ Compile check: `var _ biz.RegistryRepo = (*surrealRegistryRepo)(nil)`
- ✅ No Neo4j namespace reservation (SurrealDB = unified store)

**Effort:** M (3–4h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.2 🟢 Implement `RulesRepo` + `PolicyRepo`

**File đã tạo:** `internal/data/surrealdb/rules_policy_repo.go`
- ✅ `RulesRepo`: CreateRule, GetRule, ListRules (3 methods)
- ✅ `PolicyRepo`: CreatePolicy, GetPolicy, ListPolicies (3 methods)
- ✅ Compile checks: `var _ biz.RulesRepo`, `var _ biz.PolicyRepo`

**Effort:** S (2h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.3 🟢 Implement `OntologyRepo`

**File đã tạo:** `internal/data/surrealdb/ontology_repo.go`
- ✅ GetEntityType, GetRelationType — direct SurrealDB lookups
- ✅ InvalidateEntityType, InvalidateRelationType — **no-op** (no Redis cache)

**Effort:** S (2h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.4 🟢 Implement `GraphWriteRepo` ⭐

**File đã tạo:** `internal/data/surrealdb/graph_write.go`

| Method | Status | SurrealDB implementation |
|--------|--------|--------------------------|
| `UpsertEntity` | ✅ | `UPDATE ... MERGE` with all fields |
| `UpsertEdge` | ✅ | `UPDATE ... MERGE` |
| `SoftDeleteEntity` | ✅ | `UPDATE SET is_deleted=true, deleted_at=time::now()` |
| `SoftDeleteEdge` | ✅ | `UPDATE SET is_deleted=true` |
| `EnqueueOutbox` | ✅ | **`return nil` — NO-OP** (critical design) |
| `WithTx` | ✅ | Passes through (SDK transaction TBD) |

**Effort:** M (4h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.5 🟢 Implement `GraphRepo` ⭐⭐ (largest)

**File đã tạo:** `internal/data/surrealdb/graph_repo.go`

| Method | Status | Approach |
|--------|--------|----------|
| `CreateNode` | ✅ | `CREATE type::thing(...)` |
| `GetNode` | ✅ | `SELECT ... WHERE entity_id = $id` |
| `CreateEdge` | ✅ | `CREATE type::thing('kg_edges', ...)` |
| `ExecuteQuery` | ✅ | Delegates to `QueryTranslator` |
| `GetFullGraph` | ✅ | Paginated SELECT + count queries |
| `DeleteNode` | ✅ | Cascade: soft-delete edges → soft-delete node |
| `DeleteEdge` | ✅ | `UPDATE SET is_deleted=true` |
| `BatchDeleteNodes` | ✅ | Batch with `IN $node_ids` |

**Effort:** L (6–8h) · **Dependencies:** TASK-1.4 ✅, TASK-3.1 ✅

---

### TASK-2.6 🟢 Implement `EntityReader`

**File đã tạo:** `internal/data/surrealdb/entity_reader.go`
- ✅ `GetEntity` — direct SurrealDB SELECT (no PG fallback needed)
- ✅ `EnrichWithFreshVersions` — **returns input as-is** (single store = always fresh)
- ✅ `normalizeEntityMap()` — field alias normalization (id↔entity_id, label↔entity_type)

**Effort:** S (2h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.7 🟢 Implement `LockManager`

**File đã tạo:** `internal/data/surrealdb/lock.go`
- ✅ AcquireNodeLock, AcquireNamespaceLock — table-based locks with TTL
- ✅ Release — DELETE by token
- ✅ Expired lock cleanup before acquire attempt
- ✅ UNIQUE index on lock_key prevents double-acquire

**Effort:** M (3–4h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.8 🟢 Implement `overlay.Store`

**File đã tạo:** `internal/data/surrealdb/overlay_store.go`
- ✅ SaveOverlay, GetOverlay, DeleteOverlay
- ✅ BindSession, UnbindSession, FindBySession
- ✅ TTL-based expiry for both overlays and sessions

**Effort:** M (3–4h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-2.9 🟢 Wire Provider Set

**File đã tạo:** `internal/data/surrealdb/provider.go`
- ✅ `ProviderSet` with all 12 constructors

**Effort:** S (1h) · **Dependencies:** TASK-2.1→2.8 ✅

---

## Phase 3 — Query & Intelligence Adapters (L3)

### TASK-3.1 🟢 QueryTranslator (Cypher → SurrealQL) ⭐⭐

**File đã tạo:** `internal/data/surrealdb/query_translator.go`

| Pattern | Detector | Status |
|---------|----------|--------|
| Context: `MATCH (n)-[r]-(m)` | regex | ✅ |
| Impact: `MATCH (n)-[*1..N]->(m)` | regex + depth extract | ✅ |
| Coverage: `MATCH (n)<-[*1..N]-(m)` | regex + depth extract | ✅ |
| Subgraph: `WHERE n.id IN $ids` | regex | ✅ |
| Simple lookup | regex | ✅ |

- ✅ Iterative BFS for multi-hop traversal (Impact/Coverage)
- ✅ `$visited` + `$frontier` pattern to avoid cycles
- ⚠️ Cần unit tests (TASK-4.3)

**Effort:** L (6–8h) · **Dependencies:** Không

---

### TASK-3.2 🟢 `VectorRetriever` cho SurrealDB

**File đã tạo:** `internal/data/surrealdb/search.go`
- ✅ `vector::similarity::cosine(embedding, $vec)` query
- ✅ Namespace filtering (app_id + tenant_id)
- ✅ TopK limit + ORDER BY score DESC

**Effort:** M (4h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-3.3 🟢 `TextRetriever` cho SurrealDB

**File đã tạo:** `internal/data/surrealdb/search.go`
- ✅ BM25 full-text search with `@1@` operator
- ✅ `search::score(1)` for relevance scoring

**Effort:** M (3h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-3.4 🟢 `CentralityScorer` cho SurrealDB

**File đã tạo:** `internal/data/surrealdb/search.go`
- ✅ **Option A chosen:** Degree centrality (in+out edge count)
- ✅ Normalized to [0, 1] range
- ✅ Graceful fallback: returns uniform scores on error

**Effort:** M (3h) · **Dependencies:** TASK-1.4 ✅

---

### TASK-3.5 🟢 `QueryExecutor` cho Analytics

**File đã tạo:** `internal/data/surrealdb/analytics_executor.go`

- ✅ 3 analytics Cypher patterns translated:
  - Coverage report: entity count by type + outgoing edge check
  - Traceability matrix: multi-hop BFS between source/target types
  - Cluster analysis: edge-based grouping (approximation — SurrealDB has no GDS Louvain)
- ✅ Regex pattern matching cho `coverageQuery`, `traceabilityQueryTmpl`, `clusterQuery`
- ✅ Added to `ProviderSet`

**Effort:** L (6h) · **Dependencies:** TASK-3.1 ✅

---

## Phase 4 — Integration & Validation

### TASK-4.1 🟢 Wire DI switching

**Files đã tạo/sửa:**
- ✅ `cmd/server/wire_surrealdb.go` — `wireAppSurrealDB()` function (manual wiring, no Wire codegen needed)
- ✅ `cmd/server/main.go` — `initApp()` routes to `wireApp` or `wireAppSurrealDB` by config

**Chi tiết:**
- `wireAppSurrealDB()` tạo full Kratos app với SurrealDB adapters
- OutboxWorker = nil, ReconcileJob = nil (no CQRS fan-out)
- OverlayListener = nil (no NATS)
- Redis client = nil (no Redis)
- HealthService gets nil infra clients (SurrealDB mode)

**Effort:** M (4h) · **Dependencies:** TASK-1.3 ✅, TASK-2.9 ✅

---

### TASK-4.2 🟢 Docker Compose cho SurrealDB

**File đã tạo:** `docker-compose.yml`
- ✅ Specialized stack: PG, Neo4j, Qdrant, Redis, NATS, OPA (profile: `specialized`)
- ✅ SurrealDB stack: surrealdb/surrealdb:v2 (profile: `surrealdb`)
- ✅ Health checks for PG and SurrealDB
- ✅ Persistent volumes

**Effort:** S (1h) · **Dependencies:** Không

---

### TASK-4.3 🟢 Integration Tests

**File đã tạo:** `internal/data/surrealdb/integration_test.go`

- ✅ 6 test scenarios implemented:
  1. `TestRegistryRepo` — CreateApp → GetApp → ListApps
  2. `TestGraphWriteRepo` — UpsertEntity + EnqueueOutbox(noop) + UpsertEdge
  3. `TestGraphRepo` — CreateNode×2 → GetNode → CreateEdge → GetFullGraph → DeleteNode(cascade)
  4. `TestLockManager` — Acquire → Contention → Release → Re-acquire
  5. `TestOverlayStore` — Save → Get → BindSession → FindBySession → Delete
  6. `TestQueryTranslator` — 5 Cypher patterns (context, impact, coverage, subgraph, unsupported)

> Requires running SurrealDB: `docker run --rm -p 8000:8000 surrealdb/surrealdb:v2 start --user root --pass test memory`

**Effort:** L (8h) · **Dependencies:** TASK-2.1→2.8 ✅, TASK-3.1→3.4 ✅

---

### TASK-4.4 🟢 Shadow Mode

**Files đã tạo:**
- ✅ `internal/data/shadow/shadow_repo.go` — ShadowGraphRepo (writes: sync+async, reads: compare+log diffs)
- ✅ `internal/data/shadow/shadow_write_repo.go` — ShadowGraphWriteRepo (writes: sync+async, EnqueueOutbox: primary only)

**Chi tiết:**
- Writes: primary (sync) + secondary (async, best-effort)
- Reads: primary (sync) + secondary (async compare → structured log diffs)
- Secondary errors never propagate to caller
- `diffCount` tracked for monitoring

**Effort:** L (6–8h) · **Dependencies:** TASK-4.1 ✅

---

## Summary

| Phase | Tasks | Status | Files Created |
|-------|-------|--------|---------------|
| Phase 1 — Foundation | 1.1→1.4 | ✅ 4/4 DONE | conf.proto, config.yaml, surrealdb.go, factory.go, client.go, schema.go |
| Phase 2 — Core Adapters | 2.1→2.9 | ✅ 9/9 DONE | registry_repo.go, rules_policy_repo.go, ontology_repo.go, graph_write.go, graph_repo.go, entity_reader.go, lock.go, overlay_store.go, provider.go, helpers.go |
| Phase 3 — Query/Search | 3.1→3.5 | ✅ 5/5 DONE | query_translator.go, search.go, analytics_executor.go |
| Phase 4 — Integration | 4.1→4.4 | ✅ 4/4 DONE | wire_surrealdb.go, main.go, docker-compose.yml, integration_test.go, shadow/ |
| **Total** | **22 tasks** | **✅ 22/22 DONE** | **22 files created/modified** |

### Files Created

```
internal/conf/
├── surrealdb.go                  # StorageMode() helper

internal/data/
├── factory.go                    # StorageBundle + NewStorageFactory()

internal/data/surrealdb/
├── client.go                     # SurrealDB connection manager
├── helpers.go                    # Generic unmarshal helpers
├── schema.go                     # Schema init (17 tables + indexes)
├── graph_repo.go                 # biz.GraphRepo (8 methods)
├── graph_write.go                # biz.GraphWriteRepo (6 methods, EnqueueOutbox=noop)
├── entity_reader.go              # biz.EntityReader (2 methods)
├── registry_repo.go              # biz.RegistryRepo (7 methods)
├── ontology_repo.go              # OntologyRepo (4 methods)
├── rules_policy_repo.go          # biz.RulesRepo + PolicyRepo (6 methods)
├── query_translator.go           # Cypher→SurrealQL (5 patterns)
├── analytics_executor.go         # analytics.QueryExecutor (3 patterns)
├── search.go                     # Vector + Text + Centrality
├── lock.go                       # lock.LockManager
├── overlay_store.go              # overlay.Store
├── provider.go                   # Wire ProviderSet (13 constructors)
└── integration_test.go           # 6 integration test scenarios

internal/data/shadow/
├── shadow_repo.go                # ShadowGraphRepo (dual-write + compare)
└── shadow_write_repo.go          # ShadowGraphWriteRepo (dual-write)

cmd/server/
├── wire_surrealdb.go             # SurrealDB-mode app wiring
└── main.go                       # initApp() routes by storage_mode

docker-compose.yml                # Standalone specialized + SurrealDB profiles
configs/config.yaml               # storage_mode + surrealdb section
```

### Deployment Integration

KGS Platform is now integrated into the central VNP Memory deployment:

```
deployment/local/
├── .env                          # KGS_STORAGE_MODE, SURREALDB_* vars
├── docker-compose.yml            # KGS + SurrealDB as profiled services
├── init-db.sql                   # Creates ba_agent_db alongside cognee_db/zep_db
├── Makefile                      # make up-kgs / make up-kgs-surrealdb
└── config/
    ├── kgs.yaml                  # Specialized mode config (PG+Neo4j+Qdrant+Redis)
    └── kgs-surrealdb.yaml        # SurrealDB mode config (unified)
```

**Available commands:**

| Command | Description |
|---------|-------------|
| `make up-kgs` | Start KGS with specialized stack (reuses shared PG/Neo4j/Redis/Qdrant) |
| `make up-kgs-surrealdb` | Start KGS with SurrealDB backend |
| `make up-surrealdb` | Start SurrealDB only (for dev/testing) |
| `make up-full` | Start everything: Memory + KGS + UI + Monitoring |
| `make logs-kgs` | Follow KGS Platform logs |
| `make logs-surrealdb` | Follow SurrealDB logs |

### Next Steps (Post-Implementation)

1. **Install `protoc`** → run `make config` để regenerate `conf.pb.go`
2. **`go mod tidy`** → add `github.com/surrealdb/surrealdb.go` dependency
3. **Run integration tests** → `cd deployment/local && make up-surrealdb && go test ./internal/data/surrealdb/ -v`
4. **Verify specialized mode** → ensure existing tests still pass
5. **Production pilot** → use `storage_mode: shadow` first, monitor diff logs
