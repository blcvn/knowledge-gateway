# KGS — Knowledge Graph Service
## Platform Implementation Plan

> **Mô hình:** Shared Graph + Namespace Isolation  
> **Customization:** Ontology · Rules · Access Control · Response Views  
> **Ưu tiên Phase 1:** Core Platform — App Registry · Ontology Service · Graph API  
> **Tổng thời gian:** 8 tuần (4 sprints × 2 tuần)  
> **Stack:** Neo4j 5.x · PostgreSQL 16 · Redis 7 · OPA · FastAPI · Python 3.11+

---

## Mục Lục

1. [Nguyên Tắc Xuyên Suốt](#1-nguyên-tắc-xuyên-suốt)
2. [Kiến Trúc Module](#2-kiến-trúc-module)
3. [Sprint 1 — App Registry & Gateway](#3-sprint-1--app-registry--gateway-tuần-1-2)
4. [Sprint 2 — Ontology Service](#4-sprint-2--ontology-service-tuần-3-4)
5. [Sprint 3 — Graph API](#5-sprint-3--graph-api-tuần-5-6)
6. [Sprint 4 — Rule Engine & Access Control](#6-sprint-4--rule-engine--access-control-tuần-7-8)
7. [Onboarding Flow: BA Agent System](#7-onboarding-flow-ba-agent-system)
8. [Dependency Map](#8-dependency-map)
9. [Risk Register](#9-risk-register)
10. [Pre-Implementation Checklist](#10-pre-implementation-checklist)

---

## 1. Nguyên Tắc Xuyên Suốt

Những nguyên tắc này áp dụng cho **mọi** task trong plan, không lặp lại ở từng sprint.

**Namespace-first** — mọi Cypher query đều được inject prefix `{app_id}__` bởi Query Planner trước khi chạm Neo4j. Không có ngoại lệ. App developer không bao giờ tự viết namespace.

**PostgreSQL là source of truth** — ontology, rules, policies, app config đều sống trong Postgres. Neo4j chứa graph data. Redis chứa cache + queue. Nếu conflict, Postgres thắng.

**Async-first cho side effects** — rule execution, vector embedding, event propagation đều là async qua Redis Streams. Không có side effect nào block Graph API response.

**Additive-only schema changes** — ontology sau khi có data chỉ được thêm fields, không xóa hay đổi tên. Breaking change = tạo version mới.

**Soft delete everywhere** — không có hard delete trên graph data. Dùng `status=DEPRECATED`, `valid_to=datetime()`. Hard delete chỉ được phép cho app bị xóa hoàn toàn bởi Platform Admin.

**Fail loudly với error context** — mọi validation failure phải trả về error message bao gồm: `rule_violated`, `entity_type`, `field`, `received_value`. Không trả về generic 400.

---

## 2. Kiến Trúc Module

```
kgs/
├── gateway/            # Auth middleware, rate limit, audit log
├── registry/           # App CRUD, API key management, quota
├── ontology/           # Entity types, relation types, validation
├── graph/              # Node/edge CRUD, Query Planner, context API
├── rules/              # Rule CRUD, scheduler, event-driven runner
├── access_control/     # Policy CRUD, OPA integration, PDP
├── views/              # Response view definition + resolver
├── events/             # Redis Streams producer/consumer
├── storage/
│   ├── neo4j.py        # Driver wrapper, namespaced execute
│   ├── postgres.py     # Async connection pool
│   └── redis.py        # Cache + Streams client
└── shared/
    ├── models.py        # Pydantic models dùng chung
    ├── errors.py        # KGSError hierarchy
    └── constants.py
```

### Request Flow tổng quát

```
Client Request (API Key)
    │
    ▼
[Gateway] Auth → inject AppContext { app_id, scopes, rate_limit }
    │
    ▼
[Rate Limiter] Redis sliding window per app_id
    │
    ▼
[Router] → registry | ontology | graph | rules | access_control
    │
    ▼
[Service Layer]
    ├── Ontology cache lookup (Redis, TTL=5m)
    ├── Policy check → OPA evaluate
    ├── Payload JSON Schema validation
    └── Query Planner → inject namespace → execute Neo4j
    │
    ▼
[Async Events] → Redis Streams → Rule Engine / Vector Embedding
    │
    ▼
Response (plain hoặc view-resolved)
```

---

## 3. Sprint 1 — App Registry & Gateway (Tuần 1-2)

> **Mục tiêu:** Platform có thể đăng ký app, issue API key, authenticate request, enforce rate limit, và reserve namespace trong Neo4j.

---

### Task 1.1 — PostgreSQL Schema: Registry

**Effort:** 0.5 ngày | **File:** `registry/migrations/001_registry.sql`

```sql
CREATE TABLE kgs_applications (
    app_id          VARCHAR(50)  PRIMARY KEY,
    app_name        VARCHAR(200) NOT NULL,
    description     TEXT,
    owner_email     VARCHAR(200) NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE|SUSPENDED|DELETED
    settings        JSONB        DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ  DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE kgs_api_keys (
    key_id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          VARCHAR(50)  REFERENCES kgs_applications(app_id),
    key_hash        VARCHAR(64)  NOT NULL UNIQUE,   -- bcrypt hash
    key_prefix      VARCHAR(12)  NOT NULL,           -- 'kgs_ba_' hiển thị cho user
    label           VARCHAR(100),                    -- 'production', 'staging'
    scopes          TEXT[]       NOT NULL DEFAULT '{}',
    rate_limit_rpm  INTEGER      DEFAULT 1000,
    expires_at      TIMESTAMPTZ,
    last_used_at    TIMESTAMPTZ,
    is_active       BOOLEAN      DEFAULT TRUE,
    created_at      TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE kgs_quotas (
    app_id              VARCHAR(50) PRIMARY KEY REFERENCES kgs_applications(app_id),
    max_nodes           BIGINT      DEFAULT 1000000,
    max_edges           BIGINT      DEFAULT 5000000,
    max_entity_types    INTEGER     DEFAULT 50,
    max_relation_types  INTEGER     DEFAULT 100,
    max_rules           INTEGER     DEFAULT 200,
    max_rpm             INTEGER     DEFAULT 1000
);

CREATE TABLE kgs_audit_log (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      VARCHAR(50),
    key_id      UUID,
    method      VARCHAR(10),
    endpoint    TEXT,
    status_code INTEGER,
    latency_ms  INTEGER,
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX audit_app_time ON kgs_audit_log(app_id, created_at DESC);
```

**Done khi:** `alembic upgrade head` không lỗi, tất cả tables và index tồn tại.

---

### Task 1.2 — App Registry API

**Effort:** 2 ngày | **File:** `registry/router.py`

Endpoints cần implement:

| Method | Endpoint | Mô tả | Auth |
|---|---|---|---|
| `POST` | `/admin/apps` | Tạo app mới, tự tạo quota default | Platform Admin Key |
| `GET` | `/admin/apps` | List apps với filter status | Platform Admin Key |
| `GET` | `/admin/apps/{app_id}` | Chi tiết app + usage stats | Platform Admin Key |
| `PATCH` | `/admin/apps/{app_id}` | Cập nhật status (SUSPENDED/ACTIVE) | Platform Admin Key |
| `POST` | `/admin/apps/{app_id}/keys` | Issue API key mới | Platform Admin Key |
| `GET` | `/admin/apps/{app_id}/keys` | List keys (ẩn hash, hiện prefix) | Platform Admin Key |
| `DELETE` | `/admin/apps/{app_id}/keys/{key_id}` | Revoke key | Platform Admin Key |

```python
# registry/service.py — issue_api_key()
import secrets, bcrypt

def issue_api_key(app_id: str, label: str, scopes: list[str]) -> dict:
    plain_key = f"kgs_{app_id[:4]}_{secrets.token_urlsafe(32)}"
    key_hash  = bcrypt.hashpw(plain_key.encode(), bcrypt.gensalt()).decode()
    prefix    = plain_key[:12]   # 'kgs_ba_xxxxx'

    # Lưu vào DB (chỉ hash, không bao giờ lưu plain)
    db.insert("kgs_api_keys", {
        "app_id": app_id, "key_hash": key_hash,
        "key_prefix": prefix, "label": label, "scopes": scopes
    })

    # Trả plain key DUY NHẤT 1 lần này
    return { "key": plain_key, "prefix": prefix, "label": label }
```

**Done khi:** POST → GET → PATCH flow hoạt động, plain key không tồn tại trong DB.

---

### Task 1.3 — Gateway Auth Middleware

**Effort:** 1 ngày | **File:** `gateway/middleware.py`

```python
# gateway/middleware.py
class KGSAuthMiddleware:
    async def __call__(self, request: Request, call_next):
        raw_key = request.headers.get("Authorization", "").removeprefix("Bearer ")

        if not raw_key:
            return JSONResponse({"error": "missing_api_key"}, status_code=401)

        # 1. Tìm theo prefix (tránh full-table scan bcrypt)
        prefix   = raw_key[:12]
        key_row  = await db.fetchone(
            "SELECT * FROM kgs_api_keys WHERE key_prefix=$1 AND is_active=TRUE", prefix
        )
        if not key_row or not bcrypt.checkpw(raw_key.encode(), key_row["key_hash"].encode()):
            return JSONResponse({"error": "invalid_api_key"}, status_code=401)

        # 2. Kiểm tra app status
        app = await db.fetchone("SELECT * FROM kgs_applications WHERE app_id=$1", key_row["app_id"])
        if app["status"] != "ACTIVE":
            return JSONResponse({"error": "app_suspended"}, status_code=403)

        # 3. Inject AppContext vào request state
        request.state.app_context = AppContext(
            app_id=key_row["app_id"],
            scopes=key_row["scopes"],
            rate_limit_rpm=key_row["rate_limit_rpm"]
        )

        # 4. Cập nhật last_used_at async (không block)
        asyncio.create_task(update_last_used(key_row["key_id"]))

        return await call_next(request)
```

**Done khi:** Request không có key → 401. Key của App A trên endpoint App B → 403.

---

### Task 1.4 — Rate Limiter

**Effort:** 1 ngày | **File:** `gateway/rate_limiter.py`

```python
# Sliding window counter per app_id trong Redis
async def check_rate_limit(app_id: str, rpm: int) -> bool:
    key    = f"ratelimit:{app_id}:{int(time.time() // 60)}"
    count  = await redis.incr(key)
    if count == 1:
        await redis.expire(key, 120)   # 2 phút để dọn sạch key cũ
    if count > rpm:
        raise RateLimitExceeded(f"Rate limit {rpm} RPM exceeded for {app_id}")
    return True
```

**Done khi:** App vượt quá RPM → 429 với `Retry-After` header.

---

### Task 1.5 — Namespace Reservation trong Neo4j

**Effort:** 0.5 ngày | **File:** `registry/neo4j_provisioner.py`

```python
# Khi app được tạo: tạo 1 meta node đánh dấu namespace đã reserve
async def reserve_namespace(app_id: str):
    await neo4j.execute("""
        MERGE (ns:__KGS_Namespace { app_id: $app_id })
        ON CREATE SET ns.created_at = datetime(), ns.status = 'ACTIVE'
    """, app_id=app_id)

# Khi app bị delete: đánh dấu namespace retired
async def release_namespace(app_id: str):
    await neo4j.execute("""
        MATCH (ns:__KGS_Namespace { app_id: $app_id })
        SET ns.status = 'DELETED', ns.deleted_at = datetime()
    """, app_id=app_id)
```

**Done khi:** Mỗi app mới tạo có 1 `__KGS_Namespace` node trong Neo4j.

---

### 3.1 Definition of Done — Sprint 1

| Hạng mục | Tiêu chí | Test |
|---|---|---|
| App CRUD | Tạo, list, update status hoạt động | Postman collection |
| API Key | Issue → authenticate → revoke flow | Integration test |
| Auth | Key sai → 401, App suspended → 403 | Unit test |
| Isolation | Key App A không dùng được endpoint App B | Security test |
| Rate Limit | Vượt RPM → 429, sau 1 phút → OK | Load test |
| Namespace | App tạo → `__KGS_Namespace` node tồn tại | Neo4j query |
| Audit Log | Mọi request có entry trong kgs_audit_log | SQL count check |

---

## 4. Sprint 2 — Ontology Service (Tuần 3-4)

> **Mục tiêu:** App có thể khai báo entity types và relation types. Platform dùng ontology này để validate mọi write operation.

---

### Task 2.1 — PostgreSQL Schema: Ontology

**Effort:** 0.5 ngày | **File:** `ontology/migrations/002_ontology.sql`

```sql
CREATE TABLE kgs_entity_types (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          VARCHAR(50)  REFERENCES kgs_applications(app_id),
    type_name       VARCHAR(100) NOT NULL,
    display_name    VARCHAR(200),
    description     TEXT,
    id_property     VARCHAR(100) NOT NULL,       -- field nào là unique ID
    properties      JSONB        NOT NULL,        -- JSON Schema object
    required_props  TEXT[]       DEFAULT '{}',
    searchable_props TEXT[]      DEFAULT '{}',    -- đưa vào vector index
    is_system       BOOLEAN      DEFAULT FALSE,
    created_at      TIMESTAMPTZ  DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  DEFAULT NOW(),
    UNIQUE(app_id, type_name)
);

CREATE TABLE kgs_relation_types (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          VARCHAR(50)  REFERENCES kgs_applications(app_id),
    type_name       VARCHAR(100) NOT NULL,
    display_name    VARCHAR(200),
    description     TEXT,
    from_types      TEXT[]       NOT NULL,        -- entity types allowed sebagai source
    to_types        TEXT[]       NOT NULL,         -- entity types allowed sebagai target
    cardinality     VARCHAR(20)  DEFAULT 'MANY_TO_MANY',
    properties      JSONB        DEFAULT '{}',
    is_system       BOOLEAN      DEFAULT FALSE,
    UNIQUE(app_id, type_name)
);
```

**Done khi:** Migration chạy thành công, UNIQUE constraint hoạt động.

---

### Task 2.2 — Ontology CRUD API

**Effort:** 2 ngày | **File:** `ontology/router.py`

| Method | Endpoint | Scope Required |
|---|---|---|
| `POST` | `/apps/{app_id}/ontology/entity-types` | `ontology:manage` |
| `GET` | `/apps/{app_id}/ontology/entity-types` | `ontology:read` |
| `PUT` | `/apps/{app_id}/ontology/entity-types/{name}` | `ontology:manage` |
| `DELETE` | `/apps/{app_id}/ontology/entity-types/{name}` | `ontology:manage` |
| `POST` | `/apps/{app_id}/ontology/relation-types` | `ontology:manage` |
| `GET` | `/apps/{app_id}/ontology/relation-types` | `ontology:read` |
| `PUT` | `/apps/{app_id}/ontology/relation-types/{name}` | `ontology:manage` |
| `GET` | `/apps/{app_id}/ontology` | `ontology:read` |
| `POST` | `/apps/{app_id}/ontology/validate` | `ontology:read` |

```python
# ontology/service.py — create_entity_type()
async def create_entity_type(app_id: str, payload: EntityTypeCreate) -> EntityType:
    # 1. Kiểm tra quota
    current = await count_entity_types(app_id)
    quota   = await get_quota(app_id)
    if current >= quota.max_entity_types:
        raise QuotaExceeded(f"max_entity_types={quota.max_entity_types} reached")

    # 2. Validate JSON Schema syntax của properties field
    validate_json_schema(payload.properties)

    # 3. Lưu vào Postgres
    et = await db.insert("kgs_entity_types", { "app_id": app_id, **payload.dict() })

    # 4. Tạo Neo4j uniqueness constraint cho id_property
    label = f"{app_id}__{payload.type_name}"
    await neo4j.execute(f"""
        CREATE CONSTRAINT {label}_id_unique IF NOT EXISTS
        FOR (n:{label}) REQUIRE n.{payload.id_property} IS UNIQUE
    """)

    # 5. Invalidate ontology cache
    await redis.delete(f"ontology:{app_id}")

    return et
```

**Quy tắc PUT (update):**
- Chỉ cho phép **thêm** properties mới vào schema
- Không được xóa required_props đã có
- Không được thay đổi `id_property`
- Trả về 409 nếu vi phạm backward compatibility

**Done khi:** Tạo entity type → Neo4j constraint tồn tại. Update thêm field → OK. Update xóa field → 409.

---

### Task 2.3 — JSON Schema Validator

**Effort:** 1.5 ngày | **File:** `ontology/validator.py`

```python
# ontology/validator.py
from jsonschema import validate, ValidationError as SchemaError

class OntologyValidator:
    def __init__(self, ontology_cache: OntologyCache):
        self.cache = ontology_cache

    async def validate_node_payload(self, app_id: str, entity_type: str, props: dict):
        et = await self.cache.get_entity_type(app_id, entity_type)
        if not et:
            raise OntologyError(
                rule_violated="entity_type_not_registered",
                entity_type=entity_type,
                message=f"Entity type '{entity_type}' không tồn tại trong ontology của app '{app_id}'"
            )
        try:
            validate(instance=props, schema={
                "type": "object",
                "properties": et.properties,
                "required": et.required_props,
                "additionalProperties": False   # strict mode
            })
        except SchemaError as e:
            raise OntologyError(
                rule_violated="json_schema_validation",
                field=e.path[-1] if e.path else None,
                received_value=e.instance,
                message=e.message
            )

    async def validate_edge_payload(self, app_id: str, rel_type: str,
                                     from_type: str, to_type: str, props: dict):
        rt = await self.cache.get_relation_type(app_id, rel_type)
        if not rt:
            raise OntologyError(rule_violated="relation_type_not_registered", ...)

        if from_type not in rt.from_types:
            raise OntologyError(rule_violated="relation_source_type_not_allowed", ...)

        if to_type not in rt.to_types:
            raise OntologyError(rule_violated="relation_target_type_not_allowed", ...)
```

**Done khi:** Node với field sai type → 422 với `rule_violated` + `field` + `received_value` rõ ràng.

---

### Task 2.4 — Ontology Cache

**Effort:** 0.5 ngày | **File:** `ontology/cache.py`

```python
# Cache ontology per app trong Redis, TTL = 5 phút
class OntologyCache:
    TTL = 300  # seconds

    async def get_entity_type(self, app_id: str, type_name: str) -> EntityType | None:
        key = f"ontology:{app_id}:et:{type_name}"
        cached = await redis.get(key)
        if cached:
            return EntityType.parse_raw(cached)
        et = await db.fetchone("SELECT * FROM kgs_entity_types WHERE app_id=$1 AND type_name=$2",
                               app_id, type_name)
        if et:
            await redis.setex(key, self.TTL, EntityType(**et).json())
        return EntityType(**et) if et else None

    async def invalidate_app(self, app_id: str):
        # Xóa tất cả cache của app khi ontology thay đổi
        keys = await redis.keys(f"ontology:{app_id}:*")
        if keys:
            await redis.delete(*keys)
```

**Done khi:** Second request ontology không hit Postgres (verify qua PG query log).

---

### Task 2.5 — Neo4j Constraint Auto-Sync

**Effort:** 1 ngày | **File:** `ontology/neo4j_sync.py`

```python
# Chạy khi platform khởi động: sync constraints từ Postgres → Neo4j
async def sync_all_constraints():
    entity_types = await db.fetch("SELECT app_id, type_name, id_property FROM kgs_entity_types")
    for et in entity_types:
        label    = f"{et['app_id']}__{et['type_name']}"
        id_prop  = et["id_property"]
        await neo4j.execute(f"""
            CREATE CONSTRAINT {label}_unique IF NOT EXISTS
            FOR (n:{label}) REQUIRE n.{id_prop} IS UNIQUE
        """)
```

**Done khi:** Restart platform → tất cả constraints vẫn tồn tại trong Neo4j.

---

### 4.1 Definition of Done — Sprint 2

| Hạng mục | Tiêu chí | Test |
|---|---|---|
| Entity Type CRUD | Tạo, read, update (additive) hoạt động | Integration test |
| Relation Type CRUD | Tạo, read, validate from/to_types | Integration test |
| Neo4j Constraint | Tạo entity type → constraint xuất hiện Neo4j | `SHOW CONSTRAINTS` |
| Schema Validation | Node sai field → 422 với error detail đầy đủ | Unit test (10 cases) |
| Additive-only | PUT thêm field → OK; PUT xóa field → 409 | Unit test |
| Cache | Ontology lookup không hit PG sau first request | PG slow query log |
| Quota | Vượt max_entity_types → 429 với quota info | Unit test |

---

## 5. Sprint 3 — Graph API (Tuần 5-6)

> **Mục tiêu:** App Service có thể CRUD nodes và edges, query context, impact analysis, coverage — tất cả namespace-aware và validated.

---

### Task 3.1 — Query Planner

**Effort:** 2 ngày | **File:** `graph/query_planner.py`

Query Planner là component quan trọng nhất của Sprint 3. Mọi Cypher đều đi qua đây.

```python
class QueryPlanner:
    def __init__(self, app_id: str, ontology: OntologyCache):
        self.app_id   = app_id
        self.ontology = ontology

    def ns(self, entity_type: str) -> str:
        """Tạo namespaced Neo4j label"""
        return f"{self.app_id}__{entity_type}"

    def build_create_node(self, entity_type: str, props: dict) -> CypherQuery:
        label = self.ns(entity_type)
        return CypherQuery(
            cypher=f"CREATE (n:{label} $props) RETURN n",
            params={"props": {**props, "_kgs_app_id": self.app_id,
                              "_kgs_created_at": datetime.utcnow().isoformat()}}
        )

    def build_find_nodes(self, entity_type: str, filters: dict,
                          limit: int = 50, offset: int = 0) -> CypherQuery:
        label   = self.ns(entity_type)
        clauses = [f"n.`{k}` = ${k}" for k in filters]
        where   = ("WHERE " + " AND ".join(clauses)) if clauses else ""
        return CypherQuery(
            cypher=f"MATCH (n:{label}) {where} RETURN n SKIP {offset} LIMIT {limit}",
            params=filters
        )

    def build_create_edge(self, rel_type: str,
                           from_label: str, from_id_prop: str, from_id_val: str,
                           to_label:   str, to_id_prop:   str, to_id_val:   str,
                           props: dict) -> CypherQuery:
        fn = self.ns(from_label)
        tn = self.ns(to_label)
        return CypherQuery(
            cypher=f"""
                MATCH (a:{fn} {{ {from_id_prop}: $from_val }})
                MATCH (b:{tn} {{ {to_id_prop}:   $to_val   }})
                CREATE (a)-[r:{rel_type} $props]->(b)
                RETURN r
            """,
            params={"from_val": from_id_val, "to_val": to_id_val,
                    "props": {**props, "_kgs_app_id": self.app_id,
                              "_kgs_created_at": datetime.utcnow().isoformat()}}
        )

    def build_subgraph(self, entity_type: str, node_id: str,
                        depth: int, max_nodes: int = 500) -> CypherQuery:
        label    = self.ns(entity_type)
        id_prop  = f"{entity_type}.id"    # lookup từ ontology thực tế
        prefix   = self.app_id
        return CypherQuery(
            cypher=f"""
                MATCH (start:{label} {{ {id_prop}: $node_id }})
                CALL apoc.path.subgraphAll(start, {{
                    maxLevel: $depth,
                    labelFilter: '+{prefix}__',
                    limit: $max_nodes
                }}) YIELD nodes, relationships
                RETURN nodes, relationships
            """,
            params={"node_id": node_id, "depth": depth, "max_nodes": max_nodes}
        )

    def build_impact_analysis(self, entity_type: str, node_id: str) -> CypherQuery:
        label   = self.ns(entity_type)
        prefix  = self.app_id
        id_prop = "req_id"   # từ ontology
        return CypherQuery(
            cypher=f"""
                MATCH (r:{label} {{ {id_prop}: $node_id }})
                CALL apoc.path.subgraphAll(r, {{
                    maxLevel: 4,
                    labelFilter: '+{prefix}__'
                }}) YIELD nodes, relationships
                WITH nodes, relationships,
                     [rel IN relationships | rel.impact_weight * rel.confidence] AS weights
                RETURN
                    [n IN nodes | {{ id: n.{id_prop}, label: labels(n)[0] }}] AS affected_nodes,
                    round(reduce(s=0.0, w IN weights | s + w) / size(weights), 2) AS impact_score
            """,
            params={"node_id": node_id}
        )

    def build_coverage_check(self, entity_type: str,
                               relation_type: str, status_filter: str = None) -> CypherQuery:
        label  = self.ns(entity_type)
        where  = f"WHERE n.status = '{status_filter}'" if status_filter else ""
        return CypherQuery(
            cypher=f"""
                MATCH (n:{label}) {where}
                WHERE NOT (n)-[:{relation_type}]->()
                RETURN n
            """,
            params={}
        )
```

**Done khi:** Unit test cho 5 query builders, verify namespaced labels trong output Cypher.

---

### Task 3.2 — Node CRUD

**Effort:** 1.5 ngày | **File:** `graph/nodes/router.py`

**Validation pipeline cho mọi write:**

```
1. Auth check (gateway đã xử lý)
2. Scope check: graph:write required
3. Ontology lookup: entity_type có tồn tại không?
4. JSON Schema validation qua OntologyValidator
5. Policy check qua OPA
6. Namespace inject qua QueryPlanner
7. Execute Neo4j
8. Emit event → Redis Streams
```

| Method | Endpoint | Response |
|---|---|---|
| `POST` | `/apps/{app_id}/nodes` | 201 + node data |
| `GET` | `/apps/{app_id}/nodes/{node_id}` | 200 + node data (với optional `?view=`) |
| `PATCH` | `/apps/{app_id}/nodes/{node_id}` | 200 + updated node |
| `DELETE` | `/apps/{app_id}/nodes/{node_id}` | 204 (soft delete: status=DEPRECATED) |
| `GET` | `/apps/{app_id}/nodes` | 200 + paginated list |
| `POST` | `/apps/{app_id}/nodes/batch` | 207 + per-item result |
| `GET` | `/apps/{app_id}/nodes/{node_id}/neighbors` | 200 + neighbors |

```python
# graph/nodes/service.py
async def create_node(app_context: AppContext, entity_type: str, props: dict) -> Node:
    planner   = QueryPlanner(app_context.app_id, ontology_cache)
    validator = OntologyValidator(ontology_cache)

    # Validate
    await validator.validate_node_payload(app_context.app_id, entity_type, props)

    # Policy check
    await policy_engine.check(app_context, resource=entity_type, action="CREATE")

    # Check quota
    await quota_service.check_node_quota(app_context.app_id)

    # Execute
    query  = planner.build_create_node(entity_type, props)
    result = await neo4j.execute(query.cypher, query.params)

    # Emit async event
    await event_bus.publish("node.created", {
        "app_id": app_context.app_id,
        "entity_type": entity_type,
        "node_id": props.get(id_property),
        "searchable": [props.get(f) for f in et.searchable_props]
    })

    return Node.from_neo4j(result)
```

**Done khi:** POST → GET → PATCH → DELETE flow, soft delete không xóa khỏi Neo4j.

---

### Task 3.3 — Edge CRUD

**Effort:** 1 ngày | **File:** `graph/edges/router.py`

```python
async def create_edge(app_context: AppContext, rel_type: str,
                       from_node_id: str, to_node_id: str, props: dict) -> Edge:
    # 1. Validate relation type tồn tại
    # 2. Resolve from/to entity types từ node IDs
    # 3. Validate from_type → to_type allowed cho relation type này
    # 4. Validate edge props theo relation type schema
    # 5. Policy check: action="RELATE"
    # 6. Build query + execute
    # 7. Emit "edge.created" event → trigger ON_WRITE rules
    ...
```

**Done khi:** Edge với from/to type không nằm trong whitelist → 422 với `rule_violated="relation_source_type_not_allowed"`.

---

### Task 3.4 — Context & Query API

**Effort:** 1.5 ngày | **File:** `graph/query/router.py`

| Endpoint | Cache TTL | Guardrails |
|---|---|---|
| `GET /apps/{app_id}/context?entity_id=&depth=` | 5 phút | max_depth=4, max_nodes=500 |
| `GET /apps/{app_id}/impact?entity_id=` | 2 phút | max_depth=4 |
| `GET /apps/{app_id}/coverage?entity_type=&without_relation=` | 10 phút | limit=1000 |
| `GET /apps/{app_id}/path?from=&to=` | 15 phút | max_depth=6, limit=5 paths |
| `POST /apps/{app_id}/query` | No cache | whitelist operations only |
| `POST /apps/{app_id}/query/explain` | No cache | dry-run, no write |

```python
# Guardrails config
GUARDRAILS = {
    "max_depth":        4,
    "max_nodes":        500,
    "max_edges":        1000,
    "min_confidence":   0.5,    # filter edges confidence thấp
    "max_paths":        5,
    "query_timeout_ms": 5000,
}

# Whitelist cho POST /query
ALLOWED_QUERY_TYPES = [
    "FIND_NODES",       # filter by entity_type + properties
    "FIND_PATH",        # shortest path giữa 2 nodes
    "AGGREGATE",        # count, group by
    "FIND_NEIGHBORS",   # neighbors của 1 node
]
```

**Done khi:** `/context` với depth=5 → bị clamp xuống depth=4. Raw Cypher trong `POST /query` → 400.

---

### Task 3.5 — Response View Resolver

**Effort:** 1 ngày | **File:** `views/resolver.py`

```python
class ViewResolver:
    async def resolve(self, app_id: str, view_name: str, raw_node: dict) -> dict:
        view = await get_view(app_id, view_name)
        if not view:
            return raw_node     # fallback: trả raw nếu không có view

        result = {}

        # 1. Field mapping
        for output_field, source_prop in view.field_mapping.items():
            result[output_field] = raw_node.get(source_prop)

        # 2. Embed related nodes
        for rel_config in view.include_relations:
            neighbors = await graph_api.get_neighbors(
                app_id, raw_node["_id"],
                relation_type=rel_config["relation_type"],
                depth=rel_config.get("depth", 1)
            )
            result[rel_config["embed_as"]] = neighbors

        # 3. Exclude fields
        for field in view.exclude_fields:
            result.pop(field, None)

        # 4. Computed fields (safe eval với whitelist functions)
        for field, expr in view.computed_fields.items():
            result[field] = safe_eval(expr, context=result)

        return result
```

**Done khi:** `GET /nodes/{id}?view=requirement_full` trả về fields theo mapping, embed related nodes, không có excluded fields.

---

### 5.1 Definition of Done — Sprint 3

| Hạng mục | Tiêu chí | Test |
|---|---|---|
| Query Planner | 5 builder methods, tất cả output có namespace prefix | Unit test assertions trên Cypher string |
| Node CRUD | Full flow POST/GET/PATCH/DELETE với validation | Integration test |
| Edge CRUD | Relation whitelist enforcement, soft delete | Integration test |
| Namespace Isolation | Node của App A không accessible qua App B endpoint | Security test |
| `/context` | Depth clamp, max_nodes, cache hoạt động | Integration + load test |
| `/impact` | impact_score chính xác với test graph | Unit test với seed |
| `/coverage` | Trả đúng nodes thiếu target relation | Assertion test |
| Response View | field_mapping + embed + exclude hoạt động | Integration test |
| Performance | p95 < 100ms cho CRUD, p95 < 500ms cho `/context` depth=3 | k6 50 VUs |

---

## 6. Sprint 4 — Rule Engine & Access Control (Tuần 7-8)

> **Mục tiêu:** Mỗi app tự quản lý rules và access policies. Rule Engine chạy async, không ảnh hưởng Graph API latency.

---

### Task 4.1 — PostgreSQL Schema: Rules & Policies

**Effort:** 0.5 ngày | **File:** `rules/migrations/003_rules_policies.sql`

```sql
CREATE TABLE kgs_rules (
    rule_id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          VARCHAR(50)  REFERENCES kgs_applications(app_id),
    rule_name       VARCHAR(200) NOT NULL,
    rule_type       VARCHAR(30)  NOT NULL,   -- STRUCTURAL|CONSISTENCY|CONFLICT|ENRICHMENT
    trigger         VARCHAR(30)  NOT NULL,   -- SCHEDULED|ON_WRITE|ON_DEMAND
    schedule_cron   VARCHAR(50),
    trigger_events  TEXT[]       DEFAULT '{}',
    status          VARCHAR(20)  DEFAULT 'ACTIVE',
    priority        INTEGER      DEFAULT 100,
    config          JSONB        NOT NULL,
    action_config   JSONB        NOT NULL,
    last_run_at     TIMESTAMPTZ,
    last_run_result JSONB,
    UNIQUE(app_id, rule_name)
);

CREATE TABLE kgs_rule_executions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id     UUID        REFERENCES kgs_rules(rule_id),
    app_id      VARCHAR(50),
    trigger     VARCHAR(30),
    started_at  TIMESTAMPTZ DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    status      VARCHAR(20),    -- RUNNING|SUCCESS|FAILED|TIMEOUT
    matches     INTEGER,        -- số nodes/edges match pattern
    actions     INTEGER,        -- số actions được thực hiện
    error_log   TEXT
);

CREATE TABLE kgs_policies (
    policy_id   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      VARCHAR(50)  REFERENCES kgs_applications(app_id),
    policy_name VARCHAR(200) NOT NULL,
    effect      VARCHAR(10)  NOT NULL,   -- ALLOW|DENY
    subjects    JSONB        NOT NULL,   -- { roles: [...], scopes: [...] }
    resources   JSONB        NOT NULL,   -- { entity_types: [...] }
    actions     TEXT[]       NOT NULL,
    conditions  JSONB        DEFAULT '{}',
    priority    INTEGER      DEFAULT 100,
    is_active   BOOLEAN      DEFAULT TRUE,
    UNIQUE(app_id, policy_name)
);
```

---

### Task 4.2 — Rule CRUD API

**Effort:** 1.5 ngày | **File:** `rules/router.py`

| Method | Endpoint | Mô tả |
|---|---|---|
| `POST` | `/apps/{app_id}/rules` | Tạo rule, validate config theo rule_type schema |
| `GET` | `/apps/{app_id}/rules` | List với filter type/status/trigger |
| `PATCH` | `/apps/{app_id}/rules/{rule_id}` | Cập nhật, pause/resume |
| `POST` | `/apps/{app_id}/rules/{rule_id}/run` | Chạy on-demand, trả về execution_id |
| `GET` | `/apps/{app_id}/rules/{rule_id}/executions` | List execution history |
| `DELETE` | `/apps/{app_id}/rules/{rule_id}` | Chỉ xóa được khi status=DRAFT |

**Config schema validation theo rule_type:**

```python
RULE_CONFIG_SCHEMAS = {
    "CONSISTENCY": {
        "required": ["check_query"],
        "properties": {
            "check_query": {"type": "object",
                            "required": ["entity_type", "without_relation"]}
        }
    },
    "STRUCTURAL": {
        "required": ["match_pattern", "condition"],
        "properties": {
            "match_pattern": {"type": "string"},
            "condition":     {"type": "string"}
        }
    },
    "CONFLICT": {
        "required": ["conflict_type"],
        "properties": {
            "conflict_type":        {"type": "string", "enum": ["STRUCTURAL","KEYWORD","SEMANTIC"]},
            "semantic_threshold":   {"type": "number", "minimum": 0.7, "maximum": 1.0}
        }
    }
}
```

---

### Task 4.3 — Rule Runner: Scheduled

**Effort:** 1 ngày | **File:** `rules/scheduler.py`

```python
# Dùng APScheduler, mỗi app có cron job riêng
# Distributed lock qua Redis để tránh chạy double khi scale horizontal

class RuleScheduler:
    async def start(self):
        rules = await db.fetch(
            "SELECT * FROM kgs_rules WHERE trigger='SCHEDULED' AND status='ACTIVE'"
        )
        for rule in rules:
            self.scheduler.add_job(
                func=self.run_rule,
                trigger=CronTrigger.from_crontab(rule["schedule_cron"]),
                id=str(rule["rule_id"]),
                args=[rule["rule_id"], rule["app_id"]],
                replace_existing=True
            )

    async def run_rule(self, rule_id: str, app_id: str):
        lock_key = f"rule_lock:{rule_id}"
        async with redis.lock(lock_key, timeout=300):  # 5 phút timeout
            execution = await start_execution(rule_id, app_id, trigger="SCHEDULED")
            try:
                result = await execute_rule(rule_id)
                await complete_execution(execution.id, result)
            except Exception as e:
                await fail_execution(execution.id, str(e))
```

---

### Task 4.4 — Rule Runner: ON_WRITE (Event-Driven)

**Effort:** 1 ngày | **File:** `rules/event_consumer.py`

```python
# Consumer group trên Redis Stream "kgs:graph:events"
class RuleEventConsumer:
    STREAM  = "kgs:graph:events"
    GROUP   = "rule_engine"

    async def consume(self):
        while True:
            messages = await redis.xreadgroup(
                self.GROUP, "worker-1", self.STREAM,
                count=10, block=1000
            )
            for msg in messages:
                event     = parse_event(msg)
                app_id    = event["app_id"]
                event_type = event["type"]   # "node.created", "edge.created"

                # Tìm rules của app match event này
                rules = await db.fetch("""
                    SELECT * FROM kgs_rules
                    WHERE app_id=$1 AND trigger='ON_WRITE'
                      AND $2 = ANY(trigger_events) AND status='ACTIVE'
                    ORDER BY priority ASC
                """, app_id, event_type)

                for rule in rules:
                    asyncio.create_task(execute_rule(rule["rule_id"], context=event))

                await redis.xack(self.STREAM, self.GROUP, msg["id"])
```

---

### Task 4.5 — OPA Integration: Policy Decision Point

**Effort:** 1.5 ngày | **Files:** `access_control/opa_client.py`, `access_control/policy_sync.py`

```python
# access_control/opa_client.py
class PolicyDecisionPoint:
    OPA_URL = "http://localhost:8181"

    async def check(self, app_context: AppContext, resource_type: str,
                     action: str, resource_props: dict = None) -> bool:
        input_data = {
            "app_id":        app_context.app_id,
            "scopes":        app_context.scopes,
            "roles":         app_context.roles,
            "resource_type": resource_type,
            "action":        action,
            "resource":      resource_props or {}
        }
        resp = await httpx.post(
            f"{self.OPA_URL}/v1/data/kgs/allow",
            json={"input": input_data}
        )
        result = resp.json()
        if not result.get("result", False):
            raise AccessDenied(
                policy_name=result.get("denied_by", "unknown"),
                resource_type=resource_type,
                action=action
            )
        return True
```

```rego
# opa/policies/kgs.rego
package kgs

default allow = false

# ALLOW nếu có policy ALLOW khớp và không có DENY nào override
allow {
    some policy in data.policies[input.app_id]
    policy.effect == "ALLOW"
    policy.is_active == true
    subject_matches(policy.subjects)
    resource_matches(policy.resources)
    input.action in policy.actions
    conditions_pass(policy.conditions)

    not deny_overrides
}

deny_overrides {
    some policy in data.policies[input.app_id]
    policy.effect == "DENY"
    policy.is_active == true
    subject_matches(policy.subjects)
    resource_matches(policy.resources)
    input.action in policy.actions
    conditions_pass(policy.conditions)
}
```

```python
# access_control/policy_sync.py
# Sync policies từ Postgres → OPA data bundle mỗi 30 giây
async def sync_policies_to_opa():
    while True:
        policies = await db.fetch("SELECT * FROM kgs_policies WHERE is_active=TRUE")
        bundle   = group_by_app(policies)
        await httpx.put(f"{OPA_URL}/v1/data/policies", json=bundle)
        await asyncio.sleep(30)
```

**Done khi:** Reviewer gọi CREATE node → 403 với `policy_name="reviewer_readonly"`.

---

### Task 4.6 — Policy CRUD API

**Effort:** 1 ngày | **File:** `access_control/router.py`

| Method | Endpoint | Mô tả |
|---|---|---|
| `POST` | `/apps/{app_id}/policies` | Tạo policy |
| `GET` | `/apps/{app_id}/policies` | List policies |
| `PATCH` | `/apps/{app_id}/policies/{policy_id}` | Cập nhật, activate/deactivate |
| `POST` | `/apps/{app_id}/policies/evaluate` | Test: given subject+resource+action → allow/deny + why |
| `DELETE` | `/apps/{app_id}/policies/{policy_id}` | Xóa policy |

```python
# POST /apps/ba_agent/policies/evaluate — cực kỳ hữu ích khi debug
# Request:
{
  "subject": { "roles": ["reviewer"], "scopes": ["graph:read"] },
  "resource": { "entity_type": "Requirement", "properties": { "status": "APPROVED" } },
  "action": "DELETE"
}
# Response:
{
  "decision": "DENY",
  "matched_policy": "deny_delete_approved",
  "reason": "condition matched: status in ['APPROVED','IMPLEMENTED']"
}
```

---

### 6.1 Definition of Done — Sprint 4

| Hạng mục | Tiêu chí | Test |
|---|---|---|
| Rule CRUD | Tạo, list, pause, on-demand run | Integration test |
| Scheduled Rule | CONSISTENCY rule chạy đúng cron, log execution | Integration test |
| ON_WRITE Rule | Tạo edge → rule trigger trong < 2s | End-to-end test |
| OPA Auth | Reviewer CREATE → 403 với policy_name | API test |
| Policy DENY > ALLOW | DENY policy override ALLOW khi conflict | Unit test |
| `/evaluate` | Trả đúng decision + matched_policy | Assertion test |
| Policy Sync | Policy mới tạo → OPA nhận trong < 30s | Integration test |
| INFERRED edges | Rule tạo edge với confidence=0.7, không overwrite manual | Neo4j query |

---

## 7. Onboarding Flow: BA Agent System

Walkthrough đầy đủ để onboard BA Agent lên KGS từ zero.

### Step 1 — Đăng ký Application
```bash
curl -X POST https://kgs.example.com/admin/apps \
  -H "Authorization: Bearer $PLATFORM_ADMIN_KEY" \
  -d '{ "app_id": "ba_agent", "app_name": "BA Agent System", "owner_email": "ba@example.com" }'
```

### Step 2 — Issue API Key
```bash
curl -X POST https://kgs.example.com/admin/apps/ba_agent/keys \
  -H "Authorization: Bearer $PLATFORM_ADMIN_KEY" \
  -d '{ "label": "production", "scopes": ["graph:read","graph:write","ontology:manage","rules:manage"] }'
# → { "key": "kgs_ba_ag_xxxxxxxx" }  ← lưu ngay, chỉ hiện 1 lần
```

### Step 3 — Khai báo Ontology
```bash
export KGS_KEY="kgs_ba_ag_xxxxxxxx"

# Entity types
curl -X POST .../apps/ba_agent/ontology/entity-types -H "Authorization: Bearer $KGS_KEY" \
  -d '{ "type_name": "Requirement", "id_property": "req_id", "required_props": [...], "properties": {...} }'

# Lặp lại cho: UseCase, API, Component, TestCase

# Relation types
curl -X POST .../apps/ba_agent/ontology/relation-types \
  -d '{ "type_name": "HAS_USECASE", "from_types": ["Requirement"], "to_types": ["UseCase"], ... }'

# Lặp lại cho: USES_API, IMPLEMENTED_BY, VERIFIED_BY, COVERED_BY
```

### Step 4 — Đăng ký Rules
```bash
# Rule: detect missing test coverage
curl -X POST .../apps/ba_agent/rules \
  -d '{
    "rule_name": "missing_test_coverage",
    "rule_type": "CONSISTENCY",
    "trigger": "SCHEDULED",
    "schedule_cron": "0 */6 * * *",
    "config": { "check_query": { "entity_type": "Requirement", "status": "APPROVED", "without_relation": "VERIFIED_BY" } },
    "action_config": { "action": "CREATE_FLAG", "flag_type": "MISSING_COVERAGE", "severity": "WARNING" }
  }'
```

### Step 5 — Đăng ký Policies
```bash
# Service có full write
curl -X POST .../apps/ba_agent/policies \
  -d '{ "policy_name": "service_write_all", "effect": "ALLOW", "subjects": {"roles":["ba_agent_service"]}, "resources": {"entity_types":["*"]}, "actions": ["CREATE","READ","UPDATE","RELATE"] }'

# Reviewer chỉ đọc
curl -X POST .../apps/ba_agent/policies \
  -d '{ "policy_name": "reviewer_readonly", "effect": "ALLOW", "subjects": {"roles":["reviewer"]}, "resources": {"entity_types":["Requirement","UseCase"]}, "actions": ["READ"] }'

# Không ai xóa Requirement đã APPROVED
curl -X POST .../apps/ba_agent/policies \
  -d '{ "policy_name": "deny_delete_approved", "effect": "DENY", "subjects": {"roles":["*"]}, "resources": {"entity_types":["Requirement"]}, "actions": ["DELETE"], "conditions": {"property_filters": {"status": ["APPROVED","IMPLEMENTED"]}} }'
```

### Step 6 — Verify & Test
```bash
# Kiểm tra ontology
curl .../apps/ba_agent/ontology

# Test policy
curl -X POST .../apps/ba_agent/policies/evaluate \
  -d '{ "subject": {"roles":["reviewer"]}, "resource": {"entity_type":"Requirement"}, "action": "DELETE" }'
# → { "decision": "DENY", "matched_policy": "deny_delete_approved" }

# Health check
curl .../health/apps/ba_agent
```

### Step 7 — Sử dụng Graph API
```bash
# Tạo node
curl -X POST .../apps/ba_agent/nodes \
  -d '{ "entity_type": "Requirement", "properties": { "req_id": "REQ-AUTH-001", "title": "Login", ... } }'

# Tạo edge
curl -X POST .../apps/ba_agent/edges \
  -d '{ "relation_type": "HAS_USECASE", "from_node_id": "REQ-AUTH-001", "to_node_id": "UC-001", "properties": { "confidence": 1.0 } }'

# Query context
curl ".../apps/ba_agent/context?entity_id=REQ-AUTH-001&depth=2"
```

---

## 8. Dependency Map

```
Task 1.1 (PG Schema: Registry)
    └─► Task 1.2 (App CRUD API)
    └─► Task 1.5 (Namespace Reservation)

Task 1.2 (App CRUD API)
    └─► Task 1.3 (Gateway Auth Middleware)  ← BLOCKER cho mọi thứ còn lại

Task 1.3 (Gateway Auth)
    └─► Task 1.4 (Rate Limiter)
    └─► Task 2.2 (Ontology CRUD API)        ← Sprint 2 bắt đầu được
    └─► Task 3.2 (Node CRUD)                ← Sprint 3 bắt đầu được

Task 2.1 (PG Schema: Ontology)
    └─► Task 2.2 (Ontology CRUD API)
    └─► Task 2.3 (JSON Schema Validator)

Task 2.2 (Ontology CRUD API)
    └─► Task 2.4 (Ontology Cache)
    └─► Task 2.5 (Neo4j Constraint Sync)
    └─► Task 3.1 (Query Planner)            ← cần ontology để resolve id_property

Task 3.1 (Query Planner)
    └─► Task 3.2 (Node CRUD)
    └─► Task 3.3 (Edge CRUD)
    └─► Task 3.4 (Context & Query API)

Task 3.3 (Edge CRUD)
    └─► Task 4.4 (Rule Runner ON_WRITE)     ← emit edge.created event

Task 4.1 (PG Schema: Rules & Policies)
    └─► Task 4.2 (Rule CRUD API)
    └─► Task 4.6 (Policy CRUD API)

Task 4.3 (Rule Scheduler) — parallel với Task 4.5 (OPA)
Task 4.4 (Event Consumer)  — parallel với Task 4.5 (OPA)
```

**Critical path:** 1.3 → 2.2 → 2.3 → 3.1 → 3.2 → 3.3

---

## 9. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| APOC unavailable trên managed Neo4j | Medium | High | Viết fallback manual multi-hop MATCH đến depth 4; test cả 2 paths |
| Namespace collision: hai app đặt cùng `app_id` | Low | Critical | `app_id` là PRIMARY KEY trong PG; Reserve step atomic với Neo4j `MERGE` |
| OPA bundle sync chậm → stale policies | Medium | Medium | Reduce sync interval xuống 10s; force sync khi policy thay đổi |
| Ontology cache stale khi update | Low | Medium | Invalidate cache ngay khi POST/PUT ontology thành công |
| Rule engine chạy double khi scale horizontal | Medium | Medium | Redis distributed lock per rule_id, TTL = 2x expected runtime |
| App vượt quota node nhưng bypass qua batch | Medium | Medium | Quota check trong batch là atomic: kiểm tra tổng trước khi insert |
| Query Planner inject sai namespace | Low | Critical | Unit test cho mọi builder method asserting namespace prefix trong output |
| Edge relation type thay đổi sau khi có data | Low | High | Disallow `from_types`/`to_types` change khi có edges dùng relation type đó |
| OPA Rego bug cho `DENY > ALLOW` logic | Medium | High | Test suite cho tất cả policy combinations: allow-only, deny-only, conflict |

---

## 10. Pre-Implementation Checklist

### Infrastructure

- [ ] Neo4j 5.x running — APOC plugin installed (`CALL apoc.help('') YIELD name` để verify)
- [ ] PostgreSQL 16 running — `pg_crypto` extension enabled (cho `gen_random_uuid()`)
- [ ] Redis 7 running — persistence AOF enabled, `maxmemory-policy=allkeys-lru`
- [ ] OPA 0.60+ running as sidecar — `http://localhost:8181` reachable
- [ ] Python 3.11+ venv với dependencies installed

### Python Dependencies

```bash
pip install fastapi uvicorn[standard] asyncpg redis[hiredis] neo4j \
            opa-python-client httpx jsonschema bcrypt apscheduler \
            pydantic python-dotenv alembic pytest pytest-asyncio
```

### Environment Variables

```bash
# .env — KHÔNG commit

# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_neo4j_password
NEO4J_DATABASE=kgs

# PostgreSQL
POSTGRES_DSN=postgresql+asyncpg://kgs_user:password@localhost:5432/kgs_platform

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your_redis_password

# OPA
OPA_URL=http://localhost:8181

# Platform
PLATFORM_ADMIN_KEY=padmin_xxxxxxxxxxxxxxxx   # dùng cho /admin/* endpoints
APP_ENV=development
LOG_LEVEL=DEBUG

# Quota defaults
DEFAULT_MAX_NODES=1000000
DEFAULT_MAX_RPM=1000
```

### Database Setup

```bash
# 1. Postgres migrations
alembic upgrade head

# 2. Neo4j initial setup
python -c "from kgs.storage.neo4j import create_platform_indexes; create_platform_indexes()"

# 3. OPA bundle khởi tạo
curl -X PUT http://localhost:8181/v1/data/policies -d '{}'
```

### Smoke Test Sequence

```bash
# Start server
uvicorn kgs.main:app --reload --port 8000

# 1. Tạo app
curl -X POST localhost:8000/admin/apps \
  -H "Authorization: Bearer $PLATFORM_ADMIN_KEY" \
  -d '{ "app_id": "test_app", "app_name": "Test", "owner_email": "test@example.com" }'

# 2. Issue key
curl -X POST localhost:8000/admin/apps/test_app/keys \
  -H "Authorization: Bearer $PLATFORM_ADMIN_KEY" \
  -d '{ "label": "test", "scopes": ["graph:read","graph:write","ontology:manage"] }'

# 3. Tạo entity type
curl -X POST localhost:8000/apps/test_app/ontology/entity-types \
  -H "Authorization: Bearer kgs_test_xxxx" \
  -d '{ "type_name": "Item", "id_property": "item_id", "required_props": ["item_id","name"], "properties": { "item_id": {"type":"string"}, "name": {"type":"string"} } }'

# 4. Tạo node
curl -X POST localhost:8000/apps/test_app/nodes \
  -H "Authorization: Bearer kgs_test_xxxx" \
  -d '{ "entity_type": "Item", "properties": { "item_id": "I-001", "name": "Test Item" } }'

# 5. Verify namespace trong Neo4j
# MATCH (n:test_app__Item) RETURN n  → phải thấy node
```

### Definition of Done — Full Platform (Sprint 1-4)

| Milestone | Tiêu chí | Test |
|---|---|---|
| Full onboarding | App đăng ký → API Key → Ontology → Graph CRUD < 30 phút | E2E walkthrough |
| Namespace isolation | App A không query được data App B dù bruteforce | Security test suite |
| Schema enforcement | Node sai field → 422 với error rõ ràng | 20 negative test cases |
| Rule automation | CONSISTENCY rule chạy cron, log đầy đủ | Integration test |
| Access control | DENY policy override ALLOW, error có policy_name | Policy test matrix |
| Performance | Graph API p95 < 100ms CRUD, p95 < 500ms `/context` depth=3 | k6 50 VUs, 5 phút |
| Audit trail | 100% requests có entry trong kgs_audit_log | SQL count vs access log |

---

*KGS Platform Implementation Plan v1.0 — Generated từ KGS Platform Architecture Specification v1.0*  
*BA Agent System là reference tenant đầu tiên — mọi tenant sau build theo cùng onboarding flow này.*
