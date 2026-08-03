# C4 Level 3 — Component Diagram

> **C4 Level**: 3 — Component  
> **Câu hỏi**: Trong KG API Server có những module nào? Tương tác thế nào?  
> **Audience**: Backend developers, architects

---

## Overview — 3 Planes Architecture

KG API Server được tổ chức thành **3 planes logic** + **cross-cutting concerns**:

```
╔═══════════════════════════════════════════════════════════════════════════════╗
║  { KG API SERVER }                                                            ║
║                                                                               ║
║  ┌─────────────────────────────────────────────────────────────────────────┐  ║
║  │  Cross-Cutting Layer                                                     │  ║
║  │  < HTTP Router >  < Auth Middleware >  < Rate Limiter >  < Audit Log >  │  ║
║  └────────────────────────────┬────────────────────────────────────────────┘  ║
║                               │                                               ║
║  ┌────────────────────┐  ┌───▼────────────────┐  ┌───────────────────────┐   ║
║  │ IDENTITY & ACCESS  │  │  ONTOLOGY PLANE     │  │  DATA PLANE           │   ║
║  │ PLANE              │  │                     │  │                       │   ║
║  │                    │  │  < DomainRegistry > │  │  < WriteService >     │   ║
║  │  < IdentityRes. >  │  │  < NodeTypeReg. >   │  │  < ReadService >      │   ║
║  │  < AccessResolver> │◀─┤  < OntologyRes. >   ├─▶│  < SearchService >    │   ║
║  │  < AccessGrant  >  │  │  < QueryTemplate >  │  │  < MCPServer >        │   ║
║  │  < TenantRegistry> │  │  < StatusFieldCfg > │  │  < IngestService >    │   ║
║  │  < AppRegistry >   │  │  < CrossDomainRule> │  │                       │   ║
║  └────────────────────┘  └────────────────────┘  └───────────────────────┘   ║
║                                    │                                           ║
║                      ┌─────────────▼───────────────┐                          ║
║                      │  SYNC & CONSISTENCY          │                          ║
║                      │                              │                          ║
║                      │  < GraphSyncWorker >         │                          ║
║                      │  < VectorSyncWorker >        │                          ║
║                      │  < AccessSyncWorker >        │                          ║
║                      │  < ReconciliationWorker >    │                          ║
║                      │  < OutboxPoller >            │                          ║
║                      └──────────────────────────────┘                          ║
║                                                                               ║
║  ┌─────────────────────────────────────────────────────────────────────────┐  ║
║  │  Platform Adapters (internal/platform/)                                  │  ║
║  │  < PostgresPool >  < GraphStoreAdapter >  < VectorStoreAdapter >        │  ║
║  │  < FTSAdapter >    < RedisCache >          < EmbeddingClient >           │  ║
║  └─────────────────────────────────────────────────────────────────────────┘  ║
╚═══════════════════════════════════════════════════════════════════════════════╝
```

---

## Plane 1: Identity & Access

```
┌──────────────────────────────────────────────────────────────────────────┐
│  IDENTITY & ACCESS PLANE (internal/identity/, internal/access/)          │
│                                                                          │
│  < IdentityResolver >                                                    │
│    resolve(api_key) → (tenant_id, app_id)                               │
│    SHA256 hash lookup → Redis cache (30s) → PostgreSQL fallback          │
│    Invalidate: app.status = 'revoked'                                    │
│                                                                          │
│  < TenantRegistry >                                                      │
│    CRUD operations on tenants table                                      │
│    Platform sentinel tenant management                                   │
│                                                                          │
│  < AppRegistry >                                                         │
│    CRUD operations on apps table                                         │
│    API key generation (bcrypt hash) + rotation                          │
│                                                                          │
│  < AccessResolver >                                                      │
│    resolve_visible_owners(tenant_id, app_id) → Set[VisibleOwner]        │
│    Algorithm:                                                            │
│      1. self + tenant-wide entries                                       │
│      2. platform ("*:*")                                                │
│      3. default_sharing_policy (if share_within_tenant_read)            │
│      4. active AccessGrants (not expired)                               │
│    Cache: Redis "acl:{t}:{a}" TTL 60s                                   │
│    Invalidate: ACCESS_GRANT_CHANGED event                               │
│                                                                          │
│  < AccessGrantStore >                                                    │
│    CRUD for access_grants table                                          │
│    Grant creation + revocation                                           │
│    Publishes ACCESS_GRANT_CHANGED events to outbox                      │
│                                                                          │
│  < AuditLogger >                                                         │
│    Writes to access_audit_log (partitioned by month)                    │
│    Every API call logged (allow + deny)                                  │
└──────────────────────────────────────────────────────────────────────────┘
```

**Key interactions**:
- Every request → `IdentityResolver` (first middleware)
- Every data access → `AccessResolver` (builds acl_tokens)
- Grant changes → `AccessGrantStore` → `OutboxPoller` → `AccessSyncWorker`

---

## Plane 2: Ontology Plane

```
┌──────────────────────────────────────────────────────────────────────────┐
│  ONTOLOGY PLANE (internal/ontology/, internal/searchprofile/)            │
│                                                                          │
│  < DomainRegistry >                                                      │
│    CRUD for domains table                                                │
│    Domain lifecycle: draft → active → deprecated                        │
│    Ownership enforcement (tenant can only modify own domains)           │
│                                                                          │
│  < NodeTypeRegistry >                                                    │
│    CRUD for node_type_schemas                                           │
│    Schema validation rules storage                                       │
│                                                                          │
│  < RelTypeRegistry >                                                     │
│    CRUD for rel_type_schemas                                            │
│    Cross-domain rel rule storage                                         │
│                                                                          │
│  < OntologyResolver >                                                    │
│    get_effective_ontology(t, a) → List[Domain]                          │
│    = platform + own + granted domains                                   │
│    Cache: Redis "ontology:effective:{t}:{a}" TTL 300s                  │
│                                                                          │
│  < NodeValidator >                                                       │
│    validate(domain_id, node_type, properties) → ValidationResult        │
│    Steps:                                                                │
│      1. Domain in effective ontology?                                   │
│      2. NodeTypeSchema exists?                                           │
│      3. Required props present + type match?                            │
│      4. Validation rules pass?                                           │
│      5. CrossDomainRelRules satisfied?                                  │
│                                                                          │
│  < QueryTemplateRegistry >                                               │
│    CRUD for domain_query_templates (DSL-based, NOT Cypher)             │
│    Template activation (draft → active)                                 │
│    Template discovery (list by domain)                                  │
│                                                                          │
│  < QueryTemplateCompiler >                                               │
│    compile(pattern_spec, domain_id, acl_tokens) → CypherQuery           │
│    ALWAYS injects ACL filter at EVERY graph hop                         │
│    Reads status_field_config to inject status filter                    │
│    Max hop depth = 5 (validated at template registration)               │
│                                                                          │
│  < StatusFieldConfigRegistry >                                           │
│    CRUD for domain_status_field_configs                                 │
│    Used by: StatusGate (filter/warn), VectorSync (map status_value),    │
│             QueryTemplateCompiler (inject filter_status)                │
│                                                                          │
│  < CrossDomainRuleRegistry >                                             │
│    CRUD for cross_domain_rel_rules                                      │
│    Evaluated by NodeValidator at write time                             │
│                                                                          │
│  < SearchProfileRegistry >                                               │
│    CRUD for domain search profiles                                      │
│    Controls: embedding_fields, fts_fields, hybrid weights               │
└──────────────────────────────────────────────────────────────────────────┘
```

**Key interactions**:
- `WriteService` → `NodeValidator` → `NodeTypeRegistry` + `CrossDomainRuleRegistry`
- `ReadService` → `QueryTemplateRegistry` → `QueryTemplateCompiler`
- `SearchService` → `StatusFieldConfigRegistry` (for status filter)
- `VectorSyncWorker` → `StatusFieldConfigRegistry` (map status_value + authority_score)

---

## Plane 3: Data Plane

```
┌──────────────────────────────────────────────────────────────────────────┐
│  DATA PLANE (internal/write/, internal/read/, internal/search/,          │
│              internal/mcp/)                                              │
│                                                                          │
│  < WriteService >                                                        │
│    write_node(t, a, domain_id, node_type, props) → node_id             │
│    write_relationship(...) → rel_id                                      │
│    bulk_write_nodes([...]) → []node_id                                  │
│    delete_node(id) → soft delete                                         │
│    delete_by_external_ref_prefix(prefix)                                │
│    Flow:                                                                 │
│      1. AuthZ: has_write_permission?                                    │
│      2. NodeValidator.validate(...)                                      │
│      3. PostgreSQL transaction:                                          │
│           SET LOCAL app.tenant_id, app.app_id                           │
│           INSERT kg_nodes                                                │
│           INSERT kg_relationships (cross-domain)                        │
│           INSERT kg_outbox_events (NODE_UPSERTED)                       │
│           COMMIT                                                         │
│      4. 202 Accepted                                                     │
│                                                                          │
│  < IngestService >                                                       │
│    ingest_document(file_url, domain_id, metadata) → job_id             │
│    get_ingest_status(job_id) → { status, nodes_created, errors }       │
│    Async document parsing → node creation pipeline                      │
│                                                                          │
│  < SyncSessionService >                                                  │
│    open_sync_session() → session_id                                     │
│    commit_session(session_id)                                           │
│    abandon_session(session_id)                                          │
│    Graph version-aware bulk sync                                        │
│                                                                          │
│  < ReadService >                                                         │
│    execute_template(t, a, domain_id, template_name, params) → records  │
│    Flow:                                                                 │
│      1. Load template from domain_query_templates                       │
│      2. Status must be "active"                                         │
│      3. AccessResolver → acl_tokens                                     │
│      4. QueryTemplateCompiler.compile(DSL, acl_tokens)                  │
│      5. GraphDB.run(cypher, timeout=3000ms)                             │
│      6. AuditLogger.log(...)                                            │
│                                                                          │
│    read_node(id, mode) → node + relationships                           │
│      mode=realtime: check graph version vs PG version                  │
│                     fallback to PG if graph stale                       │
│      mode=non-realtime: read from graph projection                      │
│                                                                          │
│  < SearchService >                                                       │
│    semantic_search(t, a, query, domain_ids, top_k) → results           │
│    rag_search(t, a, query) → answer_context                             │
│    fulltext_search(t, a, query, domain_ids, fields, mode) → results    │
│    hybrid_search(t, a, query, semantic_weight, fts_operator) → results │
│    graph_search(t, a, query_spec) → results                            │
│    Flow (semantic):                                                      │
│      1. AccessResolver → acl_tokens                                     │
│      2. Build filter: acl + is_deleted + domain_ids + status_value     │
│      3. EmbeddingClient.embed(query)                                    │
│      4. VectorDB.search(vector, filter, top_k)                         │
│      5. [Optional] authority rerank                                     │
│      6. AuditLogger.log(...)                                            │
│                                                                          │
│  < MCPServer >                                                           │
│    Implements MCP protocol over HTTP + SSE                              │
│    Session management (connect → session_id)                            │
│    Tool dispatch: tools/list, tools/call                                │
│    9 tools: kg_search, kg_search_rag, kg_read_pattern,                 │
│              kg_list_domains, kg_list_templates, kg_get_node,           │
│              kg_write_node, kg_check_access, kg_integrity               │
│    Delegates to: SearchService, ReadService, WriteService               │
│    Same auth/ACL model as REST (connection-level auth)                  │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Sync & Consistency Layer

```
┌──────────────────────────────────────────────────────────────────────────┐
│  SYNC & CONSISTENCY (internal/workers/)                                  │
│                                                                          │
│  < OutboxPoller >                                                        │
│    Polls kg_outbox_events WHERE status = 'PENDING'                      │
│    Batch size configurable                                               │
│    Updates status: PENDING → PROCESSING → DONE/FAILED                  │
│    Dead-letter after 5 retries → status = DEAD_LETTER                  │
│                                                                          │
│  < GraphSyncWorker >                                                     │
│    Handles: NODE_UPSERTED, NODE_DELETED                                 │
│      → compute_acl_visible_to(node)                                     │
│      → GraphStore.merge_node(node, acl_visible_to, _kg_sync_version)   │
│    Handles: STATUS_VALUE_CHANGED                                        │
│      → Read cascade_rules from domain_status_field_configs              │
│      → Execute cascade Cypher (generic, no hardcoded label names)      │
│    Handles: REL_UPSERTED, REL_DELETED                                  │
│      → GraphStore.merge_relationship(rel, _kg_sync_version)             │
│                                                                          │
│  < VectorSyncWorker >                                                    │
│    Handles: NODE_UPSERTED                                               │
│      → Fetch full node from PG                                          │
│      → Build embedding text from search profile fields                 │
│      → EmbeddingClient.embed(text) → vector                            │
│      → Map status_value from StatusFieldConfig                         │
│      → Map authority_score from StatusFieldConfig                      │
│      → VectorStore.upsert(node_id, vector, payload)                    │
│    Handles: NODE_DELETED                                                │
│      → VectorStore.delete(node_id)                                      │
│                                                                          │
│  < AccessSyncWorker >                                                    │
│    Handles: ACCESS_GRANT_CHANGED (grant created or revoked)            │
│      → Find all nodes owned by grantor in grant scope                  │
│      → Recompute acl_visible_to for each node                          │
│      → Bulk update: GraphStore.update_acl(nodes)                       │
│      → Bulk update: VectorStore.update_payload_acl(nodes)              │
│      → Redis: DEL "acl:{grantee}:*" (immediate)                        │
│                                                                          │
│  < ReconciliationWorker >                                               │
│    Schedule: hourly                                                     │
│    Compares: kg_nodes count vs Graph node count vs Vector doc count     │
│    Compares: _kg_sync_version across 3 stores                          │
│    Issues detected:                                                     │
│      graph_mismatch, vector_mismatch, orphan_graph_node,               │
│      orphan_vector_doc, stale_projection_version                       │
│    Reports via: GET /v1/kg/integrity/tenant/{t}                        │
│    Repair via:  POST /v1/kg/integrity/repair/rebuild                   │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Platform Adapters Layer

```
┌──────────────────────────────────────────────────────────────────────────┐
│  PLATFORM ADAPTERS (internal/platform/)                                  │
│                                                                          │
│  < GraphStoreAdapter >  (internal/platform/graphstore/)                 │
│    Interface: merge_node, merge_rel, query, delete, update_property     │
│    Implementations:                                                     │
│      - memory.go       ← in-memory (dev/test)                          │
│      - neo4j.go        ← Bolt driver (Neo4j)                           │
│      - surreal.go      ← Memgraph via Bolt                             │
│      - nebula_real.go  ← Nebula nGQL protocol                          │
│    Factory: backends.go (selected by GRAPH_ADAPTER env)                │
│                                                                          │
│  < VectorStoreAdapter > (internal/platform/vectorstore/)               │
│    Interface: upsert, search, delete, update_payload                    │
│    Implementations:                                                     │
│      - memory.go       ← in-memory (dev/test)                          │
│      - pgvector.go     ← pgvector extension via pgx                    │
│      - milvus.go       ← Milvus gRPC SDK                              │
│      - (qdrant via backends.go)                                         │
│    Factory: backends.go (selected by VECTOR_ADAPTER env)               │
│                                                                          │
│  < FTSAdapter >         (internal/platform/fts/)                       │
│    Interface: index, search, delete                                     │
│    Implementations: memory.go, postgres.go (tsvector)                  │
│                                                                          │
│  < PostgresPool >       (internal/platform/postgres/)                  │
│    Connection pool management (pgx/v5)                                 │
│    SET LOCAL context injection (tenant_id, app_id)                     │
│    Transaction helpers                                                  │
│                                                                          │
│  < RedisCache >         (internal/platform/rediscache/)                │
│    Get/Set/Del with TTL                                                 │
│    Rate limit increment + check                                         │
│    Session management (MCP sessions)                                   │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Cross-Cutting Concerns

```
┌──────────────────────────────────────────────────────────────────────────┐
│  CROSS-CUTTING (internal/httpapi/, internal/observability/, etc.)        │
│                                                                          │
│  < HTTP Router >  (httpapi/)                                            │
│    Route registration: REST + MCP                                       │
│    Middleware chain: identity → rate-limit → audit                      │
│    Error envelope: { error: { code, message, details } }               │
│                                                                          │
│  < Auth Middleware >                                                     │
│    Extract Bearer token from Authorization header                       │
│    Call IdentityResolver.resolve()                                      │
│    Strip tenant_id/app_id from JSON body (security invariant P5)       │
│    Inject (tenant_id, app_id) into request context                     │
│                                                                          │
│  < Rate Limiter >                                                        │
│    Redis-backed sliding window counter                                  │
│    Per-tenant, per-time-window                                          │
│    Tier-based limits: free=60rpm, pro=600rpm, enterprise=6000rpm       │
│    X-RateLimit-{Limit, Remaining, Reset} headers                       │
│                                                                          │
│  < Health Check >  (observability/)                                     │
│    GET /healthz → { service, postgres, redis } (public)                │
│    Checks: DB connectivity, cache connectivity                          │
│                                                                          │
│  < Metrics >  (runtimeobs/)                                             │
│    GET /v1/kg/metrics → worker lag, projection queue depth             │
│    OpenTelemetry integration (go.opentelemetry.io/otel)                │
│                                                                          │
│  < Integrity API >  (integrity/)                                        │
│    GET /v1/kg/integrity/tenant/{t}    ← full drift report              │
│    GET /v1/kg/integrity/missing-bridges                                 │
│    GET /v1/kg/integrity/orphans                                         │
│    POST /v1/kg/integrity/repair/rebuild                                 │
│    POST /v1/kg/integrity/repair/purge-orphans                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Component Interaction Map

Key interactions between components (selected critical paths):

```
WRITE PATH:
  HTTP Handler
    → Auth Middleware (IdentityResolver)
    → WriteService
        → AccessResolver (has_write_permission?)
        → NodeValidator
            → OntologyResolver (effective ontology check)
            → NodeTypeRegistry (schema lookup)
            → CrossDomainRuleRegistry (rule check)
        → PostgresPool (INSERT kg_nodes + outbox)
    → 202 Accepted

  [Async]
  OutboxPoller → GraphSyncWorker → GraphStoreAdapter → Graph DB
  OutboxPoller → VectorSyncWorker → EmbeddingClient → VectorStoreAdapter → Vector DB

READ PATH:
  HTTP Handler
    → Auth Middleware (IdentityResolver)
    → ReadService
        → AccessResolver (acl_tokens)
        → QueryTemplateRegistry (load DSL)
        → QueryTemplateCompiler (DSL + acl_tokens → Cypher)
        → GraphStoreAdapter.query(cypher)
        → AuditLogger
    → 200 OK

SEARCH PATH:
  HTTP Handler
    → Auth Middleware (IdentityResolver)
    → SearchService
        → AccessResolver (acl_tokens)
        → StatusFieldConfigRegistry (status filter)
        → EmbeddingClient.embed(query)
        → VectorStoreAdapter.search(vector, filter)
        → [Optional] StatusGate (filter/warn)
        → AuditLogger
    → 200 OK

MCP TOOL CALL:
  MCP Connect (SSE)
    → Auth Middleware (connection-level)
    → MCPServer.session_create()
  MCP Message
    → MCPServer.dispatch(tool_name, args)
        → SearchService / ReadService / WriteService (same as REST)
    → JSON-RPC response
```

---

## Component Responsibility Matrix

| Component | Plane | Package | Primary DB interaction |
|:---|:---:|:---|:---|
| IdentityResolver | Identity | `identity/` | Redis → PostgreSQL |
| AccessResolver | Identity | `access/` | Redis → PostgreSQL |
| AccessGrantStore | Identity | `access/` | PostgreSQL + Redis invalidation |
| AuditLogger | Identity | `access/` | PostgreSQL |
| DomainRegistry | Ontology | `ontology/` | PostgreSQL |
| NodeTypeRegistry | Ontology | `ontology/` | PostgreSQL |
| NodeValidator | Ontology | `ontology/` | PostgreSQL (via registries) |
| QueryTemplateRegistry | Ontology | `ontology/` | PostgreSQL |
| QueryTemplateCompiler | Ontology | `ontology/` | None (pure logic) |
| StatusFieldConfigRegistry | Ontology | `ontology/` | PostgreSQL |
| WriteService | Data | `write/` | PostgreSQL (write) |
| ReadService | Data | `read/` | Graph DB (read) |
| SearchService | Data | `search/` | Vector DB (read) |
| MCPServer | Data | `mcp/` | Delegates to other services |
| GraphSyncWorker | Sync | `workers/` | PostgreSQL (read) + Graph DB (write) |
| VectorSyncWorker | Sync | `workers/` | PostgreSQL (read) + Vector DB (write) |
| AccessSyncWorker | Sync | `workers/` | PostgreSQL + Graph + Vector + Redis |
| ReconciliationWorker | Sync | `workers/` | PostgreSQL + Graph + Vector (read) |
