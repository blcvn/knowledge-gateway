# Technical Design Document: KGS Platform

> **Version:** 1.0.0  
> **Date:** 2026-04-08  
> **Status:** Active  
> **Repository:** `services/kgs-platform`

---

## 1. Overview

**KGS Platform** (Knowledge Graph Service Platform) is a multi-tenant, cloud-native backend service that provides a **managed Knowledge Graph** infrastructure. It acts as the canonical graph data store and policy engine for the VNP Design Platform ecosystem.

The service enables downstream consumers (e.g., `pipeline-shim`, `service-kg-to-preview`) to:

- **Store and traverse** structured knowledge as nodes and relationships in a Neo4j graph database
- **Define ontologies** (entity types and relation types) to enforce schema contracts on graph data
- **Execute business rules** automatically — both on a CRON schedule and reactively on data write events
- **Manage access control policies** via OPA (Open Policy Agent) with Rego
- **Register multi-tenant applications** with API key-based authentication
- **Ingest document events** from Kafka to populate the graph automatically

---

## 2. Technology Stack

| Layer | Technology | Role |
|-------|-----------|------|
| Language | Go 1.24+ | Core implementation |
| Framework | Kratos v2 | Service scaffolding, DI, transport |
| Graph DB | Neo4j 5.x | Primary knowledge graph store |
| Relational DB | PostgreSQL (via GORM) | Metadata, ontology, rules, policies |
| Cache | Redis | Ontology cache, event stream (Redis Streams) |
| Policy Engine | OPA (Open Policy Agent) | Attribute-based access control |
| Vector Store | Qdrant | Semantic similarity search on knowledge chunks |
| Event Bus | Apache Kafka | External document ingestion events |
| API Protocol | gRPC + REST (HTTP/1.1) | Dual transport via `google.api.http` annotations |
| Dependency Injection | Google Wire | Compile-time DI graph |
| Serialization | Protocol Buffers (proto3) | API contracts |
| Job Scheduling | gocron v2 | Cron-based rule execution |
| Container | Docker (Alpine multi-stage) | Deployment artifact |

---

## 3. Architecture

### 3.1 Clean Architecture Layers

The service follows the **Clean Architecture** pattern enforced by the Kratos framework:

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Transport Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │  HTTP Server  │  │  gRPC Server  │  │     Worker Server        │  │
│  │  (port 8000) │  │  (port 9000) │  │  (Background Goroutines) │  │
│  └──────┬───────┘  └──────┬───────┘  └────────────┬─────────────┘  │
│         └──────────────────┴──────────────────────┘                 │
│                          Service Layer                               │
│  ┌───────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐ ┌────────────┐ │
│  │ GraphSvc  │ │OntoSvc   │ │ Registry │ │ Rules │ │   Policy   │ │
│  └─────┬─────┘ └────┬─────┘ └────┬─────┘ └───┬───┘ └─────┬──────┘ │
│         └───────────┴────────────┴────────────┘           │         │
│                          Business Logic Layer                        │
│  ┌──────────────┐ ┌─────────────┐ ┌────────────┐ ┌───────────────┐ │
│  │ GraphUsecase │ │ RulesUsecase│ │OntologySync│ │  OPAClient    │ │
│  │ QueryPlanner │ │  RuleRunner │ │PolicySync  │ │  EventRunner  │ │
│  └──────┬───────┘ └──────┬──────┘ └─────┬──────┘ └───────┬───────┘ │
│          └───────────────┴───────────────┘               │          │
│                          Data Layer                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │
│  │  Neo4j   │ │ Postgres  │ │  Redis   │ │  Qdrant  │ │  OPA     │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Infrastructure Topology

```
┌── External Clients ─────────────────────────────────────────────────┐
│   pipeline-shim, service-kg-to-preview, frontend                     │
└────────────┬───────────────────────────────┬────────────────────────┘
             │ gRPC / REST                   │ Kafka produce
             ▼                               ▼
┌── kgs-platform ─────────────────────────────────────────────────────┐
│  ┌─────────────┐  ┌─────────────┐                                   │
│  │  HTTP :8000 │  │  gRPC :9000 │   ← Auth middleware + rate limit  │
│  └──────┬──────┘  └──────┬──────┘                                   │
│         │                │                                           │
│  ┌──────▼────────────────▼──────┐                                   │
│  │         Service Layer         │                                   │
│  │ Graph | Ontology | Registry   │                                   │
│  │ Rules | Policy | Greeter      │                                   │
│  └──────────────┬───────────────┘                                   │
│                 │                                                    │
│  ┌──────────────▼───────────────┐   ┌──────────────────────────┐   │
│  │       Business Layer          │   │    Worker Server          │   │
│  │ GraphUsecase + QueryPlanner   │   │ RuleRunner (cron)         │   │
│  │ OntologySyncManager           │   │ EventRunner (Redis stream) │   │
│  │ PolicyUsecase + OPAClient     │   │ PolicySyncRunner          │   │
│  │ RulesUsecase + EventRunner    │   │ Kafka Consumer            │   │
│  └──────────────┬───────────────┘   └──────────────────────────┘   │
│                 │                                                    │
│  ┌──────────────▼───────────────────────────────────────────────┐   │
│  │                    Data Layer                                  │   │
│  │  PostgreSQL   Neo4j      Redis       Qdrant      OPA sidecar  │   │
│  │  (metadata)  (graph)    (cache/      (vector)   (policies)    │   │
│  │                          events)                               │   │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. API Surfaces

### 4.1 Dual Transport

All services support both **gRPC** (port 9000) and **REST HTTP** (port 8000) via `google.api.http` annotations in the `.proto` definitions. The Kratos framework auto-generates both transports from a single service implementation.

Both servers apply identical middleware:
- `recovery.Recovery()` — panic recovery
- `middleware.Auth()` — API key authentication
- `middleware.RateLimiter()` — request rate limiting

---

### 4.2 Graph Service — `api/graph/v1/graph.proto`

Manages Knowledge Graph nodes and edges in Neo4j.

| RPC | HTTP | Description |
|-----|------|-------------|
| `CreateNode` | `POST /v1/graph/nodes` | Create a new typed node (label + properties JSON) |
| `GetNode` | `GET /v1/graph/nodes/{node_id}` | Retrieve a node by ID |
| `UpdateNode` | `PUT /v1/graph/nodes/{node_id}` | Batch-update node properties |
| `DeleteNode` | `DELETE /v1/graph/nodes/{node_id}` | Detach-delete node and all edges |
| `CreateEdge` | `POST /v1/graph/edges` | Create a directed typed relationship |
| `GetContext` | `GET /v1/graph/nodes/{node_id}/context` | Neighborhood traversal (configurable direction & depth) |
| `GetImpact` | `GET /v1/graph/nodes/{node_id}/impact` | Downstream impact analysis (up to `max_depth`) |
| `GetCoverage` | `GET /v1/graph/nodes/{node_id}/coverage` | Upstream coverage analysis (up to `max_depth`) |
| `GetSubgraph` | `POST /v1/graph/subgraph` | Induced subgraph for a set of node IDs |

**Direction parameter** for `GetContext`: `INCOMING | OUTGOING | BOTH`  
**Guardrails**: max depth = 10, max nodes per subgraph query = 1000

---

### 4.3 Ontology Service — `api/ontology/v1/ontology.proto`

Manages the schema contract for the Knowledge Graph.

| RPC | HTTP | Description |
|-----|------|-------------|
| `CreateEntityType` | `POST /v1/ontology/entities` | Register a new node label with a JSON Schema |
| `CreateRelationType` | `POST /v1/ontology/relations` | Register a directed edge type with source/target constraints |
| `ListEntityTypes` | `GET /v1/ontology/entities` | List all registered entity schemas |
| `ListRelationTypes` | `GET /v1/ontology/relations` | List all registered relation schemas |

---

### 4.4 Registry Service — `api/registry/v1/registry.proto`

Multi-tenant application registration and API key management.

| RPC | HTTP | Description |
|-----|------|-------------|
| `CreateApp` | `POST /v1/apps` | Register a new tenant application |
| `GetApp` | `GET /v1/apps/{app_id}` | Get app metadata |
| `ListApps` | `GET /v1/apps` | List all registered apps |
| `IssueApiKey` | `POST /v1/apps/{app_id}/keys` | Issue a new API key (scopes, TTL) |
| `RevokeApiKey` | `DELETE /v1/keys/{key_hash}` | Revoke an existing API key |

---

### 4.5 Rules Service — `api/rules/v1/rules.proto`

Manages business rules that execute Cypher queries on the graph.

| RPC | HTTP | Description |
|-----|------|-------------|
| `CreateRule` | `POST /v1/rules` | Create a rule (SCHEDULED or ON_WRITE trigger) |
| `GetRule` | `GET /v1/rules/{id}` | Get a specific rule |
| `ListRules` | `GET /v1/rules` | List all rules for the tenant |

**Rule triggers**:
- `SCHEDULED` — executed by the cron `RuleRunner` using `gocron`
- `ON_WRITE` — executed reactively by the `EventRunner` listening on Redis Streams

---

### 4.6 Access Control Service — `api/accesscontrol/v1/policy.proto`

Manages OPA Rego policies stored in PostgreSQL and synced live to the OPA sidecar.

| RPC | HTTP | Description |
|-----|------|-------------|
| `CreatePolicy` | `POST /v1/policies` | Create a Rego policy document |
| `GetPolicy` | `GET /v1/policies/{id}` | Get a specific policy |
| `ListPolicies` | `GET /v1/policies` | List all policies for the tenant |

---

## 5. Business Logic Layer (`internal/biz`)

### 5.1 GraphUsecase

Central use case for all graph mutations and traversal queries. It composes:

| Component | Role |
|-----------|------|
| `GraphRepo` | Neo4j data persistence interface |
| `QueryPlanner` | Generates safe, namespaced Cypher queries |
| `OPAClient` | Policy enforcement before each write operation |
| `redis.Client` | Publishes domain events to Redis Streams (`kgs:events:nodes`) |

**Write flow (Create/Update/Delete):**
1. OPA policy check (`EvaluatePolicy`) — fails closed if OPA is unreachable
2. Persist to Neo4j via `GraphRepo`
3. Publish event to `kgs:events:nodes` Redis Stream

---

### 5.2 QueryPlanner

Generates safe, namespaced Cypher queries. All queries are scoped to the tenant namespace via the `app_id` property, which is embedded on every node and edge.

> **Design note:** Neo4j does not support parameterized labels or relationship types in Cypher. The `QueryPlanner` safely interpolates these at build time while keeping all value bindings parameterized to prevent injection.

| Method | Query Pattern |
|--------|--------------|
| `BuildContextQuery` | 1-hop neighborhood traversal, configurable direction |
| `BuildImpactQuery` | Downstream directed path: `(n)-[*1..depth]->(m)` |
| `BuildCoverageQuery` | Upstream directed path: `(n)<-[*1..depth]-(m)` |
| `BuildSubgraphQuery` | Induced subgraph within a node set |

---

### 5.3 OPAClient

HTTP client interfacing with the **OPA sidecar container** running on `http://opa:8181`.

- **EvaluatePolicy**: `POST /v1/data/kgs/allow` — evaluates the input `{app_id, action, resource}` against the loaded Rego policies
- **PutPolicy**: `PUT /v1/policies/{id}` — uploads a raw Rego policy string to OPA

**Default Rego policy** (`configs/kgs.rego`): Default-deny; allows `demo-app` for development. Real environments load policies from the database via `PolicySyncRunner`.

---

### 5.4 RuleRunner (Scheduled Rules)

Implements `transport.Server` via `WorkerServer`. On startup, it:
1. Queries all `SCHEDULED` and active rules from PostgreSQL
2. Registers each as a `gocron` job using the rule's `Cron` expression
3. Executes the rule's `CypherQuery` against Neo4j at each trigger interval
4. Dispatches the result to the configured `Action` (webhook, push notification, etc.)

---

### 5.5 EventRunner (Reactive ON_WRITE Rules)

Subscribes to the `kgs:events:nodes` **Redis Stream** using consumer group `kgs-worker-group`:

1. Creates the stream consumer group on start (idempotent)
2. Reads batches of up to 10 messages (blocking)
3. For each event, retrieves all `ON_WRITE` rules for the event's `app_id`
4. Executes each active rule's Cypher query with the event payload as parameters
5. Acknowledges (`XACK`) processed messages

---

### 5.6 PolicySyncRunner

Background runner that polls PostgreSQL every **30 seconds** and pushes all active Rego policies to OPA via `PutPolicy`. This ensures the OPA sidecar's in-memory policy state stays in sync with the database without requiring restart.

---

### 5.7 OntologySyncManager

Responsible for syncing Neo4j index/constraint definitions that reflect the ontology schema stored in PostgreSQL. Currently a stub — production implementation would generate `CREATE CONSTRAINT` Cypher statements from `EntityType.Schema` JSON Schemas.

---

## 6. Data Layer (`internal/data`)

### 6.1 Neo4j — Graph Store

All graph mutations are executed via the official `neo4j-go-driver/v5` in **managed transactions**.

| Operation | Session Mode |
|-----------|-------------|
| `CreateNode` | `AccessModeWrite` |
| `UpdateNode` | `AccessModeWrite` |
| `DeleteNode` | `AccessModeWrite` (DETACH DELETE) |
| `CreateEdge` | `AccessModeWrite` |
| `ExecuteQuery` | `AccessModeRead` |

**Namespace isolation**: every node and edge carries an `app_id` property. All queries include `{app_id: $app_id}` as a filter to guarantee tenant isolation.

**Namespace reservation**: When an `App` is registered, `registryRepo.CreateApp` executes a `MERGE` on the `__KGS_Namespace` node in Neo4j:

```cypher
MERGE (n:__KGS_Namespace {app_id: $app_id})
ON CREATE SET n.created_at = datetime()
```

---

### 6.2 PostgreSQL — Relational Metadata

Managed by **GORM** with auto-migration on startup. Stores:

| Table | Model | Description |
|-------|-------|-------------|
| `apps` | `App` | Registered tenant applications |
| `api_keys` | `APIKey` | Hashed API keys (SHA-256) with scopes and TTL |
| `quotas` | `Quota` | Per-app resource limits |
| `audit_logs` | `AuditLog` | Administrative action log |
| `entity_types` | `EntityType` | Ontology node schemas (JSON Schema) |
| `relation_types` | `RelationType` | Ontology edge schemas with source/target constraints |
| `rules` | `Rule` | Business rules with Cypher queries |
| `rule_executions` | `RuleExecution` | Rule execution history (SUCCESS / FAILED) |
| `policies` | `Policy` | OPA Rego policy documents |

---

### 6.3 Redis — Cache and Event Stream

Two usage modes:

| Mode | Key/Stream | Purpose |
|------|-----------|---------|
| **Cache** | `ontology:entity:{appID}:{name}` | Ontology entity type cache (5-min TTL) |
| **Cache** | `ontology:relation:{appID}:{name}` | Ontology relation type cache (5-min TTL) |
| **Stream** | `kgs:events:nodes` | Domain event bus for node mutations |

---

### 6.4 Qdrant — Vector Store

REST-based client (`QdrantClient`) providing:

| Method | Endpoint | Purpose |
|--------|---------|---------|
| `Upsert` | `PUT /collections/{col}/points` | Upsert embedding vectors |
| `Search` | `POST /collections/{col}/points/search` | ANN search with score threshold |

Default collection: `knowledge_chunks`. Used for semantic similarity retrieval over knowledge graph content.

---

### 6.5 Ontology Seed

On every startup, `SeedOntology` populates the default **knowledge ontology** for the `system` app context into PostgreSQL (idempotent — skips existing records):

**19 Entity Types** across 4 layers:

| Layer | Entity Types |
|-------|-------------|
| PRD/URD | Feature, UserStory, BusinessRule, Actor, UseCase, DataEntity, Constraint |
| SRS | SRSRequirement, SystemInterface |
| UI Doc | UIScreen, UIComponent, UIFlow, UIValidationRule |
| Test Artifacts | TestRequirement, TestDesign, TestCase, TestSuite, TestScript |

**16 Relation Types** including: `REFINES`, `DERIVES_FROM`, `TESTS`, `IMPLEMENTS`, `GROUPS`, `SPECIFIES_INTERFACE`, `RENDERED_ON`, `CONTAINS_COMPONENT`, `NAVIGATES_TO`, `VALIDATES_FIELD`, `TESTED_ON_SCREEN`, `HAS_CHILD`, `DEPENDS_ON`, `RELATED_TO`, `AUTOMATES`, `PART_OF`

---

## 7. Event-Driven Integration

### 7.1 Kafka Consumer — Document Ingestion

The service subscribes to the `document.ingested` Kafka topic (consumer group: `knowledge-service`) and processes `DocumentIngestedEvent` messages:

```go
type DocumentIngestedEvent struct {
    DocID      string
    AppID      string
    DocType    string         // PRD|SRS|UI|TESTCASE
    NodeType   string         // KG node label
    Properties map[string]any
    ParentID   string         // Optional
    EdgeType   string         // Optional
}
```

**Processing**: For each event, the consumer calls `GraphUsecase.CreateNode`. If `ParentID` and `EdgeType` are present, it also calls `CreateEdge` to link the node to its parent — enabling automatic graph population from document analysis pipelines.

---

### 7.2 Redis Streams — Internal Events

Graph mutations (Create/Update/Delete) published to `kgs:events:nodes` stream with fields:

| Field | Value |
|-------|-------|
| `event_type` | `node.created` \| `node.updated` \| `node.deleted` |
| `app_id` | Tenant identifier |
| `label` / `node_id` | Context-dependent |

The `EventRunner` consumes these events to trigger `ON_WRITE` business rules.

---

## 8. Security Design

### 8.1 Multi-Tenancy

All data is namespaced by `app_id`:
- Neo4j: every node/edge carries `{app_id: $app_id}` property
- PostgreSQL: all tables have indexed `app_id` column
- Namespace nodes (`__KGS_Namespace`) prevent cross-tenant data leakage

### 8.2 Access Control

All requests pass through the `Auth` middleware which validates the API key. The `OPAClient` provides fine-grained attribute-based access control:

- **Input**: `{ app_id, action, resource }`
- **Decision endpoint**: `POST /v1/data/kgs/allow`
- **Fail-closed**: if OPA is unreachable, the request is denied
- **Default policy**: deny-all; specific permissions via Rego rules

Policies are stored in PostgreSQL and synced to OPA every 30 seconds via `PolicySyncRunner`.

### 8.3 API Key Model

| Attribute | Description |
|-----------|-------------|
| Storage | SHA-256 hash stored; plaintext returned only at issuance |
| Scopes | Comma-separated (e.g., `read,write`) |
| TTL | Configurable; `0` = never expires |
| Prefix | First few chars for identification without exposing the full hash |

---

## 9. Worker Server

The `WorkerServer` implements `transport.Server` (Kratos interface) and is registered alongside HTTP/gRPC servers. It manages the lifecycle of all background processes:

| Worker | Trigger | Managed Via |
|--------|---------|-------------|
| `RuleRunner` | CRON schedule | `gocron` scheduler |
| `EventRunner` | Redis Stream `kgs:events:nodes` | Blocking `XREADGROUP` loop |
| `PolicySyncRunner` | Every 30 seconds | `time.Ticker` |
| `Kafka Consumer` | `document.ingested` topic | `kafka-go` reader goroutine |

All workers implement graceful `Start/Stop` semantics.

---

## 10. Configuration (`configs/config.yaml`)

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s

data:
  database:
    driver: postgres
    source: host=... user=... password=... dbname=kgs_platform port=5432 ...
  redis:
    addr: 127.0.0.1:6379
    read_timeout: 0.2s
    write_timeout: 0.2s
  neo4j:
    uri: bolt://localhost:7687
    user: neo4j
    password: ...
    database: kgs
  opa:
    url: http://localhost:8181
  qdrant:
    url: http://localhost:6333
    collection: knowledge_chunks
  kafka:
    brokers:
      - localhost:9092
    topic_document_ingested: document.ingested
```

Configuration schema is defined in `internal/conf/conf.proto` and compiled to a Go struct. The service accepts a directory path (`-conf /app/configs`) and merges YAML files at startup.

---

## 11. Deployment

### 11.1 Docker Image

Multi-stage build:

| Stage | Base | Purpose |
|-------|------|---------|
| `builder` | `golang:1.25rc2-alpine` | Compile static binary (`CGO_ENABLED=0`) |
| Runtime | `alpine:3.20` | Minimal runtime with `ca-certificates`, `tzdata`, `netcat-openbsd` |

**Exposed ports**: `8000` (HTTP), `9000` (gRPC)  
**Entry point**: `./server -conf /app/configs`

### 11.2 External Dependencies

| Service | Port | Protocol |
|---------|------|---------|
| PostgreSQL | 5432 | TCP |
| Neo4j | 7687 | Bolt |
| Redis | 6379 | TCP |
| OPA (sidecar) | 8181 | HTTP |
| Qdrant | 6333 | HTTP |
| Kafka | 9092 | TCP |

---

## 12. Dependency Injection

The service uses **Google Wire** for compile-time dependency injection. Provider sets are registered at each layer:

| Layer | ProviderSet exported from |
|-------|--------------------------|
| `biz` | `NewGraphUsecase`, `NewRulesUsecase`, `NewPolicyUsecase`, `NewOPAClient`, `NewQueryPlanner`, `NewRuleRunner`, `NewEventRunner`, `NewPolicySyncRunner`, `NewOntologySyncManager`, `NewViewResolver` |
| `data` | `NewData`, `NewGraphRepo`, `NewRegistryRepo`, `NewOntologyRepo`, `NewRulesRepo`, `NewPolicyRepo`, `NewRedisClient` |
| `service` | `NewGraphService`, `NewOntologyService`, `NewRegistryService`, `NewRulesService`, `NewPolicyService` |
| `server` | `NewHTTPServer`, `NewGRPCServer`, `NewWorkerServer` |

---

## 13. Known Gaps and Future Work

| Area | Current State | Planned Work |
|------|--------------|--------------|
| `GetNode` | Returns empty reply (stub) | Full Neo4j node fetch by ID |
| `GetContext/Impact/Coverage` | Calls Usecase but doesn't map result to proto | Implement result-to-proto mapping |
| AppID propagation | Hardcoded `"system"` / `"demo-app"` in service layer | Extract `app_id` from Auth middleware context |
| `OntologySyncManager` | Stub — logs and returns | Implement constraint generation and Neo4j sync |
| `PolicySyncRunner` | Hardcoded `"demo-app"` tenant | Iterate over all active apps |
| `RuleRunner` | Hardcoded `"demo-app"` tenant | Multi-tenant rule iteration |
| `ViewResolver` | Stub field shaping | Full view definition from Postgres |
| `Rule.Action` | Logged but not dispatched | Implement webhook / push notification dispatchers |
| `RuleExecution` | Model defined but not written | Write execution log on RuleRunner completion |
| OPA `PutPolicy` URL | Hardcoded `localhost:8181` | Use configured OPA base URL |

---

## 14. Key Design Decisions

### D1 — Dual Transport (gRPC + HTTP)
Using Kratos `google.api.http` annotations provides both gRPC for internal high-performance service-to-service calls and REST HTTP for external clients and tooling without duplicating service logic.

### D2 — Namespace-per-tenant in Neo4j
Rather than maintaining separate Neo4j databases per tenant, all nodes share a single graph namespace. Tenant isolation is enforced by the `app_id` property and parameterized queries. This reduces operational overhead while maintaining strict isolation at the application layer.

### D3 — OPA as a Sidecar
The OPA sidecar pattern decouples policy evaluation from service business logic. The `PolicySyncRunner` periodic sync ensures policies defined in the database are always reflected without requiring service restarts.

### D4 — Fail-Closed OPA
If OPA is unreachable, all write operations are denied. This is an explicit security decision: data integrity is preferred over availability for write operations.

### D5 — Redis Streams for Internal Events
Instead of a full Kafka loop-back for internal graph events, Redis Streams provide a lightweight, low-latency event bus for the `ON_WRITE` rule trigger pattern, avoiding cross-service Kafka overhead for internal workflows.

### D6 — QueryPlanner Isolation
The `QueryPlanner` centralizes all Cypher generation. Since Neo4j does not support parameterized labels/relationship types, the QueryPlanner safely manages string interpolation while keeping all value bindings parameterized — preventing injection while allowing dynamic schema queries.
