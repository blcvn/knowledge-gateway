# Knowledge Graph & Cognitive Layer — Implementation Plan

> **Dự án:** URD/SRS chuẩn IEEE · Multi-Agent BA · Traceability Mapping  
> **Version:** 1.0  
> **Tổng thời gian:** 8 tuần (4 sprints × 2 tuần)  
> **Tech Stack:** Neo4j 5.x · PostgreSQL 16 · Redis 7 · Qdrant · FastAPI · LangGraph

---

## Mục Lục

1. [Tổng Quan & Nguyên Tắc](#1-tổng-quan--nguyên-tắc)
2. [Kiến Trúc Tổng Thể](#2-kiến-trúc-tổng-thể)
3. [Sprint 1 — Foundation](#3-sprint-1--foundation-tuần-1-2)
4. [Sprint 2 — Agent Memory](#4-sprint-2--agent-memory-tuần-3-4)
5. [Sprint 3 — Reasoning & Partial Regenerate](#5-sprint-3--reasoning--partial-regenerate-tuần-5-6)
6. [Sprint 4 — Production Hardening](#6-sprint-4--production-hardening-tuần-7-8)
7. [Dependency Map](#7-dependency-map)
8. [Risk Register](#8-risk-register)
9. [Pre-Implementation Checklist](#9-pre-implementation-checklist)

---

## 1. Tổng Quan & Nguyên Tắc

### Mục tiêu cuối cùng

```
PRD → URD → SRS → API → Test → Code
```

Hệ thống phải đảm bảo **4 khả năng cốt lõi**:

| Khả năng | Mô tả | Phụ thuộc vào |
|---|---|---|
| **Trace** | Đi từ Requirement bất kỳ đến Code/Test | Traceability Graph |
| **Impact Analysis** | Khi R thay đổi → biết gì bị ảnh hưởng | Graph traversal + edge weights |
| **Coverage Analysis** | Requirement nào chưa có Test | Cypher query pattern |
| **Explainability** | Giải thích tại sao agent ra quyết định | Reasoning path API |

### Nguyên tắc xuyên suốt

- **Vertical slice trước, horizontal mở rộng sau** — Sprint 1 phải có 1 trace path hoàn chỉnh end-to-end trước khi làm tính năng phụ
- **Không xóa data, chỉ deprecate** — dùng `valid_to`, `status = DEPRECATED` để giữ audit trail
- **Agent không query raw graph** — mọi truy vấn phải đi qua Context Builder API
- **Inferred ≠ Manual** — relations do Rule Engine tạo ra phải có `relation_type = INFERRED_DEPENDENCY`, không overwrite manual relations
- **Postgres là source of truth** — Neo4j, Qdrant, Redis là derived stores, đồng bộ qua event

---

## 2. Kiến Trúc Tổng Thể

```
┌─────────────────────────────────────────────────────┐
│                  Agent Orchestrator                  │
│              (LangGraph — Python)                    │
└──────────────────────┬──────────────────────────────┘
                       │  Chỉ gọi qua API
┌──────────────────────▼──────────────────────────────┐
│              Context Builder Service                 │
│                   (FastAPI)                          │
│  /context  /impact  /coverage  /reasoning-path       │
│  /validate  /conflicts  /dirty/propagate             │
└──────────┬──────────────────────────┬───────────────┘
           │                          │
┌──────────▼──────────┐  ┌────────────▼──────────────┐
│   Cognitive Layer   │  │      Rule Engine          │
│                     │  │  (Async, event-triggered) │
│  • Traceability     │  │  • Structural rules       │
│    Graph            │  │  • Consistency check      │
│  • Memory Graph     │  │  • Conflict detection     │
│  • Reasoning        │  └────────────┬──────────────┘
└──────────┬──────────┘               │
           │                          │
┌──────────▼──────────────────────────▼───────────────┐
│                  Storage Layer                       │
│                                                      │
│  Neo4j 5.x      PostgreSQL 16    Redis 7   Qdrant   │
│  (Graph)        (Source of Truth) (Cache)  (Vector) │
└──────────────────────────────────────────────────────┘
                           │
              ┌────────────▼───────────┐
              │   Event Bus            │
              │   (Redis Streams)      │
              └────────────────────────┘
```

### Module Boundaries

```
src/
├── graph/          # Neo4j queries, schema, migrations
├── memory/         # Memory Graph, compression, Redis
├── reasoning/      # Rule Engine, conflict detection
├── context_api/    # FastAPI — Context Builder Service
├── agent/          # LangGraph agent loops
├── sync/           # Outbox pattern, event consumers
└── seed/           # Test data generators
```

---

## 3. Sprint 1 — Foundation (Tuần 1-2)

> **Mục tiêu:** Có 1 trace path hoàn chỉnh chạy được. Mọi thứ khác build trên nền này.

### 3.1 Tasks

#### Task 1.1 — Neo4j Infrastructure Setup
**Effort:** 1 ngày | **Owner:** Backend Lead

```cypher
-- Chạy file: graph/migrations/001_initial_schema.cypher

CREATE CONSTRAINT req_id_unique  IF NOT EXISTS FOR (r:Requirement) REQUIRE r.req_id  IS UNIQUE;
CREATE CONSTRAINT uc_id_unique   IF NOT EXISTS FOR (u:UseCase)     REQUIRE u.uc_id   IS UNIQUE;
CREATE CONSTRAINT api_id_unique  IF NOT EXISTS FOR (a:API)         REQUIRE a.api_id  IS UNIQUE;
CREATE CONSTRAINT tc_id_unique   IF NOT EXISTS FOR (t:TestCase)    REQUIRE t.tc_id   IS UNIQUE;
CREATE CONSTRAINT comp_id_unique IF NOT EXISTS FOR (c:Component)   REQUIRE c.comp_id IS UNIQUE;

CREATE INDEX req_status_idx  IF NOT EXISTS FOR (r:Requirement) ON (r.status);
CREATE INDEX req_version_idx IF NOT EXISTS FOR (r:Requirement) ON (r.version);
CREATE INDEX tc_status_idx   IF NOT EXISTS FOR (t:TestCase)    ON (t.status, t.type);
CREATE INDEX api_service_idx IF NOT EXISTS FOR (a:API)         ON (a.service);
```

**Done khi:** `neo4j-admin validate` không có lỗi.

---

#### Task 1.2 — PostgreSQL Schema
**Effort:** 1 ngày | **Owner:** Backend Lead

```sql
-- graph/migrations/001_pg_schema.sql

-- Bảng audit log cho tất cả thay đổi
CREATE TABLE entity_audit_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type  VARCHAR(50)  NOT NULL,
    entity_id    VARCHAR(100) NOT NULL,
    action       VARCHAR(20)  NOT NULL,  -- CREATED | UPDATED | DELETED | DEPRECATED
    changed_by   VARCHAR(100) NOT NULL,
    old_data     JSONB,
    new_data     JSONB,
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);

-- Bảng outbox cho event-driven sync
CREATE TABLE event_outbox (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type  VARCHAR(50)  NOT NULL,
    entity_id    VARCHAR(100) NOT NULL,
    event_type   VARCHAR(50)  NOT NULL,
    payload      JSONB        NOT NULL,
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    processed    BOOLEAN      DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    retry_count  INTEGER      DEFAULT 0
);

CREATE INDEX outbox_unprocessed ON event_outbox(processed, created_at) WHERE NOT processed;

-- Bảng requirements metadata (source of truth)
CREATE TABLE requirements (
    req_id       VARCHAR(100) PRIMARY KEY,
    title        TEXT         NOT NULL,
    description  TEXT         NOT NULL,
    type         VARCHAR(30)  NOT NULL,
    priority     VARCHAR(10)  NOT NULL,
    status       VARCHAR(20)  NOT NULL,
    version      VARCHAR(20)  NOT NULL,
    source_doc   VARCHAR(100),
    author       VARCHAR(100),
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NOW()
);
```

**Done khi:** Alembic migration chạy thành công, tables tồn tại.

---

#### Task 1.3 — Seed Data Generator
**Effort:** 1 ngày | **Owner:** QA Engineer

Tạo file `seed/seed_data.py`:

```python
# Yêu cầu tối thiểu cho seed data:
# - 50 Requirement (35 FUNCTIONAL, 15 NON_FUNCTIONAL)
# - 20 UseCase
# - 30 API
# - 10 Component
# - 60 TestCase
# - Relations với confidence đa dạng: 0.5, 0.7, 0.9, 1.0
# - 5 requirement KHÔNG có TestCase  → để test coverage detection
# - 2 cặp requirement conflict       → để test conflict detection
# - Mix status: DRAFT, APPROVED, IMPLEMENTED, VERIFIED
```

**Done khi:** Query coverage detection trả về đúng 5 requirements thiếu test.

---

#### Task 1.4 — Vertical Slice: Full Trace Path
**Effort:** 2 ngày | **Owner:** Backend Dev

Implement module `graph/traceability.py` với các functions:

| Function | Input | Output |
|---|---|---|
| `create_requirement(data)` | RequirementDTO | node_id |
| `create_trace_chain(req_id, uc, api, comp, tc)` | IDs | bool |
| `get_impact_analysis(req_id)` | req_id | ImpactResult |
| `get_coverage_report(release)` | version | CoverageReport |
| `get_reasoning_path(from_id, to_id)` | 2 node IDs | PathResult |

**Cypher mẫu cho impact analysis:**

```cypher
MATCH (r:Requirement { req_id: $req_id })
CALL apoc.path.subgraphAll(r, {
    maxLevel: 4,
    relationshipFilter: 'REQUIREMENT_HAS_USECASE>|USECASE_HAS_API>|API_IMPLEMENTED_BY_COMPONENT>|API_COVERED_BY_TEST>'
}) YIELD nodes, relationships
WITH r, nodes, relationships,
     [rel IN relationships | rel.impact_weight * rel.confidence] AS weights
RETURN
    r.req_id                                                          AS requirement,
    [n IN nodes WHERE 'UseCase'   IN labels(n) | n.uc_id]            AS affected_usecases,
    [n IN nodes WHERE 'API'       IN labels(n) | n.api_id]           AS affected_apis,
    [n IN nodes WHERE 'Component' IN labels(n) | n.comp_id]          AS affected_components,
    [n IN nodes WHERE 'TestCase'  IN labels(n) | n.tc_id]            AS affected_tests,
    round(reduce(s=0.0, w IN weights | s + w) / size(weights), 2)    AS impact_score
```

**Done khi:** Unit test tạo trace chain REQ-001 → UC-001 → API-001 → COMP-001 → TC-001, query impact analysis trả về đúng kết quả.

---

#### Task 1.5 — Context Builder API (Skeleton)
**Effort:** 2 ngày | **Owner:** Backend Dev

```
context_api/
├── main.py          # FastAPI app, middleware, error handlers
├── routers/
│   ├── context.py   # GET /context
│   └── coverage.py  # GET /coverage
├── guardrails.py    # Config: max_depth=4, max_nodes=500
└── cache.py         # Redis cache wrapper, TTL config
```

**Endpoints Sprint 1:**

- `GET /context?entity={id}&depth={1-4}` — trả về ranked subgraph
- `GET /coverage?release={version}` — trả về coverage report
- `GET /health/graph` — node count, edge count, latency

**Response format chuẩn:**

```json
{
  "entity": { "id": "REQ-AUTH-001", "type": "Requirement" },
  "subgraph": {
    "nodes": [{ "id": "UC-001", "type": "UseCase", "relevance_score": 0.95 }],
    "edges": [{ "from": "REQ-AUTH-001", "to": "UC-001", "type": "REQUIREMENT_HAS_USECASE", "confidence": 1.0 }]
  },
  "meta": { "depth": 2, "node_count": 5, "query_time_ms": 12, "cached": true }
}
```

**Done khi:** `/context?entity=REQ-001&depth=2` phản hồi < 100ms, `/coverage` trả đúng coverage rate với seed data.

---

### 3.2 Definition of Done — Sprint 1

| Hạng mục | Tiêu chí | Kiểm tra bằng |
|---|---|---|
| Graph Schema | Tất cả constraints và indexes tạo thành công | `neo4j-admin validate` |
| Vertical Slice | Tạo trace path đầy đủ REQ→UC→API→COMP→TC | Cypher query thủ công |
| Impact Analysis | `impact_score` chính xác với seed data | Unit test Python |
| Coverage Query | Detect đúng 5 requirements không có test | Assert với seed data |
| `/context` API | Response < 100ms, depth=2, 1000 nodes | Load test k6 |
| `/coverage` API | Coverage rate chính xác ± 0.1% | Assert với seed data |
| Seed Data | 50 req, 20 UC, 30 API, 60 TC trong graph | Count query Neo4j |

---

## 4. Sprint 2 — Agent Memory (Tuần 3-4)

> **Mục tiêu:** Agent có thể chạy loop, ghi nhớ context, compress memory sau mỗi session.

### 4.1 Tasks

#### Task 2.1 — Redis Short-Term Memory
**Effort:** 1 ngày | **Owner:** Backend Dev

```python
# memory/redis_memory.py

REDIS_SCHEMA = {
    # Key pattern: session:{session_id}:context
    # Type: Redis Hash
    # TTL: 1800s (30 phút)
    "context_snapshot": {
        "key": "session:{session_id}:context",
        "fields": ["working_entities", "state_summary", "loop_index", "plan_steps"],
        "ttl": 1800
    },
    # Key pattern: session:{session_id}:plan
    # Type: Redis List
    "plan_steps": {
        "key": "session:{session_id}:plan",
        "ttl": 1800
    }
}
```

---

#### Task 2.2 — LangGraph Agent Loop
**Effort:** 3 ngày | **Owner:** AI Engineer

```
State → Action → Tool → Observation → Update State → (loop)
```

```python
# agent/ba_agent.py — State machine

class AgentState(TypedDict):
    session_id:   str
    task:         str
    loop_index:   int
    thoughts:     list[str]
    tool_results: list[dict]
    context:      dict        # từ Context Builder API
    plan:         list[str]
    done:         bool

# Tools available to agent:
# - get_context(entity_id, depth)   → gọi /context API
# - get_impact(req_id)              → gọi /impact API
# - get_coverage()                  → gọi /coverage API
# - create_requirement(data)        → gọi graph/traceability.py
# - update_requirement(req_id, data)
# - create_relation(from_id, to_id, rel_type)
```

**Quy tắc quan trọng:**
- Agent **không được** gọi Neo4j driver trực tiếp
- Mọi write phải qua validation layer trước khi commit
- Sau 10 vòng lặp phải trigger memory compression

---

#### Task 2.3 — Memory Compression Pipeline
**Effort:** 2 ngày | **Owner:** AI Engineer

```python
# memory/compression.py

COMPRESS_PROMPT = """
Tóm tắt các thought sau thành 1 đoạn không quá 200 tokens.
Giữ lại: decisions đã đưa ra, entities đã xử lý, kết quả quan trọng.
Bỏ đi: các bước lặp lại, failed attempts, internal reasoning trung gian.

Thoughts:
{thoughts}

Output: JSON {{ "summary": "...", "key_decisions": [...], "entities_processed": [...] }}
"""

async def compress_memory(session_id: str, loop_index: int):
    # 1. Lấy thoughts từ Neo4j (vòng lặp <= loop_index)
    # 2. Gọi LLM với COMPRESS_PROMPT
    # 3. Tạo KnowledgeEntity trong Neo4j
    # 4. Embed summary vào Qdrant
    # 5. Xóa thoughts cũ (giữ lại 5 gần nhất)
    # 6. Update AgentSession.token_count
```

**Trigger compression khi:**
- `loop_index % 10 == 0`
- `token_count > 80_000`
- Session kết thúc (status = COMPLETED | FAILED)

---

#### Task 2.4 — AgentSession Audit Log
**Effort:** 1 ngày | **Owner:** Backend Dev

```sql
-- Thêm vào PostgreSQL

CREATE TABLE agent_sessions (
    session_id    UUID PRIMARY KEY,
    agent_type    VARCHAR(30)  NOT NULL,
    task_desc     TEXT,
    status        VARCHAR(20)  NOT NULL DEFAULT 'RUNNING',
    started_at    TIMESTAMPTZ  DEFAULT NOW(),
    ended_at      TIMESTAMPTZ,
    token_count   INTEGER      DEFAULT 0,
    loop_count    INTEGER      DEFAULT 0,
    error_log     TEXT
);

CREATE TABLE agent_decisions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID REFERENCES agent_sessions,
    loop_index    INTEGER,
    decision_type VARCHAR(50),  -- CREATE_RELATION | DEPRECATE_REQ | GENERATE_USECASE
    target_id     VARCHAR(100),
    rationale     TEXT,
    created_at    TIMESTAMPTZ  DEFAULT NOW()
);
```

---

#### Task 2.5 — Qdrant Setup & Embedding Pipeline
**Effort:** 1 ngày | **Owner:** Backend Dev

```python
# memory/vector_store.py

# Collection config
QDRANT_COLLECTIONS = {
    "knowledge": {
        "vector_size": 1536,          # OpenAI text-embedding-3-small
        "distance": "Cosine",
        "payload_schema": {
            "ke_id": "keyword",
            "entity_type": "keyword", # Requirement | KnowledgeEntity
            "entity_id": "keyword",
            "version": "keyword"
        }
    }
}

async def embed_requirement(req: Requirement):
    text = f"{req.title}\n{req.description}\n{' '.join(req.acceptance_criteria or [])}"
    vector = await openai.embed(text)
    await qdrant.upsert("knowledge", vector, payload={"entity_type": "Requirement", "entity_id": req.req_id})
```

---

### 4.2 Definition of Done — Sprint 2

| Hạng mục | Tiêu chí | Kiểm tra bằng |
|---|---|---|
| Redis Memory | ContextSnapshot persist sau page refresh | Manual test |
| Agent Loop | Chạy 20 vòng lặp không crash, ghi log đầy đủ | Integration test |
| Compression | KnowledgeEntity được tạo, thoughts cũ bị xóa | Neo4j query |
| Qdrant | Embedding của 50 requirements đã index | Qdrant dashboard |
| Audit Log | AgentSession lưu đầy đủ vào Postgres | SQL query |
| Token Budget | Agent tự compress khi loop_index % 10 == 0 | Log trace |

---

## 5. Sprint 3 — Reasoning & Partial Regenerate (Tuần 5-6)

> **Mục tiêu:** Hệ thống tự phát hiện vấn đề và xử lý thay đổi có kiểm soát.

### 5.1 Tasks

#### Task 3.1 — Rule Engine Core
**Effort:** 2 ngày | **Owner:** Backend Dev

```python
# reasoning/rule_engine.py

class RuleEngine:
    def __init__(self, graph, pg, qdrant):
        self.rules = RULES_REGISTRY   # load từ rules_registry.py
        self.graph = graph
        self.pg    = pg
        self.qdrant = qdrant

    async def run_all(self, trigger: str = "scheduled"):
        results = []
        for rule in self.rules:
            if rule["type"] == "STRUCTURAL":
                result = await self._run_structural(rule)
            elif rule["type"] == "CONSISTENCY":
                result = await self._run_consistency(rule)
            elif rule["type"] == "CONFLICT":
                result = await self._run_conflict(rule)
            results.append(result)
        await self._persist_results(results)
        return results
```

**Rules cần implement Sprint 3:**

| Rule ID | Type | Mô tả | Action |
|---|---|---|---|
| `STRUCT-001` | STRUCTURAL | Requirement → UseCase → API → tạo INFERRED_DEPENDENCY | CREATE relation |
| `CONSIST-001` | CONSISTENCY | Requirement APPROVED chưa có TestCase | CREATE flag |
| `CONSIST-002` | CONSISTENCY | API APPROVED chưa có TestCase | CREATE flag |
| `CONFLICT-001` | CONFLICT | Structural: 2 req cùng release, block cùng component | RAISE alert |
| `CONFLICT-002` | CONFLICT | Rule-based: keyword contradictory | RAISE alert |

---

#### Task 3.2 — Conflict Detection: Structural & Rule-Based
**Effort:** 2 ngày | **Owner:** Backend Dev

**Tầng 1 — Structural (Cypher):**

```cypher
// Tìm conflicts structural: 2 req cùng release → cùng component
MATCH (r1:Requirement)-[*..3]->(c:Component)<-[*..3]-(r2:Requirement)
WHERE r1.req_id < r2.req_id          -- tránh duplicate pair
  AND r1.version = r2.version         -- cùng release
  AND r1.priority IN ['MUST','SHOULD']
  AND r2.priority IN ['MUST','SHOULD']
RETURN r1.req_id, r2.req_id, c.comp_id AS shared_component
```

**Tầng 2 — Rule-based (Keyword):**

```python
CONTRADICTION_PAIRS = [
    ("không được", "phải"),
    ("cấm", "bắt buộc"),
    ("tối đa", "tối thiểu"),
    ("vô hiệu hóa", "cho phép"),
]

def check_keyword_conflict(req1: str, req2: str) -> bool:
    for (neg, pos) in CONTRADICTION_PAIRS:
        if neg in req1 and pos in req2: return True
        if neg in req2 and pos in req1: return True
    return False
```

**Tầng 3 — Semantic (Qdrant):** ← implement Sprint 4

---

#### Task 3.3 — Dirty-Flag Propagation
**Effort:** 1 ngày | **Owner:** Backend Dev

```cypher
// Khi Requirement thay đổi: mark dirty và propagate downstream
MATCH (r:Requirement { req_id: $req_id })
SET r.is_dirty = true,
    r.dirty_since = datetime(),
    r.dirty_reason = $change_summary
WITH r
MATCH (r)-[*1..4]->(downstream)
WHERE 'UseCase'   IN labels(downstream)
   OR 'API'       IN labels(downstream)
   OR 'TestCase'  IN labels(downstream)
   OR 'Component' IN labels(downstream)
SET downstream.is_dirty     = true,
    downstream.dirty_from_req = $req_id,
    downstream.dirty_since  = datetime()
RETURN count(downstream) AS marked_dirty
```

---

#### Task 3.4 — Redis Streams Task Queue
**Effort:** 1 ngày | **Owner:** Backend Dev

```python
# sync/task_queue.py

TASK_STREAM = "agent:tasks"

async def publish_task(task: dict):
    """Push task vào Redis Stream"""
    await redis.xadd(TASK_STREAM, {
        "task_id":     str(uuid4()),
        "agent_type":  task["agent_type"],   # BA_AGENT | REVIEW_AGENT
        "entity_type": task["entity_type"],
        "entity_id":   task["entity_id"],
        "action":      task["action"],       # REGENERATE | REVIEW | VALIDATE_COVERAGE
        "priority":    task.get("priority", "NORMAL"),
        "context":     json.dumps(task.get("context", {})),
        "created_at":  datetime.utcnow().isoformat()
    })

async def consume_tasks(consumer_group: str, agent_type: str):
    """Consumer loop cho mỗi agent type"""
    while True:
        messages = await redis.xreadgroup(consumer_group, agent_type, TASK_STREAM, count=5)
        for task in messages:
            await process_task(task)
            await redis.xack(TASK_STREAM, consumer_group, task["id"])
```

---

#### Task 3.5 — Explainability API
**Effort:** 1 ngày | **Owner:** Backend Dev

Thêm endpoint `GET /reasoning-path?from={id}&to={id}`:

```cypher
MATCH path = (start { req_id: $from_id })-[*..6]->(end { tc_id: $to_id })
WITH path, [rel IN relationships(path) | {
    type:       type(rel),
    source:     rel.source,
    confidence: rel.confidence,
    created_by: rel.created_by,
    created_at: rel.created_at
}] AS edge_info
RETURN
    [n IN nodes(path) | {
        label: labels(n)[0],
        id:    coalesce(n.req_id, n.uc_id, n.api_id, n.comp_id, n.tc_id),
        name:  coalesce(n.title, n.name, n.endpoint)
    }] AS node_chain,
    edge_info,
    length(path) AS depth
ORDER BY depth ASC LIMIT 5
```

---

### 5.2 Definition of Done — Sprint 3

| Hạng mục | Tiêu chí | Kiểm tra bằng |
|---|---|---|
| Rule Engine | STRUCT-001, CONSIST-001, CONSIST-002 chạy đúng | Unit test với seed data |
| INFERRED relations | Tạo đúng, confidence = 0.7, không overwrite manual | Cypher query |
| Conflict Detection | Phát hiện 2 cặp conflict trong seed data | Assert result |
| Dirty Propagation | Update REQ-001 → đúng N nodes bị mark dirty | Count assertion |
| Task Queue | Message được consume, agent nhận task | Integration test |
| `/reasoning-path` | Trả về path chain từ REQ đến TC | Manual test |

---

## 6. Sprint 4 — Production Hardening (Tuần 7-8)

> **Mục tiêu:** Hệ thống sẵn sàng cho production: sync đáng tin cậy, temporal query, semantic conflict, load tested.

### 6.1 Tasks

#### Task 4.1 — Outbox Pattern & Event Consumers
**Effort:** 2 ngày | **Owner:** Backend Dev

```python
# sync/outbox_processor.py

async def process_outbox():
    """Chạy mỗi 5 giây — polling loop"""
    while True:
        events = await pg.fetch("""
            SELECT * FROM event_outbox
            WHERE processed = FALSE
            ORDER BY created_at ASC
            LIMIT 50
            FOR UPDATE SKIP LOCKED
        """)

        for event in events:
            try:
                if event["entity_type"] == "Requirement":
                    await sync_requirement_to_neo4j(event["payload"])
                    await embed_requirement_to_qdrant(event["payload"])
                    await invalidate_cache(event["entity_id"])

                await pg.execute("""
                    UPDATE event_outbox
                    SET processed = TRUE, processed_at = NOW()
                    WHERE id = $1
                """, event["id"])

            except Exception as e:
                await pg.execute("""
                    UPDATE event_outbox
                    SET retry_count = retry_count + 1
                    WHERE id = $1
                """, event["id"])
                log.error(f"Outbox process failed: {e}")

        await asyncio.sleep(5)
```

---

#### Task 4.2 — Temporal Graph
**Effort:** 1 ngày | **Owner:** Backend Dev

```python
# graph/temporal.py

async def create_versioned_relation(from_id: str, to_id: str, rel_type: str,
                                     release_version: str, props: dict):
    """Tạo relation với temporal metadata"""
    await graph.query(f"""
        MATCH (a {{ {id_field(from_id)}: $from_id }})
        MATCH (b {{ {id_field(to_id)}: $to_id }})
        CREATE (a)-[r:{rel_type} {{
            confidence:   $confidence,
            source:       $source,
            version:      $version,
            created_by:   $created_by,
            created_at:   datetime(),
            valid_from:   datetime($valid_from),
            valid_to:     NULL,
            impact_weight: $impact_weight
        }}]->(b)
    """, from_id=from_id, to_id=to_id, version=release_version, **props)

async def deprecate_relation(rel_id: str):
    """Không xóa — chỉ set valid_to"""
    await graph.query("""
        MATCH ()-[r]->() WHERE id(r) = $rel_id
        SET r.valid_to = datetime()
    """, rel_id=rel_id)
```

**Query temporal tại thời điểm release:**

```cypher
MATCH (r:Requirement)-[rel:REQUIREMENT_HAS_USECASE]->(uc:UseCase)
WHERE rel.valid_from <= datetime($release_date)
  AND (rel.valid_to IS NULL OR rel.valid_to > datetime($release_date))
RETURN r, rel, uc
```

---

#### Task 4.3 — Semantic Conflict Detection
**Effort:** 2 ngày | **Owner:** AI Engineer

```python
# reasoning/semantic_conflict.py

async def detect_semantic_conflicts(new_req: Requirement) -> list[ConflictAlert]:
    # 1. Embed requirement mới
    vector = await embed(f"{new_req.title}\n{new_req.description}")

    # 2. Tìm candidates tương tự trong cùng release
    candidates = await qdrant.search(
        collection="knowledge",
        query_vector=vector,
        query_filter=Filter(must=[
            FieldCondition(key="entity_type", match=MatchValue(value="Requirement")),
            FieldCondition(key="version",     match=MatchValue(value=new_req.version))
        ]),
        limit=10,
        score_threshold=0.80
    )

    conflicts = []
    for candidate in candidates:
        if candidate.id == new_req.req_id:
            continue
        if candidate.score > 0.85:
            # Kiểm tra keyword contradiction
            existing = await get_requirement(candidate.payload["entity_id"])
            if check_keyword_conflict(new_req.description, existing.description):
                conflicts.append(ConflictAlert(
                    req1=new_req.req_id,
                    req2=existing.req_id,
                    similarity=candidate.score,
                    type="SEMANTIC_CONFLICT",
                    confidence=candidate.score
                ))
    return conflicts
```

---

#### Task 4.4 — Governance Layer
**Effort:** 1 ngày | **Owner:** Backend Lead

```python
# context_api/governance.py

RELATION_WHITELIST = {
    "Requirement": ["REQUIREMENT_HAS_USECASE", "REQUIREMENT_VERIFIED_BY_TEST",
                    "DOCUMENT_DESCRIBES_REQUIREMENT", "RELEASE_CONTAINS_REQUIREMENT"],
    "UseCase":     ["USECASE_HAS_API"],
    "API":         ["API_IMPLEMENTED_BY_COMPONENT", "API_COVERED_BY_TEST"],
}

async def validate_relation(from_type: str, to_type: str, rel_type: str) -> bool:
    allowed = RELATION_WHITELIST.get(from_type, [])
    if rel_type not in allowed:
        raise OntologyViolationError(
            f"Relation {rel_type} không được phép từ {from_type} → {to_type}"
        )
    return True

async def freeze_release(release_version: str):
    """Version freeze: không cho phép thay đổi sau khi freeze"""
    await pg.execute("""
        UPDATE releases SET frozen = TRUE, frozen_at = NOW()
        WHERE version = $1
    """, release_version)
```

---

#### Task 4.5 — Performance Testing
**Effort:** 1 ngày | **Owner:** QA Engineer

```javascript
// load_test/k6_script.js

import http from 'k6/http';

export const options = {
    scenarios: {
        context_api: {
            executor: 'constant-vus',
            vus: 50, duration: '2m',
            thresholds: { http_req_duration: ['p95<100'] }  // 95th percentile < 100ms
        },
        impact_analysis: {
            executor: 'constant-vus',
            vus: 20, duration: '2m',
            thresholds: { http_req_duration: ['p95<500'] }
        }
    }
};
```

**SLA Targets:**

| Endpoint | p50 | p95 | p99 |
|---|---|---|---|
| `GET /context` depth=2 | < 20ms | < 100ms | < 200ms |
| `GET /impact` | < 100ms | < 500ms | < 1s |
| `GET /coverage` | < 50ms | < 200ms | < 500ms |
| `GET /reasoning-path` | < 50ms | < 300ms | < 600ms |

---

### 6.2 Definition of Done — Sprint 4

| Hạng mục | Tiêu chí | Kiểm tra bằng |
|---|---|---|
| Outbox Sync | Requirement update sync sang Neo4j trong < 10s | E2E test |
| Temporal Query | Query đúng state tại Release 1.0, 1.1, 2.0 | Cypher assertions |
| Semantic Conflict | Phát hiện 2 cặp conflict trong seed data | Integration test |
| Governance | Relation ngoài whitelist bị reject với 422 | API test |
| Version Freeze | Không cho phép update entity trong frozen release | API test |
| Load Test | p95 < 100ms cho /context với 50 VUs | k6 report |
| Audit Trail | Mọi write có entry trong entity_audit_log | SQL query |

---

## 7. Dependency Map

```
Task 1.1 (Neo4j Schema)
    └─► Task 1.4 (Vertical Slice)
            └─► Task 1.5 (Context API)
                    └─► Task 2.2 (Agent Loop)
                    └─► Task 3.1 (Rule Engine)
                    └─► Task 3.5 (Explainability API)

Task 1.2 (PG Schema)
    └─► Task 1.3 (Seed Data)
    └─► Task 2.4 (Agent Audit)
    └─► Task 4.1 (Outbox Pattern)

Task 2.1 (Redis Memory)
    └─► Task 2.2 (Agent Loop)
    └─► Task 3.4 (Task Queue)

Task 2.5 (Qdrant)
    └─► Task 2.3 (Compression)
    └─► Task 4.3 (Semantic Conflict)

Task 3.3 (Dirty Flag)
    └─► Task 3.4 (Task Queue)

Task 3.1 (Rule Engine)
    └─► Task 3.2 (Conflict Detection)
    └─► Task 4.3 (Semantic Conflict)
```

---

## 8. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| APOC plugin không available trên managed Neo4j | Medium | High | Viết fallback bằng multi-hop MATCH thủ công đến depth 4 |
| Embedding cost vượt budget | Medium | Medium | Batch embed, cache aggressively, dùng model nhỏ hơn (ada-002) |
| Inconsistency giữa Neo4j và Postgres | Medium | High | Outbox pattern + idempotent consumer, retry với backoff |
| Agent loop vô hạn (infinite loop) | Low | High | Max loop count = 50, circuit breaker, timeout 5 phút/session |
| Conflict detection false positive cao | Medium | Medium | Bắt đầu với threshold 0.90, hạ dần dựa trên feedback |
| Graph traversal chậm khi data lớn | Medium | Medium | Index thêm, giới hạn max_nodes=500 trong guardrails, pagination |
| Redis TTL quá ngắn, mất short-term memory | Low | Medium | Monitor TTL hit rate, tăng TTL nếu cần, persist critical state vào PG |

---

## 9. Pre-Implementation Checklist

### Infrastructure

- [ ] Neo4j 5.x running — APOC plugin installed và enabled
- [ ] PostgreSQL 16 running — Alembic sẵn sàng
- [ ] Redis 7 running — persistence enabled (AOF), maxmemory-policy = allkeys-lru
- [ ] Qdrant running — collection `knowledge` đã tạo với vector_size = 1536
- [ ] Python 3.11+ với virtual environment

### Dependencies

```bash
pip install fastapi uvicorn neo4j psycopg2-binary redis qdrant-client \
            langgraph langchain openai alembic python-dotenv httpx pytest
```

### Environment Variables

```bash
# .env — KHÔNG commit lên git

NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_password
NEO4J_DATABASE=traceability

POSTGRES_DSN=postgresql://user:pass@localhost:5432/cognitive_layer

REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your_redis_password

QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=your_qdrant_key   # nếu dùng cloud

OPENAI_API_KEY=sk-...

APP_ENV=development
LOG_LEVEL=DEBUG
```

### Seed Data Validation

```cypher
// Chạy sau khi seed để verify

MATCH (r:Requirement) RETURN count(r) AS req_count;           // Expected: 50
MATCH (uc:UseCase)    RETURN count(uc) AS uc_count;           // Expected: 20
MATCH (a:API)         RETURN count(a) AS api_count;           // Expected: 30
MATCH (tc:TestCase)   RETURN count(tc) AS tc_count;           // Expected: 60

// 5 requirements không có test
MATCH (r:Requirement)
WHERE NOT (r)-[:REQUIREMENT_VERIFIED_BY_TEST]->(:TestCase)
  AND r.status = 'APPROVED'
RETURN count(r);  // Expected: 5
```

### First Run Smoke Test

```bash
# 1. Start services
docker-compose up -d

# 2. Run migrations
alembic upgrade head
python graph/migrations/run_cypher.py 001_initial_schema.cypher

# 3. Seed data
python seed/seed_data.py

# 4. Start API
uvicorn context_api.main:app --reload --port 8000

# 5. Smoke test
curl "http://localhost:8000/health/graph"
curl "http://localhost:8000/context?entity=REQ-001&depth=2"
curl "http://localhost:8000/coverage"
```

---

*Implementation Plan v1.0 — Generated từ Knowledge Graph & Cognitive Layer Specification v2.0*
