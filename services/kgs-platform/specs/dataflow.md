# Data Flow: API Request → Storage

> **Service:** KGS Platform (`services/kgs-platform`)  
> **Version:** 1.0.0 | **Date:** 2026-04-08

---

## 1. Architectural Layers

Every request — whether gRPC or HTTP — traverses the same 5-layer stack:

```
┌────────────────────────────────────────────────────────────┐
│  LAYER 0  │  Transport                                       │
│           │  HTTP Server (:8000) / gRPC Server (:9000)      │
├────────────────────────────────────────────────────────────┤
│  LAYER 1  │  Middleware Chain                                │
│           │  recovery → Auth() → RateLimiter()              │
├────────────────────────────────────────────────────────────┤
│  LAYER 2  │  Service Layer  (internal/service/*)            │
│           │  Proto ↔ Domain type mapping                    │
├────────────────────────────────────────────────────────────┤
│  LAYER 3  │  Business Logic Layer  (internal/biz/*)         │
│           │  Validation · Policy enforcement · Events       │
├────────────────────────────────────────────────────────────┤
│  LAYER 4  │  Data Layer  (internal/data/*)                  │
│           │  Neo4j · PostgreSQL · Redis · Qdrant · OPA      │
└────────────────────────────────────────────────────────────┘
```

---

## 2. Middleware Chain (All Requests)

The middleware chain is identical for both HTTP and gRPC transports, registered in `internal/server/http.go` and `internal/server/grpc.go`.

```
Incoming Request
      │
      ▼
┌─────────────────────────────────────┐
│  recovery.Recovery()                │  ← Kratos built-in panic recovery
│  Catches panics, returns 500        │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  middleware.Auth()                  │  ← internal/server/middleware/auth.go
│                                     │
│  1. Extract API key from header:    │
│     Authorization: <key>            │
│     X-API-Key: <key>                │
│  2. If missing → reject (401)       │
│  3. TODO: validate hash vs DB/Redis │
│  4. Inject AppContext into ctx:     │
│     { AppID: "...", Scopes: "..." } │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  middleware.RateLimiter()           │  ← internal/server/middleware/ratelimit.go
│                                     │
│  TODO: Read AppID from ctx          │
│  TODO: Redis sliding window check   │
│  (currently pass-through)           │
└──────────────┬──────────────────────┘
               │
               ▼
         Route to Service
```

---

## 3. Flow A: `CreateNode` — Graph Write Operation

**Endpoint:** `POST /v1/graph/nodes` (HTTP) or `Graph.CreateNode` (gRPC)

This is the most complex flow as it triggers a 4-stage pipeline: policy check → Neo4j write → Redis event → downstream reactive rules.

### 3.1 Full Sequence Diagram

```
Client
  │
  │  POST /v1/graph/nodes
  │  { "label": "Feature", "properties_json": "{\"id\":\"f1\",\"name\":\"Login\"}" }
  │
  ▼
Transport Layer (HTTP :8000)
  │  Decode JSON body → CreateNodeRequest proto
  │
  ▼
Middleware Chain
  │  Auth() → extract X-API-Key, inject AppContext{AppID}
  │  RateLimiter() → pass-through
  │
  ▼
service.GraphService.CreateNode()       ← internal/service/graph.go
  │
  │  1. Extract appID (currently hardcoded "system")
  │  2. json.Unmarshal(req.PropertiesJson) → map[string]any{props}
  │  3. Call: uc.CreateNode(ctx, appID, req.Label, props)
  │
  ▼
biz.GraphUsecase.CreateNode()           ← internal/biz/graph.go
  │
  │  ┌─── STEP 1: OPA Policy Check ──────────────────────────┐
  │  │                                                         │
  │  │  opa.EvaluatePolicy(ctx, appID, "CREATE_NODE", label)  │
  │  │          │                                              │
  │  │          ▼                                              │
  │  │  OPAClient.EvaluatePolicy()    ← internal/biz/opa_client.go
  │  │    POST http://opa:8181/v1/data/kgs/allow               │
  │  │    Body: {"input": {"app_id":"..","action":"CREATE_NODE","resource":"Feature"}}
  │  │          │                                              │
  │  │          ▼                                              │
  │  │    OPA evaluates kgs.rego:                              │
  │  │      default allow := false                             │
  │  │      allow if { input.app_id == "demo-app" }            │
  │  │          │                                              │
  │  │          ▼                                              │
  │  │    Response: {"result": true/false}                     │
  │  │                                                         │
  │  │  If false → return error("access denied by OPA policy") │
  │  │  If OPA unreachable → fail closed (return error)        │
  │  └─────────────────────────────────────────────────────────┘
  │
  │  ┌─── STEP 2: Neo4j Persistence ─────────────────────────┐
  │  │                                                         │
  │  │  repo.CreateNode(ctx, appID, label, properties)         │
  │  │          │                                              │
  │  │          ▼                                              │
  │  │  graphRepo.CreateNode()        ← internal/data/graph_node.go
  │  │    session = neo4j.NewSession(AccessModeWrite)          │
  │  │    session.ExecuteWrite(tx → )                          │
  │  │      Cypher:                                            │
  │  │        CREATE (n:Feature {app_id: $app_id})             │
  │  │        SET n += $props                                  │
  │  │        RETURN n                                         │
  │  │      Params: { app_id: appID, props: properties }       │
  │  │                                                         │
  │  │    Returns: node.Props (map[string]any)                │
  │  └─────────────────────────────────────────────────────────┘
  │
  │  ┌─── STEP 3: Domain Event (Redis Stream) ───────────────┐
  │  │                                                         │
  │  │  redisCli.XAdd(ctx, &redis.XAddArgs{                    │
  │  │    Stream: "kgs:events:nodes",                          │
  │  │    Values: {                                            │
  │  │      "event_type": "node.created",                      │
  │  │      "app_id":     appID,                               │
  │  │      "label":      "Feature",                           │
  │  │    },                                                   │
  │  │  })                                                     │
  │  │                                                         │
  │  │  Fire-and-forget: event published asynchronously        │
  │  └─────────────────────────────────────────────────────────┘
  │
  ▼
service.GraphService.CreateNode() (cont.)
  │
  │  4. Safely extract node "id" from result map
  │  5. json.Marshal(result) → properties_json string
  │  6. Return CreateNodeReply { node_id, label, properties_json }
  │
  ▼
Transport Layer → HTTP 200 / gRPC OK → Client

─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ (async) ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─

EventRunner (background goroutine)         ← internal/biz/event_runner.go
  │
  │  XREADGROUP "kgs:events:nodes" → messages
  │  For each message where event_type = "node.created":
  │    1. Extract app_id from message
  │    2. rulesRepo.ListRules(ctx, app_id)
  │       → SELECT * FROM rules WHERE app_id=? AND trigger_type='ON_WRITE' AND is_active=true
  │    3. For each active ON_WRITE rule:
  │       graphRepo.ExecuteQuery(ctx, rule.CypherQuery, { app_id, event: message.Values })
  │    4. XACK "kgs:events:nodes" group message.ID
```

### 3.2 Storage Writes Summary (CreateNode)

| Step | Storage | Operation | Data |
|------|---------|-----------|------|
| 1 | **OPA** (HTTP) | `POST /v1/data/kgs/allow` | `{app_id, action, resource}` → `{result: bool}` |
| 2 | **Neo4j** (Bolt) | `ExecuteWrite` → `CREATE (n:Label)` | Node with `app_id` + all properties |
| 3 | **Redis** (Stream) | `XADD kgs:events:nodes` | `{event_type, app_id, label}` |
| 4 (async) | **Neo4j** (Bolt) | `ExecuteRead` per ON_WRITE rule | Rule Cypher query result |

---

## 4. Flow B: `UpdateNode` / `DeleteNode` — Graph Write Variants

These flows are structurally identical to `CreateNode` with variations in the Cypher and event payload.

### UpdateNode

```
GraphUsecase.UpdateNode()
  │
  ├─── OPA check: action="UPDATE_NODE", resource="*"
  │
  ├─── Neo4j ExecuteWrite:
  │      MATCH (n {app_id: $app_id, id: $node_id})
  │      SET n += $props
  │      RETURN n
  │
  └─── Redis XADD: event_type="node.updated", app_id, node_id
```

### DeleteNode

```
GraphUsecase.DeleteNode()
  │
  ├─── OPA check: action="DELETE_NODE", resource="*"
  │
  ├─── Neo4j ExecuteWrite:
  │      MATCH (n {app_id: $app_id, id: $node_id})
  │      DETACH DELETE n          ← removes node + all connected edges
  │
  └─── Redis XADD: event_type="node.deleted", app_id, node_id
```

### CreateEdge

```
GraphUsecase.CreateEdge()
  │
  ├─── TODO: validate relation whitelist
  │
  └─── Neo4j ExecuteWrite:
         MATCH (a {app_id: $app_id, id: $source_node_id})
         MATCH (b {app_id: $app_id, id: $target_node_id})
         CREATE (a)-[rel:RELATION_TYPE {app_id: $app_id}]->(b)
         SET rel += $props
         RETURN rel
```

> **Note:** `CreateEdge` currently does **not** emit a Redis event and does **not** perform an OPA check (marked as TODO).

---

## 5. Flow C: Graph Read Operations (Context / Impact / Coverage / Subgraph)

Read operations have no OPA check and no Redis write. They use `AccessModeRead` sessions in Neo4j.

### General Pattern

```
Client
  │
  ├── GET /v1/graph/nodes/{node_id}/context?depth=2&direction=OUTGOING
  │
  ▼
service.GraphService.GetContext()
  │
  │  appID = "demo-app"  (hardcoded — TODO: extract from Auth ctx)
  │
  ▼
biz.GraphUsecase.GetContext(ctx, appID, nodeID, depth, direction)
  │
  ├─── STEP 1: Guardrail Validation
  │      ValidateDepth(depth) → error if depth > 10
  │
  ├─── STEP 2: QueryPlanner
  │      QueryPlanner.BuildContextQuery("", direction)
  │      → safe Cypher string (direction interpolated, values parameterized):
  │
  │        MATCH (n {app_id: $app_id, id: $node_id})-[r]->(m {app_id: $app_id})
  │        RETURN n, r, m
  │
  ├─── STEP 3: Neo4j Read
  │      graphRepo.ExecuteQuery(ctx, cypher, { app_id, node_id })
  │          │
  │          ▼
  │      graphRepo.ExecuteQuery()    ← internal/data/graph_query.go
  │        session = neo4j.NewSession(AccessModeRead)
  │        session.ExecuteRead(tx → )
  │          tx.Run(ctx, cypher, params)
  │          collect all records → []map[string]any
  │          return { "data": rows }
  │
  └─── Return map[string]any → service maps to GraphReply proto
```

### QueryPlanner Cypher Patterns

| Method | Cypher Generated | Params Used |
|--------|-----------------|-------------|
| `BuildContextQuery(dir)` | `MATCH (n {app_id:$app_id, id:$node_id})-[r]-(m)` (direction-aware) | `$app_id`, `$node_id` |
| `BuildImpactQuery(depth)` | `MATCH p=(n {app_id:$app_id, id:$node_id})-[*1..N]->(m)` | `$app_id`, `$node_id` |
| `BuildCoverageQuery(depth)` | `MATCH p=(n {app_id:$app_id, id:$node_id})<-[*1..N]-(m)` | `$app_id`, `$node_id` |
| `BuildSubgraphQuery()` | `MATCH (n {app_id:$app_id})-[r]->(m) WHERE n.id IN $node_ids AND m.id IN $node_ids` | `$app_id`, `$node_ids` |

### Guardrail Enforcement

```go
// internal/biz/graph_guardrails.go
const MaxAllowedDepth = 10
const MaxAllowedNodes = 1000

func ValidateDepth(d int) error { ... }      // used by GetContext, GetImpact, GetCoverage
func ValidateNodeCount(n int) error { ... }  // used by GetSubgraph
```

---

## 6. Flow D: Rules Management (`CreateRule`)

**Endpoint:** `POST /v1/rules`

```
Client
  │
  │  POST /v1/rules
  │  { "name": "FlagHighRisk", "trigger_type": "ON_WRITE",
  │    "cypher_query": "MATCH (n {app_id:$app_id}) WHERE ... RETURN n",
  │    "action": "webhook" }
  │
  ▼
Middleware: Auth() → RateLimiter()
  │
  ▼
service.RulesService.CreateRule()        ← internal/service/rules.go
  │
  │  1. appID = "demo-app" (hardcoded)
  │  2. Map proto fields → biz.Rule struct:
  │       { AppID, Name, Description, TriggerType,
  │         Cron, CypherQuery, Action }
  │  3. Call: uc.CreateRule(ctx, rule)
  │
  ▼
biz.RulesUsecase.CreateRule()            ← internal/biz/rules.go
  │
  │  TODO: validation specific to rules
  │
  └─── repo.CreateRule(ctx, rule)
  │
  ▼
data.rulesRepo.CreateRule()              ← internal/data/rules.go
  │
  │  db.WithContext(ctx).Create(rule)
  │  → SQL: INSERT INTO rules (app_id, name, ...) VALUES (...)
  │         Returns auto-incremented ID
  │
  ▼
Return biz.Rule (with ID populated) → service → proto RuleReply → Client
```

### Storage Write (CreateRule)

| Storage | Operation | SQL |
|---------|-----------|-----|
| **PostgreSQL** | `INSERT INTO rules` | `app_id, name, description, trigger_type, cron, cypher_query, action, payload, is_active` |

---

## 7. Flow E: Policy Management (`CreatePolicy`)

**Endpoint:** `POST /v1/policies`

```
Client
  │
  │  POST /v1/policies
  │  { "name": "allow-finpay", "rego_content": "package kgs\nallow if {...}" }
  │
  ▼
service.PolicyService.CreatePolicy()     ← internal/service/policy.go
  │
  │  appID = "demo-app"
  │  Map → biz.Policy { AppID, Name, Description, RegoContent }
  │
  ▼
biz.PolicyUsecase.CreatePolicy()         ← internal/biz/policy.go
  │
  │  TODO: rego syntax validation
  │
  └─── repo.CreatePolicy(ctx, policy)
  │
  ▼
data.policyRepo.CreatePolicy()           ← internal/data/policy.go
  │
  │  db.WithContext(ctx).Create(policy)
  │  → INSERT INTO policies (app_id, name, rego_content, is_active, ...) VALUES (...)
  │
  ▼
Policy is now persisted. PolicySyncRunner will push it to OPA within 30 seconds.

─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ (async, every 30s) ─ ─ ─ ─ ─ ─ ─ ─ ─

PolicySyncRunner                         ← internal/biz/policy_sync.go
  │
  │  ticker fires every 30 seconds
  │
  ├─── repo.ListPolicies(ctx, "demo-app")
  │      → SELECT * FROM policies WHERE app_id='demo-app' AND deleted_at IS NULL
  │
  └─── For each active policy:
         opa.PutPolicy(ctx, policyID, policy.RegoContent)
           PUT http://opa:8181/v1/policies/policy_{id}
           Body: <raw Rego text>
           → OPA loads policy into in-memory engine
```

### Storage Writes (Policy lifecycle)

| Step | Storage | Operation |
|------|---------|-----------|
| 1 | **PostgreSQL** | `INSERT INTO policies` |
| 2 (async) | **OPA** (HTTP) | `PUT /v1/policies/policy_{id}` (Rego text) |

---

## 8. Flow F: Ontology Management (`CreateEntityType`)

**Endpoint:** `POST /v1/ontology/entities`

```
Client
  │
  │  POST /v1/ontology/entities
  │  { "name": "Payment", "description": "...", "schema": "{...}" }
  │
  ▼
service.OntologyService.CreateEntityType()    ← internal/service/ontology.go
  │
  │  Currently: STUB — returns empty reply
  │  TODO: delegate to OntologyUsecase → ontologyRepo → Postgres
  │
  ▼
Returns: CreateEntityTypeReply{}
```

> **Implementation Status:** Ontology write APIs are currently stubs. Reads are served from PostgreSQL via the Ontology data layer with Redis caching (see Flow G).

---

## 9. Flow G: Ontology Cache Read

When the system needs to look up an entity type (during validation or node creation):

```
OntologyRepo.GetEntityType(ctx, appID, name)   ← internal/data/ontology.go
  │
  ├─── STEP 1: Redis Cache Lookup
  │      cacheKey = "ontology:entity:{appID}:{name}"
  │      redisCli.Get(ctx, cacheKey)
  │        │
  │        ├── HIT  → json.Unmarshal → return &EntityType  ✓ fast path
  │        └── MISS → continue to Step 2
  │
  ├─── STEP 2: PostgreSQL Query
  │      db.Where("app_id=? AND name=?", appID, name).First(&entity)
  │      → SELECT * FROM entity_types WHERE app_id=? AND name=? LIMIT 1
  │
  └─── STEP 3: Write-through Cache
         json.Marshal(entity)
         redisCli.Set(ctx, cacheKey, data, 5*time.Minute)
```

**Cache key patterns:**
- Entity types: `ontology:entity:{appID}:{name}` (5-min TTL)
- Relation types: `ontology:relation:{appID}:{name}` (5-min TTL)

---

## 10. Flow H: App Registration (`CreateApp`)

**Endpoint:** `POST /v1/apps`

```
Client
  │
  │  POST /v1/apps
  │  { "app_name": "finpay", "description": "...", "owner": "team-finpay" }
  │
  ▼
service.RegistryService.CreateApp()         ← internal/service/registry.go
  │                                            (Currently STUB)
  │  TODO: delegate to RegistryUsecase
  │
  ───── (intended flow below) ─────
  │
  ▼
data.registryRepo.CreateApp()               ← internal/data/registry.go
  │
  ├─── ATOMIC STEP 1: PostgreSQL
  │      db.WithContext(ctx).Create(app)
  │      → INSERT INTO apps (app_id, app_name, description, owner, status) VALUES (...)
  │
  └─── ATOMIC STEP 2: Neo4j Namespace Reservation
         session.ExecuteWrite(tx →
           MERGE (n:__KGS_Namespace {app_id: $app_id})
           ON CREATE SET n.created_at = datetime()
           RETURN n
         )
         → Reserves the namespace node for tenant isolation
```

### Storage Writes (CreateApp)

| Step | Storage | Operation | Purpose |
|------|---------|-----------|---------|
| 1 | **PostgreSQL** | `INSERT INTO apps` | Tenant registration |
| 2 | **Neo4j** | `MERGE (:__KGS_Namespace)` | Reserve graph namespace |

---

## 11. Flow I: Kafka Document Ingestion (Inbound Event)

This flow is the reverse direction — data arrives via Kafka and flows **into** the graph store.

```
External System (e.g., ai-orchestrator)
  │
  │  Kafka Produce → topic "document.ingested"
  │  {
  │    "docId": "prd-001", "appId": "finpay",
  │    "docType": "PRD", "nodeType": "Feature",
  │    "properties": { "id": "f1", "name": "Login", "priority": "P0" },
  │    "parentId": "uc-001", "edgeType": "REFINES"
  │  }
  │
  ▼
kafka.Consumer (background goroutine)        ← internal/kafka/consumer.go
  │
  │  reader.ReadMessage(ctx)
  │  json.Unmarshal → DocumentIngestedEvent
  │
  ▼
Consumer.handle(ctx, event)
  │
  ├─── STEP 1: Create Node
  │      graph.CreateNode(ctx, event.AppID, event.NodeType, event.Properties)
  │        │
  │        ▼  (same as Flow A but without direct middleware)
  │      biz.GraphUsecase.CreateNode()
  │        → OPA check
  │        → Neo4j: CREATE (n:Feature {app_id:"finpay"}) SET n += $props
  │        → Redis: XADD kgs:events:nodes "node.created"
  │
  └─── STEP 2: Create Edge (if parentId + edgeType present)
         nodeID = result["id"] or fallback to event.DocID
         graph.CreateEdge(ctx, event.AppID, "REFINES", "uc-001", nodeID, nil)
           │
           ▼
         Neo4j: MATCH (a {app_id:$app_id, id:"uc-001"})
                MATCH (b {app_id:$app_id, id:"f1"})
                CREATE (a)-[rel:REFINES {app_id:"finpay"}]->(b)
```

---

## 12. Flow J: Scheduled Rule Execution (Background)

```
RuleRunner                                   ← internal/biz/rule_runner.go
  │
  │  On Start():
  │    rulesRepo.ListRules(ctx, "demo-app")
  │    → SELECT * FROM rules WHERE app_id='demo-app'
  │
  │  For each SCHEDULED, active rule with Cron != "":
  │    gocron.NewJob(CronJob(rule.Cron))
  │      → schedule task: executeRule(rule)
  │
  ─ ─ ─ ─ ─ (at each cron trigger) ─ ─ ─ ─ ─

  executeRule(rule)
    │
    └─── graphRepo.ExecuteQuery(ctx, rule.CypherQuery, { app_id: rule.AppID })
           │
           ▼
         Neo4j ExecuteRead:
           Run arbitrary Cypher (e.g., fraud detection scan)
           Collect result rows
           │
           ▼
         rule.Action dispatch:
           TODO: webhook / push notification / alert
           Currently: log result only
```

---

## 13. Complete Storage Write Matrix

Summary of which operations touch which stores:

| API / Flow | Neo4j | PostgreSQL | Redis (Stream) | Redis (Cache) | OPA |
|------------|:-----:|:----------:|:--------------:|:-------------:|:---:|
| **CreateNode** | ✅ WRITE | — | ✅ XADD | — | ✅ POST |
| **UpdateNode** | ✅ WRITE | — | ✅ XADD | — | ✅ POST |
| **DeleteNode** | ✅ WRITE | — | ✅ XADD | — | ✅ POST |
| **CreateEdge** | ✅ WRITE | — | — | — | — |
| **GetContext** | ✅ READ | — | — | — | — |
| **GetImpact** | ✅ READ | — | — | — | — |
| **GetCoverage** | ✅ READ | — | — | — | — |
| **GetSubgraph** | ✅ READ | — | — | — | — |
| **CreateRule** | — | ✅ INSERT | — | — | — |
| **GetRule / ListRules** | — | ✅ SELECT | — | — | — |
| **CreatePolicy** | — | ✅ INSERT | — | — | — |
| **GetPolicy / ListPolicies** | — | ✅ SELECT | — | — | — |
| **CreateApp** (planned) | ✅ MERGE | ✅ INSERT | — | — | — |
| **GetEntityType** | — | ✅ SELECT | — | ✅ GET/SET | — |
| **GetRelationType** | — | ✅ SELECT | — | ✅ GET/SET | — |
| **Kafka Ingest** | ✅ WRITE | — | ✅ XADD | — | ✅ POST |
| **ON_WRITE Rule (async)** | ✅ READ | ✅ SELECT | — | — | — |
| **SCHEDULED Rule (async)** | ✅ READ | ✅ SELECT | — | — | — |
| **PolicySync (async)** | — | ✅ SELECT | — | — | ✅ PUT |
| **App Startup Seed** | — | ✅ INSERT | — | — | — |

---

## 14. Data Transformation Points

Summarizes how data changes format as it passes through the layers.

```
INBOUND (Client → Storage)
─────────────────────────────────────────────────────────────────

[1] Client sends JSON body or gRPC binary
       ↓
[2] Kratos Transport decodes → proto message object
       e.g. CreateNodeRequest { label: "Feature", properties_json: "{...}" }
       ↓
[3] Service Layer maps proto → domain types
       json.Unmarshal(properties_json) → map[string]any
       appID extracted (currently hardcoded)
       ↓
[4] Biz Layer validates & enriches
       OPA policy check (adds authorization signal)
       RedisStream event assembled
       ↓
[5] Data Layer (graphRepo) parameterizes Cypher
       props map → $props parameter  (Neo4j driver serializes)
       appID   → $app_id parameter
       ↓
[6] Neo4j stores node with label + {app_id} + all properties as node props

OUTBOUND (Storage → Client)
─────────────────────────────────────────────────────────────────

[6] Neo4j returns Record → Values[0].(neo4j.Node).Props  → map[string]any
       ↓
[5] Data Layer returns map[string]any
       ↓
[4] Biz Layer returns map[string]any (no further transform)
       ↓
[3] Service Layer serializes:
       json.Marshal(result) → properties_json string
       extract node_id from result["id"]
       pack into proto reply: CreateNodeReply { node_id, label, properties_json }
       ↓
[2] Kratos Transport encodes → JSON body or gRPC binary
       ↓
[1] Client receives 200 OK with JSON or gRPC response
```

---

## 15. Error Propagation Paths

```
Storage error → Data Layer → Biz Layer → Service Layer → Transport → Client

Neo4j error:
  tx.Run() fails → data layer returns (nil, err)
  → biz.CreateNode returns (nil, err)
  → service returns (nil, err) to Kratos
  → Kratos encodes as gRPC Status / HTTP 5xx

OPA unreachable (fail-closed):
  http.Client.Do() fails → OPAClient returns (false, err)
  → biz.CreateNode returns (nil, err) "access denied"
  → service propagates → 403 Forbidden

OPA returns false:
  opaResp.Result == false
  → biz returns errors.New("access denied by OPA policy")
  → service → 403 Forbidden

Depth guardrail exceeded:
  ValidateDepth(11) → ErrDepthExceeded
  → biz.GetContext returns (nil, err)
  → service → 400 Bad Request

Redis XAdd failure:
  Silent — error is not returned to caller
  Node write has already succeeded; event may be lost
```

---

## 16. Startup Data Flow (Initialization)

When the service starts, three one-time data operations run before serving traffic:

```
main() → wire.Build() → NewData() → NewApp() → app.Run()
              │
              ├─── 1. PostgreSQL AutoMigrate
              │         db.AutoMigrate(App, APIKey, Quota, AuditLog,
              │                        EntityType, RelationType,
              │                        Rule, RuleExecution, Policy)
              │         → CREATE TABLE IF NOT EXISTS ... for all models
              │
              ├─── 2. Ontology Seed
              │         SeedOntology(ctx, db, logger)
              │         For each of 19 node types:
              │           SELECT ... WHERE app_id='system' AND name=?
              │           If not found: INSERT INTO entity_types (...)
              │         For each of 16 edge types:
              │           SELECT ... WHERE app_id='system' AND name=?
              │           If not found: INSERT INTO relation_types (...)
              │
              └─── 3. Worker Server Start
                        PolicySyncRunner.Start() → syncPolicies() immediately
                          SELECT * FROM policies WHERE app_id='demo-app'
                          PUT http://opa:8181/v1/policies/policy_{id} for each
                        RuleRunner.Start()
                          SELECT * FROM rules WHERE app_id='demo-app'
                          gocron.NewJob for each SCHEDULED rule
                        EventRunner.Start()
                          XGROUP CREATE kgs:events:nodes kgs-worker-group $ (MKSTREAM)
                        Kafka Consumer.Start() (if configured)
                          kafka.NewReader(brokers, topic="document.ingested")
```
