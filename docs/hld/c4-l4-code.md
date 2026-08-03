# C4 Level 4 — Code (Key Data Types & Flows)

> **C4 Level**: 4 — Code  
> **Câu hỏi**: Các data type, interface, và flow chi tiết trong code là gì?  
> **Audience**: Backend developers  
> **Note**: L4 thường auto-generated từ code. File này chỉ document các **key structures** và **critical algorithms** không hiển nhiên từ code.

---

## 4.1 Core Domain Types

### Identity Types

```go
// internal/identity/
type Tenant struct {
    ID                    string    // UUID
    Slug                  string    // "payment-team"
    Name                  string
    Status                string    // "active" | "suspended" | "trial"
    Tier                  string    // "free" | "pro" | "enterprise"
    DefaultSharingPolicy  string    // "deny_all" | "share_within_tenant_read"
}

type App struct {
    ID            string    // UUID
    TenantID      string    // UUID
    Slug          string
    Type          string    // "agent_consumer" | "ingestion_producer" | ...
    APIKeyHash    string    // SHA256, never stored plaintext
    APIKeyPrefix  string    // 8 chars, display only
    Status        string    // "active" | "revoked"
}

type CallerIdentity struct {
    TenantID string
    AppID    string
}
```

### Access Control Types

```go
// internal/access/
type AccessGrant struct {
    ID               string
    GrantorTenantID  string
    GrantorAppID     string   // "" = entire tenant
    GranteeTenantID  string
    GranteeAppID     string   // "" = entire tenant
    ScopeType        string   // "domain" | "node_type" | "dataset_tag" | "all"
    ScopeValue       string   // "" when scope_type = "all"
    Permission       string   // "read" | "search" | "write" | "admin"
    Status           string   // "active" | "revoked" | "expired"
    ExpiresAt        *time.Time
}

type VisibleOwner struct {
    TenantID   string
    AppID      string   // "*" = any app of this tenant
    ScopeType  string   // applicable when from grant
    ScopeValue string
    Permission string
}

// Computed per request, cached in Redis
type VisibilityContext struct {
    CallerTenantID string
    CallerAppID    string
    VisibleOwners  []VisibleOwner
    ACLTokens      []string   // ["tenant-A:app-X", "tenant-A:*", "platform:*"]
}
```

### Ontology Types

```go
// internal/ontology/
type Domain struct {
    ID            string   // "payment-errors"
    Name          string
    OwnerTenantID string
    Status        string   // "draft" | "active" | "deprecated"
    Visibility    string   // "public" | "tenant_shared" | "private"
    Version       int
}

type NodeTypeSchema struct {
    ID              string   // "{domain_id}.{node_type_name}"
    DomainID        string
    NodeTypeName    string
    RequiredProps   []PropDef
    OptionalProps   []PropDef
    ValidationRules []string
}

type PropDef struct {
    Name        string
    Type        string   // "string" | "number" | "boolean" | "string[]" | "enum"
    EnumValues  []string // when Type == "enum"
    Description string
}

// Query Pattern DSL — stored as JSONB, NOT Cypher
type PatternSpec struct {
    Start  StartNode  `json:"start"`
    Hops   []Hop      `json:"hops"`
    Return []string   `json:"return_fields"`
}

type StartNode struct {
    NodeType string            `json:"node_type"`
    Match    map[string]string `json:"match"`  // { "field": "$param" }
}

type Hop struct {
    RelType      string `json:"rel_type"`
    ToNodeType   string `json:"to_node_type"`
    Direction    string `json:"direction"`    // "out" (default) | "in"
    Filter       map[string]any `json:"filter,omitempty"`
    FilterStatus string `json:"filter_status,omitempty"` // "valid_only"
}

type StatusFieldConfig struct {
    DomainID            string
    StatusFieldName     string   // nullable = no lifecycle
    ValidStatusValues   []string
    WarningStatusValues []string
    CascadeRules        []CascadeRule
    AuthorityFieldName  string   // nullable = no ranking
    AuthorityValuesMap  map[string]int
}
```

### KG Data Types

```go
// internal/write/
type Node struct {
    ID            string
    NodeType      string
    DomainID      string
    OwnerTenantID string
    OwnerAppID    string
    Visibility    string   // "public" | "tenant_shared" | "private"
    Properties    map[string]any
    ExternalRef   string   // sync key Graph/Vector (optional)
    StatusValue   string   // mapped from domain status field
    IsDeleted     bool
    DomainVersion int
}

type Relationship struct {
    ID            string
    RelType       string
    FromNodeID    string
    ToNodeID      string
    DomainID      string
    OwnerTenantID string
    OwnerAppID    string
    Properties    map[string]any
}

type OutboxEvent struct {
    ID            string
    AggregateType string   // "kg_node" | "kg_relationship" | "access_grant"
    AggregateID   string
    EventType     string   // "NODE_UPSERTED" | "NODE_DELETED" | "REL_UPSERTED" | ...
    Payload       map[string]any
    Status        string   // "PENDING" | "PROCESSING" | "DONE" | "FAILED" | "DEAD_LETTER"
    RetryCount    int
}
```

### Worker Types

```go
// internal/workers/types.go
type GraphNode struct {
    ID           string
    NodeType     string
    DomainID     string
    ACLVisibleTo []string       // ["tenant-A:app-X", "platform:*"]
    SyncVersion  int64          // _kg_sync_version
    Properties   map[string]any
    // ...
}

type VectorDocument struct {
    NodeID        string
    NodeType      string
    DomainID      string
    ACLVisibleTo  []string
    StatusValue   string
    AuthorityScore *int
    SyncVersion   int64
    DomainProps   map[string]any   // schema-less, domain-specific fields
    Embedding     []float64
}

type EntitySyncStatus struct {
    EntityID       string
    SourceVersion  int64
    GraphVersion   int64
    GraphLagClass  SyncLagClass  // SYNCED | IN_FLIGHT | LAGGING | STUCK
    VectorVersion  int64
    VectorLagClass SyncLagClass
}

type ReconciliationReport struct {
    GraphDriftCount   int
    VectorDriftCount  int
    GraphLaggingCount int
    Issues            []ReconciliationIssue
    Overall           string   // "pass" | "warn" | "fail"
}
```

---

## 4.2 Key Algorithms

### Algorithm 1: ACL Token Resolution

```
Input:  tenant_id, app_id
Output: []string (ACL tokens for query injection)

Steps:
1. Check Redis cache "acl:{tenant_id}:{app_id}"
   HIT → return cached tokens

2. MISS → compute:
   tokens = []

   // Self access
   tokens.append("{tenant_id}:{app_id}")
   tokens.append("{tenant_id}:")   // tenant-wide data

   // Platform access (always)
   tokens.append("platform:*")

   // Default sharing policy
   tenant = db.GetTenant(tenant_id)
   if tenant.DefaultSharingPolicy == "share_within_tenant_read":
     tokens.append("{tenant_id}:*")

   // Explicit grants
   grants = db.QueryActiveGrants(grantee_tenant_id=tenant_id, grantee_app_id=app_id)
   for grant in grants:
     if grant.GrantorAppID == "":
       tokens.append("{grant.GrantorTenantID}:*")
     else:
       tokens.append("{grant.GrantorTenantID}:{grant.GrantorAppID}")

3. Store in Redis TTL 60s
4. Return tokens

Invalidation trigger: ACCESS_GRANT_CHANGED event
  → DEL Redis "acl:{grantee_tenant_id}:*"
```

### Algorithm 2: Query Pattern DSL → Cypher Compilation

```
Input:  pattern_spec (PatternSpec), domain_id, acl_tokens
Output: (cypher_string, params)

Steps:
1. Load StatusFieldConfig for domain_id (may be nil)

2. Build MATCH for start node:
   "MATCH (n0:{NodeType} {field: $param})"
   "WHERE ANY(tok IN n0.acl_visible_to WHERE tok IN $acl_tokens)"
   // ↑ ALWAYS injected, cannot be disabled

3. For each hop in pattern_spec.Hops:
   alias = "n{i+1}"
   direction_arrow = if hop.Direction == "in": "<-" else: "->"

   MATCH clause:
   "(n{i}){arrow}[:{RelType}]{arrow}({alias}:{ToNodeType})"

   // ACL filter EVERY hop — security invariant
   "WHERE ANY(tok IN {alias}.acl_visible_to WHERE tok IN $acl_tokens)"

   // Optional property filter
   if hop.Filter != nil:
     add property conditions to WHERE

   // Status filter (generic — reads from domain config)
   if hop.FilterStatus == "valid_only" AND StatusFieldConfig != nil:
     "AND {alias}.{status_field_name} IN {valid_status_values}"

4. RETURN clause from return_fields

5. Bind $acl_tokens as query parameter

Constraint: max 5 hops (validated at registration time → 422 TEMPLATE_TOO_DEEP)
```

### Algorithm 3: compute_acl_visible_to (for Sync Workers)

```
Input:  node (Node struct)
Output: []string (tokens to denormalize onto Graph + Vector)

Steps:
1. tokens = ["{owner_tenant_id}:{owner_app_id}"]

2. Visibility expansion:
   if node.Visibility == "public":
     tokens.append("*:*")
   if node.Visibility == "tenant_shared":
     tokens.append("{owner_tenant_id}:*")

3. Active grants WHERE grantor covers this node:
   grants = db.QueryActiveGrants(
     grantor_tenant_id = node.OwnerTenantID,
     scope covers node.DomainID
   )
   for grant in grants:
     tokens.append("{grantee_tenant_id}:{grantee_app_id or '*'}")

4. Return deduplicated tokens
```

### Algorithm 4: Realtime Read Fallback

```
Input:  node_id, app_id, mode ("realtime" | "non-realtime")
Output: node data

if mode == "non-realtime":
  → GraphDB.GetNode(node_id, acl_tokens)
  → return graph result

if mode == "realtime":
  pg_version = PostgreSQL.GetNodeSyncVersion(node_id)
  graph_version = GraphDB.GetNodeSyncVersion(node_id)

  if graph_version >= pg_version:
    → GraphDB.GetNode(node_id, acl_tokens)  // fresh enough
  else:
    → PostgreSQL.GetNode(node_id)           // fallback to source-of-truth
    → add metadata: { "realtime_fallback": true, "graph_lag": pg_version - graph_version }
```

---

## 4.3 Platform Adapter Interfaces

```go
// GraphStore interface (internal/platform/graphstore/adapter.go)
type GraphStore interface {
    MergeNode(ctx context.Context, node GraphNode) error
    MergeRelationship(ctx context.Context, rel GraphRelationship) error
    DeleteNode(ctx context.Context, nodeID string) error
    DeleteRelationship(ctx context.Context, relID string) error
    GetNode(ctx context.Context, nodeID string, aclTokens []string) (*GraphNode, error)
    QueryTemplate(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error)
    UpdateProperty(ctx context.Context, nodeID string, key string, value any) error
    GetSyncVersion(ctx context.Context, nodeID string) (int64, error)
}

// VectorStore interface (internal/platform/vectorstore/adapter.go)
type VectorStore interface {
    Upsert(ctx context.Context, doc VectorDocument) error
    Search(ctx context.Context, vector []float64, filter VectorFilter, topK int) ([]SearchResult, error)
    Delete(ctx context.Context, nodeID string) error
    UpdatePayload(ctx context.Context, filter VectorFilter, payload map[string]any) error
    GetSyncVersion(ctx context.Context, nodeID string) (int64, error)
}

// FTS interface (internal/platform/fts/)
type FTSStore interface {
    Index(ctx context.Context, doc FTSDocument) error
    Search(ctx context.Context, query string, filter FTSFilter, topK int) ([]FTSResult, error)
    Delete(ctx context.Context, nodeID string) error
}
```

---

## 4.4 API Error Codes

| Error Code | HTTP | When |
|:---|:---:|:---|
| `INVALID_API_KEY` | 401 | API key không hợp lệ hoặc đã revoke |
| `FORBIDDEN` | 403 | Không có quyền thực hiện action |
| `NO_READ_ACCESS` | 403 | Không có quyền đọc node cụ thể |
| `NO_READ_ACCESS_TO_DOMAIN` | 403 | Domain private, không có grant |
| `TENANT_NOT_FOUND` | 404 | Tenant ID không tồn tại |
| `UNKNOWN_TEMPLATE` | 404 | Template không tồn tại hoặc chưa active |
| `VALIDATION_FAILED` | 422 | Dữ liệu không khớp ontology schema |
| `TEMPLATE_TOO_DEEP` | 422 | Query Pattern DSL vượt 5 hops |
| `DOMAIN_NOT_IN_EFFECTIVE_ONTOLOGY` | 422 | domain_id không trong effective ontology |
| `CROSS_TENANT_GRANT_REQUIRES_EXPIRY` | 400 | Cross-tenant write/admin grant thiếu expires_at |
| `RATE_LIMIT_EXCEEDED` | 429 | Vượt rate limit theo tier |
| `GRAPH_DB_TIMEOUT` | 503 | Graph DB không phản hồi trong 3000ms |
| `INTERNAL_ERROR` | 500 | Lỗi không mong đợi |

**Error envelope** (tất cả 4xx/5xx):
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Missing required property: severity",
    "details": [{ "field": "attributes.severity", "issue": "required" }],
    "request_id": "req_8f3a2b1c"
  }
}
```

---

## 4.5 MCP Tool Contracts

| Tool | Input Schema | Output |
|:---|:---|:---|
| `kg_search` | `{ query, domain_ids?, top_k? }` | `{ results: [{ node_ref, score, content, domain_id }] }` |
| `kg_search_rag` | `{ query, domain_ids?, top_k? }` | `{ answer_context: { structured_data, citations, disclaimer } }` |
| `kg_read_pattern` | `{ domain_id, template_name, params, app_id?, mode? }` | Per template return_fields schema |
| `kg_list_domains` | `{}` | `{ domains: [{ domain_id, owner, permission }] }` |
| `kg_list_templates` | `{ domain_id }` | `{ templates: [{ template_name, param_schema, description }] }` |
| `kg_get_node` | `{ id, app_id?, mode? }` | `{ node, relationships: [...] }` |
| `kg_write_node` | `{ domain_id, node_type, properties }` | `{ node_id, status }` |
| `kg_check_access` | `{}` | `{ visible_domains, visible_owners }` |
| `kg_integrity` | `{ tenant_id? }` | `{ checks: [...], overall }` |

**Authentication**: Connection-level (at `GET /v1/mcp/connect`), not per tool call.  
**Error format**: JSON-RPC error `{ code: -32000, message: "..." }` — not REST error envelope.

---

## 4.6 Sequence: Cross-Tenant Grant → Data Access

```
TenantA Admin
  POST /v1/access/grants
  { grantee_tenant_id: "B", scope_type: "domain", scope_value: "payment", permission: "search" }

  1. AuthZ: caller must be TenantA admin
  2. Validation: cross-tenant search grant (expires_at not required for search)
  3. INSERT access_grants (status=active)
  4. INSERT kg_outbox_events (ACCESS_GRANT_CHANGED)
  5. Redis: DEL "acl:B:*"  ← SYNCHRONOUS (not via queue)
  Response: 201 { id: "grant-xyz" }

  [Background — OutboxPoller picks up ACCESS_GRANT_CHANGED]

AccessSyncWorker:
  1. Find all kg_nodes WHERE owner_tenant_id = "A" AND domain_id = "payment"
  2. For each node:
     old_acl = node.acl_visible_to
     new_acl = compute_acl_visible_to(node)  ← now includes "B:*"
     if old_acl != new_acl:
       GraphDB.UpdateProperty(node.id, "acl_visible_to", new_acl)
       VectorDB.UpdatePayload(node.id, { "acl_visible_to": new_acl })
  3. Mark outbox event DONE

[~< 5 seconds after grant creation]

TenantB App1
  POST /v1/kg/search/semantic
  { query: "payment timeout errors", domain_ids: ["payment"] }

  1. IdentityResolver → (TenantB, App1)
  2. AccessResolver(TenantB, App1):
     Redis cache MISS (was invalidated)
     Recompute: includes GrantedOwner(A, *, domain, payment, search)
     acl_tokens = ["B:App1", "platform:*", "A:*"]  ← A:* added
     Cache result TTL 60s
  3. VectorDB.search(vector, filter={
       acl_visible_to: { any: ["B:App1", "platform:*", "A:*"] },
       domain_id: "payment",
       is_deleted: false
     })
  4. VectorDB finds nodes with "A:*" or "B:App1" in acl_visible_to
     → Returns TenantA's payment data ✓
  5. AuditLogger: { action: search, allowed: true, reason: "grant:grant-xyz" }
  Response: 200 { results: [...TenantA payment data...] }
```
