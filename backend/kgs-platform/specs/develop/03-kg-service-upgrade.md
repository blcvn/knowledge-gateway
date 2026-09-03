# kg-service Upgrade — KGS Graph + Ontology + Overlay + Outbox

> **Strategy:** 🔄 UPGRADE existing `services/kg-service/`  
> **Absorbs:** graph-service + ontology-service + sync-worker-service + overlay-service  
> **Effort:** 8 ngày  
> **Priority:** P0

---

## 1. Tại Sao kg-service Là Đúng Chỗ

`kg-service` đã có:
- `IngestUseCase` → episode ingestion, entity extraction
- `StoreUseCase` → node/edge CRUD (GetNode, GetEdge)
- `KnowledgeUseCase` → ontology management, subgraph queries
- `KGHandler` adapter với `ForwardService` pattern sẵn có
- Infrastructure cho PostgreSQL, NATS, Qdrant (một phần)

**KGS-spec yêu cầu thêm vào kg-service:**
- Namespace-aware graph CRUD (CreateNode, CreateEdge, DeleteNode, DeleteEdge, BatchDelete)
- OPA policy enforcement per operation
- Ontology schema validation via JSON Schema
- Overlay/Draft session management (Redis-backed)
- Outbox sync worker (PostgreSQL → Neo4j + Qdrant, background goroutine)

---

## 2. Cấu Trúc Sau Upgrade

```
services/kg-service/
├── cmd/server/
│   └── main.go              [MODIFY] Wire KGS deps + start background workers
│
├── internal/
│   ├── usecase/
│   │   ├── graphiti/        [UNCHANGED] IngestUseCase, StoreUseCase, SearchUseCase
│   │   ├── cognee/          [UNCHANGED] DatasetUseCase, CognifyUseCase, CogneeSearchUseCase
│   │   └── kgs/             [NEW PACKAGE]
│   │       ├── graph.go         ← Từ kgs-platform/internal/biz/graph.go
│   │       ├── graph_write.go   ← Từ biz/graph_write.go
│   │       ├── namespace.go     ← Từ biz/namespace.go
│   │       ├── guardrails.go    ← Từ biz/graph_guardrails.go
│   │       ├── ontology.go      ← Từ biz/ontology.go + ontology_validator.go
│   │       ├── ontology_sync.go ← Từ biz/ontology_sync.go
│   │       ├── overlay.go       ← Từ kgs-platform/internal/overlay/
│   │       └── policy.go        ← Wrapper: gọi local OPA hoặc remote policy-in-pipeline
│   │
│   ├── domain/
│   │   ├── graphiti/        [UNCHANGED]
│   │   ├── cognee/          [UNCHANGED]
│   │   └── kgs/             [NEW]
│   │       ├── entity.go    ← Từ data/models_kg.go (KGEntity, KGEdge, KGSyncOutbox)
│   │       └── events.go    ← Từ data/nats_topics.go (NATS topic constants)
│   │
│   ├── adapter/
│   │   └── grpc/
│   │       ├── router.go        [MODIFY] Thêm KGS routes
│   │       ├── kgs_graph.go     [NEW] /v1/graph/** handlers
│   │       ├── kgs_ontology.go  [NEW] /v1/ontology/** handlers
│   │       └── kgs_overlay.go   [NEW] /v1/overlay/** handlers
│   │
│   └── infra/
│       ├── postgres/        [EXTEND]
│       │   ├── entity_pg.go     ← Từ data/entity_pg.go
│       │   ├── edge_pg.go       ← Từ data/edge_pg.go
│       │   ├── graph_write.go   ← Từ data/graph_write_pg.go
│       │   ├── entity_reader.go ← Từ data/entity_reader*.go
│       │   ├── outbox.go        ← Từ data/outbox.go
│       │   └── ontology.go      ← Từ data/ontology.go
│       ├── neo4j/           [NEW]
│       │   ├── client.go        ← Từ data/graph_node.go + graph_edge.go
│       │   └── constraints.go   ← Từ data/neo4j_constraints.go
│       ├── qdrant/          [NEW]
│       │   └── client.go        ← Từ data/qdrant.go
│       ├── redis/           [NEW]
│       │   └── lock.go          ← Từ internal/lock/
│       ├── outbox/          [NEW]
│       │   └── worker.go        ← Từ internal/outbox/ + internal/batch/
│       └── nats/            [EXTEND]
│           └── publisher.go     ← Từ data/nats.go
│
└── migrations/
    └── kgs/
        ├── 001_entities.sql     [NEW] kg_entities table
        ├── 002_edges.sql        [NEW] kg_edges table
        ├── 003_outbox.sql       [NEW] kg_sync_outbox table
        └── 004_ontology.sql     [NEW] entity_types, relation_types tables
```

---

## 3. Routes Mới Thêm Vào router.go

```go
// internal/adapter/grpc/router.go — THÊM VÀO (không xóa routes cũ)

func RegisterRoutes(router *forward.Router, h *KGHandler) {
    // ── Legacy routes (UNCHANGED) ──
    router.Handle("POST", "/v1/graphiti/episodes", ...)     // giữ nguyên
    router.Handle("GET",  "/v1/graphiti/nodes/*", ...)      // giữ nguyên
    // ... tất cả routes cũ

    // ── KGS Graph API (NEW) ──
    router.Handle("POST",   "/v1/graph/nodes",        h.adaptHTTP(h.KGSCreateNode))
    router.Handle("GET",    "/v1/graph/nodes/{id}",   h.adaptHTTP(h.KGSGetNode))
    router.Handle("PUT",    "/v1/graph/nodes/{id}",   h.adaptHTTP(h.KGSUpdateNode))
    router.Handle("DELETE", "/v1/graph/nodes/{id}",   h.adaptHTTP(h.KGSDeleteNode))
    router.Handle("POST",   "/v1/graph/nodes/batch",  h.adaptHTTP(h.KGSBatchCreateNodes))
    router.Handle("DELETE", "/v1/graph/nodes/batch",  h.adaptHTTP(h.KGSBatchDeleteNodes))
    router.Handle("POST",   "/v1/graph/edges",        h.adaptHTTP(h.KGSCreateEdge))
    router.Handle("DELETE", "/v1/graph/edges/{id}",   h.adaptHTTP(h.KGSDeleteEdge))
    router.Handle("GET",    "/v1/graph",              h.adaptHTTP(h.KGSGetFullGraph))

    // ── KGS Ontology API (NEW) ──
    router.Handle("POST",   "/v1/ontology/entity-types",        h.adaptHTTP(h.KGSCreateEntityType))
    router.Handle("GET",    "/v1/ontology/entity-types",        h.adaptHTTP(h.KGSListEntityTypes))
    router.Handle("GET",    "/v1/ontology/entity-types/{name}", h.adaptHTTP(h.KGSGetEntityType))
    router.Handle("PUT",    "/v1/ontology/entity-types/{name}", h.adaptHTTP(h.KGSUpdateEntityType))
    router.Handle("DELETE", "/v1/ontology/entity-types/{name}", h.adaptHTTP(h.KGSDeleteEntityType))
    router.Handle("POST",   "/v1/ontology/relation-types",      h.adaptHTTP(h.KGSCreateRelationType))
    router.Handle("GET",    "/v1/ontology/relation-types",      h.adaptHTTP(h.KGSListRelationTypes))
    router.Handle("DELETE", "/v1/ontology/relation-types/{name}",h.adaptHTTP(h.KGSDeleteRelationType))
    router.Handle("GET",    "/v1/ontology",                     h.adaptHTTP(h.KGSGetFullOntology))

    // ── KGS Overlay API (NEW) ──
    router.Handle("POST",   "/v1/overlay",                        h.adaptHTTP(h.KGSCreateSession))
    router.Handle("GET",    "/v1/overlay",                        h.adaptHTTP(h.KGSListSessions))
    router.Handle("GET",    "/v1/overlay/{id}",                   h.adaptHTTP(h.KGSGetSession))
    router.Handle("DELETE", "/v1/overlay/{id}",                   h.adaptHTTP(h.KGSDiscardSession))
    router.Handle("POST",   "/v1/overlay/{id}/deltas/entity",     h.adaptHTTP(h.KGSAddEntityDelta))
    router.Handle("POST",   "/v1/overlay/{id}/deltas/edge",       h.adaptHTTP(h.KGSAddEdgeDelta))
    router.Handle("GET",    "/v1/overlay/{id}/deltas",            h.adaptHTTP(h.KGSListDeltas))
    router.Handle("POST",   "/v1/overlay/{id}/commit",            h.adaptHTTP(h.KGSCommitSession))
}
```

---

## 4. Handler mẫu (kgs_graph.go)

```go
// internal/adapter/grpc/kgs_graph.go
package grpc

import (
    "encoding/json"
    "net/http"
    
    "vnp-memory/services/kg-service/internal/usecase/kgs"
)

// kgsGraphHandler chứa KGS graph use case
type kgsGraphHandler struct {
    graph    *kgs.GraphUsecase
    ontology *kgs.OntologyUsecase
    overlay  *kgs.OverlayUsecase
}

func (h *KGHandler) KGSCreateNode(w http.ResponseWriter, r *http.Request) {
    appID    := r.Header.Get("X-App-ID")
    tenantID := r.Header.Get("X-Tenant-ID")

    if appID == "" {
        // Backward compat: nếu không có X-App-ID, dùng X-Tenant-ID
        appID = tenantID
    }

    var req struct {
        EntityType string         `json:"entity_type"`
        Properties map[string]any `json:"properties"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    result, err := h.kgs.graph.CreateNode(r.Context(), appID, tenantID, req.EntityType, req.Properties)
    if err != nil {
        writeKGSError(w, err)
        return
    }
    writeJSON(w, http.StatusCreated, result)
}

func (h *KGHandler) KGSDeleteNode(w http.ResponseWriter, r *http.Request) {
    appID    := r.Header.Get("X-App-ID")
    tenantID := r.Header.Get("X-Tenant-ID")
    nodeID   := r.PathValue("id")

    edgesRemoved, err := h.kgs.graph.DeleteNode(r.Context(), appID, tenantID, nodeID)
    if err != nil {
        writeKGSError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "deleted":       true,
        "edges_removed": edgesRemoved,
    })
}

// writeKGSError maps domain errors to HTTP status codes
func writeKGSError(w http.ResponseWriter, err error) {
    // Map từ biz/errors.go error types
    switch {
    case isNotFound(err):
        writeError(w, http.StatusNotFound, err.Error())
    case isForbidden(err):
        writeError(w, http.StatusForbidden, err.Error())
    case isValidation(err):
        writeError(w, http.StatusBadRequest, err.Error())
    default:
        writeError(w, http.StatusInternalServerError, err.Error())
    }
}
```

---

## 5. main.go Update

```go
// cmd/server/main.go — EXTEND (không xóa code cũ)
func main() {
    // ── Existing dependencies (UNCHANGED) ──
    db := setupPostgres(conf)
    natsConn := setupNATS(conf)
    // ... existing setup

    // ── NEW: KGS infrastructure ──
    neo4jDriver  := setupNeo4j(conf.Neo4j)
    qdrantClient := setupQdrant(conf.Qdrant)
    redisClient  := setupRedis(conf.Redis)
    lockMgr      := redis_lock.NewLockManager(redisClient)

    // ── NEW: KGS repos ──
    writeRepo    := postgres.NewGraphWriteRepo(db)
    entityReader := postgres.NewEntityReader(db, neo4jDriver)
    ontologyRepo := postgres.NewOntologyRepo(db)
    outboxRepo   := postgres.NewOutboxRepo(db)
    natsPublisher:= nats.NewPublisher(natsConn)

    // ── NEW: KGS usecases ──
    opaClient    := kgs.NewOPAClient(conf.OPA.ServerURL, redisClient)
    validator    := kgs.NewOntologyValidator(ontologyRepo, redisClient)
    planner      := kgs.NewQueryPlanner()
    graphUC      := kgs.NewGraphUsecase(writeRepo, entityReader, planner, opaClient, validator, redisClient, lockMgr, natsPublisher, logger)
    ontologyUC   := kgs.NewOntologyUsecase(ontologyRepo, redisClient, neo4jDriver, logger)
    overlayUC    := kgs.NewOverlayUsecase(redisClient, graphUC, natsPublisher, logger)

    // ── NEW: KGS handler ──
    kgsH := &kgsGraphHandler{graph: graphUC, ontology: ontologyUC, overlay: overlayUC}
    h.kgs = kgsH

    // ── NEW: Background workers ──
    outboxWorker := outbox.NewWorker(outboxRepo, neo4jDriver, qdrantClient, embeddingProvider, logger)
    go outboxWorker.Start(ctx)

    ontologySync := kgs.NewNeo4jOntologySync(neo4jDriver, ontologyRepo, logger)
    go ontologySync.Start(ctx)

    // ── Existing server setup (UNCHANGED) ──
    router := forward.NewRouter()
    RegisterRoutes(router, h)  // đã extend với KGS routes
    srv.ListenAndServe()
}
```

---

## 6. Database Migrations

```sql
-- migrations/kgs/001_entities.sql
CREATE SCHEMA IF NOT EXISTS kgs;

CREATE TABLE IF NOT EXISTS kgs.kg_entities (
    entity_id       TEXT PRIMARY KEY,
    app_id          VARCHAR(64) NOT NULL,
    tenant_id       VARCHAR(64) NOT NULL,
    entity_type     VARCHAR(128) NOT NULL,
    name            TEXT NOT NULL,
    properties      JSONB,
    confidence      FLOAT DEFAULT 1.0,
    source_file     TEXT,
    chunk_id        VARCHAR(128),
    version         INT DEFAULT 1,
    is_deleted      BOOLEAN DEFAULT FALSE,
    search_vector   tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(name,'') || ' ' || coalesce(properties::text,''))
    ) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_kgs_entity_app ON kgs.kg_entities(app_id, tenant_id);
CREATE INDEX idx_kgs_entity_type ON kgs.kg_entities(entity_type);
CREATE INDEX idx_kgs_entity_search ON kgs.kg_entities USING GIN(search_vector);

-- migrations/kgs/002_edges.sql
CREATE TABLE IF NOT EXISTS kgs.kg_edges (
    edge_id         TEXT PRIMARY KEY,
    app_id          VARCHAR(64) NOT NULL,
    tenant_id       VARCHAR(64) NOT NULL,
    from_entity_id  TEXT NOT NULL REFERENCES kgs.kg_entities(entity_id),
    to_entity_id    TEXT NOT NULL REFERENCES kgs.kg_entities(entity_id),
    relation_type   VARCHAR(128) NOT NULL,
    properties      JSONB,
    confidence      FLOAT DEFAULT 1.0,
    is_deleted      BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- migrations/kgs/003_outbox.sql
CREATE TABLE IF NOT EXISTS kgs.kg_sync_outbox (
    id          BIGSERIAL PRIMARY KEY,
    op          VARCHAR(32) NOT NULL,
    entity_id   TEXT,
    edge_id     TEXT,
    tenant_id   VARCHAR(64) NOT NULL,
    app_id      VARCHAR(64) NOT NULL,
    payload     JSONB NOT NULL,
    status      VARCHAR(16) DEFAULT 'PENDING',
    attempts    INT DEFAULT 0,
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced_at   TIMESTAMPTZ
);
CREATE INDEX idx_kgs_outbox_status ON kgs.kg_sync_outbox(status, created_at);

-- migrations/kgs/004_ontology.sql
CREATE TABLE IF NOT EXISTS kgs.entity_types (
    id              SERIAL PRIMARY KEY,
    app_id          VARCHAR(50) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    display_name    VARCHAR(200),
    description     TEXT,
    id_property     VARCHAR(100),
    schema_json     JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, name)
);

CREATE TABLE IF NOT EXISTS kgs.relation_types (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    source_types JSONB NOT NULL DEFAULT '[]',
    target_types JSONB NOT NULL DEFAULT '[]',
    cardinality  VARCHAR(20) DEFAULT 'MANY_TO_MANY',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, name)
);
```

---

## 7. Configuration Mới

```yaml
# kg-service config — thêm KGS section
kg_service:
  port: 8080

  # Existing config (unchanged)
  postgres:
    dsn: "postgres://..."

  # NEW: KGS infrastructure
  neo4j:
    uri: bolt://neo4j:7687
    username: neo4j
    password: secret

  qdrant:
    addr: qdrant:6334
    collection_prefix: "kgs_"

  redis:
    addr: redis:6379
    lock_ttl: 30s

  opa:
    server_url: http://opa-server:8181
    enabled: false  # false = skip OPA check, true = enforce

  outbox:
    enabled: true
    poll_interval: 500ms
    batch_size: 100
    max_retry: 3

  # Feature flags
  features:
    kgs_graph_enabled: true
    kgs_ontology_enabled: true
    kgs_overlay_enabled: true
    kgs_outbox_enabled: true
```

---

## 8. Effort Breakdown

| Task | Ngày |
|------|------|
| Domain models + KGS infra layer (postgres, neo4j, qdrant, redis) | 2 |
| Outbox worker | 0.5 |
| KGS graph usecase (copy + adapt từ biz/graph.go) | 1 |
| KGS ontology usecase | 0.5 |
| KGS overlay usecase | 0.5 |
| HTTP handlers (kgs_graph.go, kgs_ontology.go, kgs_overlay.go) | 1.5 |
| main.go wiring + background workers | 0.5 |
| Migrations | 0.5 |
| Unit + integration tests | 1 |
| **Total** | **8 ngày** |
