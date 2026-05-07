__KNOWLEDGE GRAPH & COGNITIVE LAYER__

Enterprise Implementation Specification

─────────────────────────────────────

*Dành cho hệ thống: URD/SRS chuẩn IEEE · Multi\-Agent BA · Traceability Mapping*

Version 2\.0  |  Tháng 2/2026

# __Mục Lục__

# __PHẦN 0 — Tổng Quan Hệ Thống & Phạm Vi__

Tài liệu này là Implementation Specification đầy đủ cho hệ thống Cognitive Layer phục vụ bài toán Requirement Traceability và Multi\-Agent BA\. Bao gồm schema chi tiết, Cypher queries mẫu, API contracts, chiến lược sync dữ liệu và các pattern production\-ready\.

## 0\.1 Phạm Vi Triển Khai

__Module__

__Mô tả__

__Priority__

__Effort__

Knowledge Graph \(Traceability\)

Graph lõi: Requirement → UseCase → API → Component → Test

P0

3 tuần

Memory Graph \(Agent\)

Short\-term & long\-term memory cho agent loop

P0

2 tuần

Rule\-Based Reasoning Engine

Structural rules, consistency check, conflict detection

P1

2 tuần

Context Builder API

REST layer trên graph, cache, ranked subgraph

P1

1 tuần

Explainability & Audit Trail

Reasoning path, version freeze, governance

P2

1 tuần

Temporal Graph

Release\-based versioning với valid\_from/valid\_to

P2

1 tuần

## 0\.2 Tech Stack Chính

__Layer__

__Technology__

__Mục đích__

Graph DB

Neo4j 5\.x \(AuraDB hoặc self\-hosted\)

Lưu trữ graph, Cypher queries

Relational DB

PostgreSQL 16

Metadata, user, config, audit log

Vector DB

Qdrant

Semantic search, embedding requirements

Cache

Redis 7

Short\-term memory, TTL\-based context

Agent Framework

LangGraph \(Python\)

Agent loop orchestration

API Gateway

FastAPI

REST endpoints, auth, rate limiting

Message Queue

Redis Streams / RabbitMQ

Async sync giữa các store

Embedding

OpenAI text\-embedding\-3\-small

Semantic similarity cho conflict detection

# __PHẦN I — Ontology & Entity Schema__

Đây là phần quan trọng nhất cần implement trước\. Toàn bộ hệ thống phụ thuộc vào schema này\. Mỗi entity type có property schema đầy đủ và constraints trong Neo4j\.

## 1\.1 Entity Schema Chi Tiết

### Requirement

__Property__

__Type__

__Required__

__Mô tả__

req\_id

String

✓

Format: REQ\-\{module\}\-\{number\}, VD: REQ\-AUTH\-001

title

String

✓

Tiêu đề ngắn gọn \(max 200 ký tự\)

description

String

✓

Mô tả đầy đủ theo chuẩn IEEE 830

type

Enum

✓

FUNCTIONAL | NON\_FUNCTIONAL | CONSTRAINT | INTERFACE

priority

Enum

✓

MUST | SHOULD | COULD | WONT \(MoSCoW\)

status

Enum

✓

DRAFT | REVIEW | APPROVED | IMPLEMENTED | VERIFIED | DEPRECATED

version

String

✓

Semantic version: 1\.0\.0

source\_doc

String

✓

ID của Document nguồn \(URD, PRD, SRS\)

author

String

✓

ID của người tạo

created\_at

DateTime

✓

ISO 8601

updated\_at

DateTime

✓

ISO 8601

rationale

String

Lý do tồn tại của requirement

acceptance\_criteria

String\[\]

List điều kiện nghiệm thu

ieee\_section

String

Section IEEE 830 tương ứng, VD: 3\.1\.2

tags

String\[\]

Nhãn phân loại tự do

embedding\_id

String

ID vector trong Qdrant

### UseCase

__Property__

__Type__

__Required__

__Mô tả__

uc\_id

String

✓

Format: UC\-\{number\}, VD: UC\-001

name

String

✓

Tên use case

actor

String\[\]

✓

Danh sách actor tham gia

preconditions

String\[\]

✓

Điều kiện tiên quyết

postconditions

String\[\]

✓

Kết quả sau khi thực hiện

main\_flow

String\[\]

✓

Luồng chính \(step\-by\-step\)

alt\_flows

JSON

Các luồng thay thế \{ id, condition, steps \}

version

String

✓

Semantic version

status

Enum

✓

DRAFT | APPROVED | DEPRECATED

### API

__Property__

__Type__

__Required__

__Mô tả__

api\_id

String

✓

Format: API\-\{service\}\-\{number\}

endpoint

String

✓

VD: POST /api/v1/auth/login

method

Enum

✓

GET | POST | PUT | PATCH | DELETE

service

String

✓

Tên microservice sở hữu

request\_schema

JSON

✓

JSON Schema của request body

response\_schema

JSON

✓

JSON Schema của response

auth\_required

Boolean

✓

Yêu cầu xác thực không

version

String

✓

API version: v1, v2

status

Enum

✓

PROPOSED | APPROVED | DEPRECATED

sla\_ms

Integer

SLA response time tính bằng ms

### Component

__Property__

__Type__

__Required__

__Mô tả__

comp\_id

String

✓

Format: COMP\-\{name\}

name

String

✓

Tên component

type

Enum

✓

SERVICE | MODULE | LIBRARY | DATABASE | QUEUE

language

String

Ngôn ngữ lập trình chính

repo\_url

String

URL repository

team\_owner

String

✓

Team sở hữu

tech\_stack

String\[\]

Danh sách công nghệ sử dụng

version

String

✓

Version hiện tại

### TestCase

__Property__

__Type__

__Required__

__Mô tả__

tc\_id

String

✓

Format: TC\-\{number\}

name

String

✓

Tên test case

type

Enum

✓

UNIT | INTEGRATION | E2E | PERFORMANCE | SECURITY

status

Enum

✓

PLANNED | AUTOMATED | MANUAL | PASSED | FAILED | SKIPPED

steps

JSON\[\]

✓

Array \{step, expected\_result\}

automation\_path

String

Path đến test script nếu automated

priority

Enum

✓

HIGH | MEDIUM | LOW

last\_run\_at

DateTime

Lần cuối chạy test

last\_run\_result

Enum

PASSED | FAILED | ERROR

## 1\.2 Relation Schema Chi Tiết

Mỗi relation \(edge\) trong graph cần có đầy đủ các metadata sau\. Đây là thiết kế quan trọng để hỗ trợ impact analysis có trọng số và audit trail\.

__Edge Property__

__Type__

__Mô tả__

confidence

Float \[0\-1\]

Độ tin cậy của relation\. Manual = 1\.0, Auto\-generated = 0\.7, Inferred = 0\.5

source

String

Nguồn tạo: 'manual', 'agent', 'rule\_engine', 'import'

version

String

Version khi relation được tạo

created\_by

String

User ID hoặc agent ID

created\_at

DateTime

Thời điểm tạo ISO 8601

impact\_weight

Float \[0\-1\]

Trọng số ảnh hưởng khi tính impact score

valid\_from

DateTime

Bắt đầu có hiệu lực \(Temporal Graph\)

valid\_to

DateTime

Hết hiệu lực\. NULL = còn hiệu lực

note

String

Ghi chú tùy chọn

## 1\.3 Neo4j Setup Script — Constraints & Indexes

// ===== CONSTRAINTS =====

CREATE CONSTRAINT req\_id\_unique IF NOT EXISTS FOR \(r:Requirement\) REQUIRE r\.req\_id IS UNIQUE;

CREATE CONSTRAINT uc\_id\_unique  IF NOT EXISTS FOR \(u:UseCase\)    REQUIRE u\.uc\_id   IS UNIQUE;

CREATE CONSTRAINT api\_id\_unique IF NOT EXISTS FOR \(a:API\)        REQUIRE a\.api\_id  IS UNIQUE;

CREATE CONSTRAINT tc\_id\_unique  IF NOT EXISTS FOR \(t:TestCase\)   REQUIRE t\.tc\_id   IS UNIQUE;

CREATE CONSTRAINT comp\_id\_unique IF NOT EXISTS FOR \(c:Component\) REQUIRE c\.comp\_id IS UNIQUE;

// ===== INDEXES =====

CREATE INDEX req\_status\_idx   IF NOT EXISTS FOR \(r:Requirement\) ON \(r\.status\);

CREATE INDEX req\_version\_idx  IF NOT EXISTS FOR \(r:Requirement\) ON \(r\.version\);

CREATE INDEX tc\_status\_idx    IF NOT EXISTS FOR \(t:TestCase\)    ON \(t\.status, t\.type\);

CREATE INDEX api\_service\_idx  IF NOT EXISTS FOR \(a:API\)         ON \(a\.service\);

# __PHẦN II — Traceability Graph: Cypher Queries Mẫu__

Phần này cung cấp các Cypher queries cụ thể cho từng use case chính\. Đây là vertical slice đầu tiên cần implement và verify\.

## 2\.1 Tạo Trace Path Đầy Đủ

// Tạo một requirement và toàn bộ trace chain

CREATE \(r:Requirement \{

  req\_id: 'REQ\-AUTH\-001', title: 'Đăng nhập bằng email/password',

  type: 'FUNCTIONAL', priority: 'MUST', status: 'APPROVED',

  version: '1\.0\.0', source\_doc: 'URD\-001', author: 'user\-01',

  created\_at: datetime\(\), updated\_at: datetime\(\)

\}\)

CREATE \(uc:UseCase \{ uc\_id: 'UC\-001', name: 'User Login', version: '1\.0\.0', status: 'APPROVED' \}\)

CREATE \(api:API \{ api\_id: 'API\-AUTH\-001', endpoint: 'POST /api/v1/auth/login',

  method: 'POST', service: 'auth\-service', version: 'v1', status: 'APPROVED' \}\)

CREATE \(comp:Component \{ comp\_id: 'COMP\-AUTH\-SVC', name: 'Auth Service',

  type: 'SERVICE', team\_owner: 'team\-backend', version: '2\.1\.0' \}\)

CREATE \(tc:TestCase \{ tc\_id: 'TC\-001', name: 'Login với credentials hợp lệ',

  type: 'INTEGRATION', status: 'AUTOMATED', priority: 'HIGH' \}\)

CREATE \(r\)\-\[:REQUIREMENT\_HAS\_USECASE \{ confidence:1\.0, source:'manual',

  impact\_weight:0\.9, created\_by:'user\-01', created\_at:datetime\(\) \}\]\->\(uc\)

CREATE \(uc\)\-\[:USECASE\_HAS\_API \{ confidence:1\.0, source:'manual',

  impact\_weight:0\.8, created\_by:'user\-01', created\_at:datetime\(\) \}\]\->\(api\)

CREATE \(api\)\-\[:API\_IMPLEMENTED\_BY\_COMPONENT \{ confidence:1\.0, source:'manual',

  impact\_weight:0\.7, created\_by:'user\-01', created\_at:datetime\(\) \}\]\->\(comp\)

CREATE \(r\)\-\[:REQUIREMENT\_VERIFIED\_BY\_TEST \{ confidence:1\.0, source:'manual',

  impact\_weight:1\.0, created\_by:'user\-01', created\_at:datetime\(\) \}\]\->\(tc\)

CREATE \(api\)\-\[:API\_COVERED\_BY\_TEST \{ confidence:1\.0, source:'manual',

  impact\_weight:0\.9, created\_by:'user\-01', created\_at:datetime\(\) \}\]\->\(tc\)

## 2\.2 Impact Analysis Query

Khi một Requirement thay đổi, query này tìm tất cả nodes bị ảnh hưởng và tính impact score:

// Impact Analysis khi Requirement thay đổi

MATCH \(r:Requirement \{ req\_id: $req\_id \}\)

CALL apoc\.path\.subgraphAll\(r, \{

  maxLevel: 4,

  relationshipFilter: 'REQUIREMENT\_HAS\_USECASE>|USECASE\_HAS\_API>|API\_IMPLEMENTED\_BY\_COMPONENT>|API\_COVERED\_BY\_TEST>'

\}\) YIELD nodes, relationships

WITH r, nodes, relationships,

     \[rel IN relationships | rel\.impact\_weight \* rel\.confidence\] AS weights

RETURN

  r\.req\_id                                            AS requirement,

  \[n IN nodes WHERE 'UseCase'   IN labels\(n\) | n\.uc\_id\]   AS affected\_usecases,

  \[n IN nodes WHERE 'API'       IN labels\(n\) | n\.api\_id\]  AS affected\_apis,

  \[n IN nodes WHERE 'Component' IN labels\(n\) | n\.comp\_id\] AS affected\_components,

  \[n IN nodes WHERE 'TestCase'  IN labels\(n\) | n\.tc\_id\]   AS affected\_tests,

  round\(reduce\(s=0\.0, w IN weights | s \+ w\) / size\(weights\), 2\) AS impact\_score

*⚠ Cài đặt APOC plugin cho Neo4j trước khi dùng apoc\.path\.subgraphAll\. Nếu không có APOC, dùng MATCH với OPTIONAL MATCH thủ công đến depth 4\.*

## 2\.3 Coverage Analysis Queries

// === 1\. Requirement chưa có TestCase ===

MATCH \(r:Requirement\)

WHERE NOT \(r\)\-\[:REQUIREMENT\_VERIFIED\_BY\_TEST\]\->\(:TestCase\)

  AND r\.status IN \['APPROVED', 'IMPLEMENTED'\]

RETURN r\.req\_id, r\.title, r\.priority, r\.status

ORDER BY r\.priority DESC;

// === 2\. API chưa có TestCase ===

MATCH \(a:API\)

WHERE NOT \(a\)\-\[:API\_COVERED\_BY\_TEST\]\->\(:TestCase\)

  AND a\.status = 'APPROVED'

RETURN a\.api\_id, a\.endpoint, a\.service;

// === 3\. Coverage Rate tổng thể ===

MATCH \(r:Requirement\) WHERE r\.status = 'APPROVED'

WITH count\(r\) AS total

MATCH \(r:Requirement\)\-\[:REQUIREMENT\_VERIFIED\_BY\_TEST\]\->\(:TestCase\)

WHERE r\.status = 'APPROVED'

WITH total, count\(DISTINCT r\) AS covered

RETURN total, covered, round\(toFloat\(covered\)/total \* 100, 1\) AS coverage\_pct;

## 2\.4 Reasoning Path Query

// Lấy đường đi từ Requirement đến TestCase \(cho Explainability\)

MATCH path = \(r:Requirement \{ req\_id: $req\_id \}\)\-\[\*\.\.6\]\->\(t:TestCase \{ tc\_id: $tc\_id \}\)

WITH path, \[rel IN relationships\(path\) | \{

  type:     type\(rel\),

  source:   rel\.source,

  confidence: rel\.confidence,

  created\_by: rel\.created\_by

\}\] AS edge\_info

RETURN

  \[n IN nodes\(path\) | \{ label: labels\(n\)\[0\], id: coalesce\(n\.req\_id, n\.uc\_id, n\.api\_id, n\.comp\_id, n\.tc\_id\), name: n\.title \}\] AS node\_chain,

  edge\_info,

  length\(path\) AS depth

ORDER BY depth ASC LIMIT 5;

## 2\.5 Temporal Query \(Release\-based\)

// Truy vấn trạng thái graph tại thời điểm Release 2\.1

MATCH \(r:Requirement\)\-\[rel:REQUIREMENT\_HAS\_USECASE\]\->\(uc:UseCase\)

WHERE rel\.valid\_from <= $release\_date

  AND \(rel\.valid\_to IS NULL OR rel\.valid\_to > $release\_date\)

  AND r\.version = $release\_version

RETURN r, rel, uc;

*ℹ Khi tạo relation mới cho release: SET rel\.valid\_from = datetime\(release\_start\), rel\.valid\_to = NULL\. Khi deprecated: SET rel\.valid\_to = datetime\(deprecation\_date\)\. KHÔNG xóa relation cũ\.*

# __PHẦN III — Memory Graph cho Agent Loop__

Memory Graph lưu trữ trạng thái và lịch sử hoạt động của agent\. Được phân tách rõ ràng thành short\-term \(Redis \+ Neo4j\) và long\-term \(Postgres \+ Neo4j\)\.

## 3\.1 Entity Schema Bổ Sung

### AgentSession

__Property__

__Type__

__Mô tả__

session\_id

String

UUID v4

agent\_type

Enum

BA\_AGENT | REVIEW\_AGENT | COVERAGE\_AGENT | IMPACT\_AGENT

task\_description

String

Mô tả nhiệm vụ ban đầu

status

Enum

RUNNING | COMPLETED | FAILED | PAUSED

started\_at

DateTime

ended\_at

DateTime

NULL nếu đang chạy

token\_count

Integer

Tổng token đã dùng

loop\_count

Integer

Số vòng lặp agent

ttl\_seconds

Integer

TTL của session \(short\-term memory\)

### ContextSnapshot

__Property__

__Type__

__Mô tả__

snapshot\_id

String

UUID v4

session\_id

String

FK đến AgentSession

loop\_index

Integer

Vòng lặp thứ mấy

working\_entities

String\[\]

IDs của entities đang được xử lý

state\_summary

String

Tóm tắt trạng thái hiện tại \(compressed\)

created\_at

DateTime

ttl\_at

DateTime

Thời điểm hết hạn \(short\-term\)

## 3\.2 Memory Architecture & Sync Strategy

__Loại Memory__

__Storage__

__TTL__

__Nội dung__

__Sync__

Short\-term \(active\)

Redis Hash

30 phút

ContextSnapshot, PlanStep

Write\-through

Short\-term \(graph\)

Neo4j \(volatile\)

2 giờ

Thought, ToolCall, Observation

Async via queue

Long\-term \(semantic\)

Qdrant

Vĩnh viễn

Embedding của Knowledge Entity

Event\-driven

Long\-term \(structured\)

Postgres

Vĩnh viễn

AgentSession audit log

Sync write

Long\-term \(graph\)

Neo4j \(stable\)

Vĩnh viễn

Knowledge Entity, stable relations

Batch compress

## 3\.3 Memory Compression — Chi Tiết

Sau mỗi agent loop \(hoặc sau N vòng lặp\), thought chain phải được nén để tránh context window overflow:

\# memory\_compression\.py

async def compress\_memory\(session\_id: str, loop\_index: int\):

    \# 1\. Lấy thought chain từ Neo4j

    thoughts = await graph\.query\('''

        MATCH \(s:AgentSession \{session\_id: $sid\}\)\-\[:GENERATED\_THOUGHT\]\->\(t:Thought\)

        WHERE t\.loop\_index <= $loop ORDER BY t\.created\_at ASC

        RETURN t\.content, t\.loop\_index''', sid=session\_id, loop=loop\_index\)

    \# 2\. Summarize bằng LLM \(giới hạn 200 token\)

    summary = await llm\.invoke\(COMPRESS\_PROMPT\.format\(thoughts=thoughts\)\)

    \# 3\. Tạo KnowledgeEntity trong Neo4j

    await graph\.query\('''

        CREATE \(k:KnowledgeEntity \{

            ke\_id: $ke\_id, content: $summary, source\_session: $sid,

            source\_loops: $loops, created\_at: datetime\(\), type: 'COMPRESSED\_MEMORY'

        \}\)''', ke\_id=uuid4\(\), summary=summary, sid=session\_id, loops=loop\_index\)

    \# 4\. Embed vào Qdrant

    embedding = await embed\(summary\)

    await qdrant\.upsert\(collection='knowledge', vectors=\[embedding\], payload=\[\{'ke\_id': ke\_id\}\]\)

    \# 5\. Xóa thoughts cũ khỏi Neo4j \(giữ last 5\)

    await graph\.query\('''

        MATCH \(s:AgentSession \{session\_id: $sid\}\)\-\[:GENERATED\_THOUGHT\]\->\(t:Thought\)

        WHERE t\.loop\_index < $keep\_from DELETE t''', keep\_from=loop\_index\-5\)

*⚠ Không bao giờ xóa AgentSession node\. Chỉ xóa Thought nodes trung gian\. Giữ lại Decision và ToolCall nodes cho audit trail\.*

# __PHẦN IV — Rule\-Based Reasoning Engine__

Rule Engine hoạt động độc lập, được trigger theo schedule hoặc event\. Không inject logic vào agent loop trực tiếp\.

## 4\.1 Conflict Detection — Chi Tiết

Đây là bài toán khó nhất\. Conflict được phát hiện theo 3 tầng:

__Tầng__

__Phương pháp__

__Ngưỡng__

__Ví dụ__

Semantic

Cosine similarity của embeddings

Similarity > 0\.85 \+ khác nhau về priority/constraint

REQ\-A: 'hệ thống phải phản hồi < 2s' vs REQ\-B: 'max response time = 5s'

Structural

Graph pattern matching

Cùng Release, cùng API, trái chiều nhau

REQ\-A → UseCase\-X; REQ\-B → UseCase\-Y; cả hai block cùng Component

Rule\-based

Cypher pattern \+ keyword list

Từ khóa contradictory: 'không được', 'cấm' vs 'phải'

REQ\-A: 'user không được xóa' vs REQ\-B: 'user phải có thể xóa account'

## 4\.2 Rule Registry — Định nghĩa Rules

\# rules\_registry\.py

RULES = \[

  \{

    'rule\_id': 'STRUCT\-001',

    'name': 'Inferred API Dependency',

    'type': 'STRUCTURAL',

    'cypher\_pattern': '''

      MATCH \(r:Requirement\)\-\[:REQUIREMENT\_HAS\_USECASE\]\->\(u:UseCase\)

            \-\[:USECASE\_HAS\_API\]\->\(a:API\)

      WHERE NOT \(r\)\-\[:REQUIREMENT\_DEPENDS\_ON\_API\]\->\(a\)

      RETURN r\.req\_id, a\.api\_id''',

    'action': 'CREATE\_INFERRED\_RELATION',

    'relation\_type': 'INFERRED\_DEPENDENCY',

    'confidence': 0\.7

  \},

  \{

    'rule\_id': 'CONSIST\-001',

    'name': 'Missing Test Coverage',

    'type': 'CONSISTENCY',

    'cypher\_pattern': '''

      MATCH \(r:Requirement\)

      WHERE r\.status = 'APPROVED'

        AND NOT \(r\)\-\[:REQUIREMENT\_VERIFIED\_BY\_TEST\]\->\(:TestCase\)

      RETURN r\.req\_id''',

    'action': 'CREATE\_FLAG',

    'flag\_type': 'MISSING\_COVERAGE'

  \},

  \{

    'rule\_id': 'CONFLICT\-001',

    'name': 'Semantic Conflict Detection',

    'type': 'CONFLICT',

    'action': 'SEMANTIC\_SIMILARITY\_CHECK',

    'threshold': 0\.85

  \}

\]

## 4\.3 Inferred Relations — Không Overwrite

Tất cả relations được tạo bởi Rule Engine phải có relation\_type riêng\. KHÔNG bao giờ overwrite manual relations:

// Tạo inferred relation an toàn

MERGE \(r:Requirement \{req\_id: $req\_id\}\)\-\[rel:INFERRED\_DEPENDENCY \{

  target\_id: $api\_id, rule\_id: $rule\_id

\}\]\->\(a:API \{api\_id: $api\_id\}\)

ON CREATE SET

  rel\.confidence  = 0\.7,

  rel\.source      = 'rule\_engine',

  rel\.rule\_id     = $rule\_id,

  rel\.created\_at  = datetime\(\),

  rel\.valid\_from  = datetime\(\)

ON MATCH SET

  rel\.updated\_at  = datetime\(\),

  rel\.confidence  = CASE WHEN rel\.confidence < 0\.9 THEN rel\.confidence \+ 0\.05 ELSE 0\.9 END

*⚠ Confidence của inferred relation tăng dần khi rule xác nhận nhiều lần, tối đa 0\.9\. Chỉ manual relation mới có confidence = 1\.0\.*

# __PHẦN V — Partial Regenerate Strategy__

Đây là use case quan trọng nhất trong hệ thống BA\. Khi một phần của SRS/URD thay đổi, cần cascade update có kiểm soát\.

## 5\.1 Dirty\-Flag Propagation Pattern

__Bước__

__Action__

__Mechanism__

1\. Detect Change

So sánh version mới với cũ của Requirement

Diff entity properties, version bump trigger

2\. Mark Dirty

SET r\.is\_dirty = true, r\.dirty\_since = datetime\(\)

Immediate write to Neo4j

3\. Propagate

Traverse downstream, mark tất cả affected nodes

BFS theo edge types, depth giới hạn = 4

4\. Notify

Gửi event vào queue cho từng agent type

Redis Streams, partitioned by entity type

5\. Agent Process

Từng agent nhận task, xử lý, clear dirty flag

ACK sau khi complete

6\. Re\-validate

Rule Engine chạy lại sau khi all agents done

Trigger via queue event

## 5\.2 Propagation Query

// Bước 2\+3: Mark dirty và propagate

MATCH \(r:Requirement \{ req\_id: $req\_id \}\)

SET r\.is\_dirty = true, r\.dirty\_since = datetime\(\), r\.dirty\_reason = $change\_summary

WITH r

MATCH \(r\)\-\[\*1\.\.4\]\->\(downstream\)

WHERE 'UseCase' IN labels\(downstream\)

   OR 'API' IN labels\(downstream\)

   OR 'TestCase' IN labels\(downstream\)

   OR 'Component' IN labels\(downstream\)

SET downstream\.is\_dirty = true,

    downstream\.dirty\_from\_req = $req\_id,

    downstream\.dirty\_since = datetime\(\)

RETURN count\(downstream\) AS marked\_dirty;

## 5\.3 Agent Task Queue Schema \(Redis Streams\)

\# Format message trong Redis Stream 'agent:tasks'

\{

  'task\_id':     'task\-uuid',

  'agent\_type':  'BA\_AGENT' | 'REVIEW\_AGENT' | 'COVERAGE\_AGENT',

  'entity\_type': 'Requirement' | 'UseCase' | 'API' | 'TestCase',

  'entity\_id':   'REQ\-AUTH\-001',

  'action':      'REGENERATE' | 'REVIEW' | 'VALIDATE\_COVERAGE',

  'priority':    'HIGH' | 'NORMAL' | 'LOW',

  'context': \{

    'changed\_req\_id': 'REQ\-AUTH\-001',

    'change\_type':    'DESCRIPTION\_UPDATED',

    'old\_version':    '1\.0\.0',

    'new\_version':    '1\.1\.0'

  \},

  'created\_at':  '2026\-02\-23T10:00:00Z'

\}

# __PHẦN VI — Context Builder API__

Agent không bao giờ query raw graph\. Tất cả truy vấn đi qua Context Builder Service để đảm bảo guardrails và hiệu năng\.

## 6\.1 API Endpoints

__Method__

__Endpoint__

__Mô tả__

__Cache TTL__

GET

/context?entity=\{id\}&depth=\{1\-4\}

Ranked subgraph xung quanh entity

5 phút

GET

/impact?req\_id=\{id\}

Impact analysis khi requirement thay đổi

2 phút

GET

/coverage?release=\{version\}

Coverage report cho release

10 phút

GET

/reasoning\-path?from=\{id\}&to=\{id\}

Đường đi giải thích giữa 2 entities

15 phút

POST

/validate

Validate một subgraph theo ontology rules

No cache

GET

/conflicts?release=\{version\}

Danh sách conflicts trong release

5 phút

POST

/dirty/propagate

Trigger dirty\-flag propagation

No cache

GET

/health/graph

Stats về graph: node count, edge count

1 phút

## 6\.2 Response Format Chuẩn

// GET /context?entity=REQ\-AUTH\-001&depth=2

\{

  'entity': \{

    'id': 'REQ\-AUTH\-001', 'type': 'Requirement', 'title': 'Đăng nhập bằng email/password'

  \},

  'subgraph': \{

    'nodes': \[

      \{ 'id': 'UC\-001', 'type': 'UseCase', 'name': 'User Login', 'relevance\_score': 0\.95 \},

      \{ 'id': 'API\-AUTH\-001', 'type': 'API', 'endpoint': 'POST /api/v1/auth/login', 'relevance\_score': 0\.88 \}

    \],

    'edges': \[

      \{ 'from': 'REQ\-AUTH\-001', 'to': 'UC\-001', 'type': 'REQUIREMENT\_HAS\_USECASE', 'confidence': 1\.0 \}

    \]

  \},

  'meta': \{

    'depth': 2, 'node\_count': 5, 'edge\_count': 4,

    'guardrail\_applied': false, 'query\_time\_ms': 12, 'cached': true

  \}

\}

## 6\.3 Guardrails Config

\# guardrails\.py

GUARDRAILS = \{

  'max\_depth':      4,      \# Giới hạn traversal depth

  'max\_nodes':      500,    \# Giới hạn số nodes trong response

  'max\_edges':      1000,   \# Giới hạn số edges

  'min\_confidence': 0\.5,    \# Loại bỏ edges confidence thấp

  'allowed\_entity\_types': \['Requirement','UseCase','API','Component','TestCase'\],

  'allowed\_relation\_types': \[

    'REQUIREMENT\_HAS\_USECASE', 'USECASE\_HAS\_API',

    'API\_IMPLEMENTED\_BY\_COMPONENT', 'API\_COVERED\_BY\_TEST',

    'REQUIREMENT\_VERIFIED\_BY\_TEST', 'INFERRED\_DEPENDENCY'

  \],

  'version\_constraint': True  \# Chỉ lấy nodes cùng release version

\}

# __PHẦN VII — Data Sync Strategy \(Multi\-Store\)__

Đây là phần quan trọng để tránh inconsistency khi có nhiều store: Neo4j, Postgres, Qdrant, Redis\.

## 7\.1 Event\-Driven Sync Flow

__Event__

__Source__

__Target\(s\)__

__Cơ chế__

__Consistency__

Requirement CREATED

Postgres \(source of truth\)

Neo4j, Qdrant

Outbox pattern → Redis Stream

Eventually consistent

Requirement UPDATED

Postgres

Neo4j, Qdrant, Redis \(cache invalidate\)

Outbox → Stream → Consumer

Eventually consistent

Relation CREATED \(manual\)

API → Neo4j

Postgres \(audit log\)

Sync write

Strong consistent

Agent WRITES to graph

LangGraph agent

Neo4j \(staging area\)

Async, with validation

Eventual

Rule Engine INFERS

Rule Engine → Neo4j

Postgres \(flag log\)

Async

Eventual

Session ENDS

Agent → Postgres

Neo4j \(compress\), Qdrant \(embed\)

Batch job

Eventual

## 7\.2 Outbox Pattern Implementation

\-\- Postgres: Bảng outbox để đảm bảo at\-least\-once delivery

CREATE TABLE event\_outbox \(

  id          UUID PRIMARY KEY DEFAULT gen\_random\_uuid\(\),

  entity\_type VARCHAR\(50\) NOT NULL,   \-\- 'Requirement', 'UseCase', etc\.

  entity\_id   VARCHAR\(100\) NOT NULL,

  event\_type  VARCHAR\(50\) NOT NULL,   \-\- 'CREATED', 'UPDATED', 'DELETED'

  payload     JSONB NOT NULL,

  created\_at  TIMESTAMPTZ DEFAULT NOW\(\),

  processed   BOOLEAN DEFAULT FALSE,

  processed\_at TIMESTAMPTZ,

  retry\_count INTEGER DEFAULT 0

\);

CREATE INDEX outbox\_unprocessed ON event\_outbox\(processed, created\_at\) WHERE NOT processed;

*⚠ CDC \(Change Data Capture\) với Debezium là giải pháp tốt hơn Outbox Pattern cho production lớn, nhưng phức tạp hơn để setup ban đầu\. Bắt đầu với Outbox Pattern\.*

# __PHẦN VIII — Implementation Roadmap__

## Sprint 1 \(Tuần 1\-2\): Foundation

1. Neo4j setup: constraints, indexes, test data seed script
2. Vertical slice đầy đủ: Requirement → UseCase → API → Component → TestCase
3. Cypher queries cho impact analysis và coverage analysis
4. FastAPI skeleton với /context và /coverage endpoints
5. Postgres schema: requirements, audit\_log, event\_outbox

## Sprint 2 \(Tuần 3\-4\): Agent Memory

1. Redis integration cho short\-term memory
2. LangGraph agent loop cơ bản với Memory Graph
3. Memory compression pipeline
4. Qdrant setup và embedding pipeline
5. AgentSession audit trail vào Postgres

## Sprint 3 \(Tuần 5\-6\): Reasoning & Partial Regenerate

1. Rule Engine: Rule Registry, STRUCT\-001 và CONSIST\-001
2. Conflict detection: tầng structural và rule\-based trước
3. Dirty\-flag propagation pattern
4. Redis Streams task queue
5. Context Builder đầy đủ với guardrails

## Sprint 4 \(Tuần 7\-8\): Production Hardening

1. Outbox pattern và event\-driven sync
2. Semantic conflict detection \(Qdrant similarity\)
3. Temporal graph với valid\_from/valid\_to
4. Explainability API: /reasoning\-path
5. Governance: relation whitelist, version freeze
6. Load testing và performance tuning

## 8\.1 Definition of Done cho Sprint 1

__Hạng mục__

__Tiêu chí__

__Kiểm tra bằng__

Graph Schema

Tất cả constraints và indexes tạo thành công

neo4j\-admin validate

Vertical Slice

Tạo được trace path REQ→UC→API→COMP→TC hoàn chỉnh

Cypher query thủ công

Impact Analysis

Query trả về impact\_score chính xác với test data

Unit test Python

Coverage Query

Detect được requirement không có test

Cypher \+ assert

/context API

Response trong < 100ms cho depth=2 với 1000 nodes

Load test k6

/coverage API

Coverage rate chính xác ± 0\.1%

Assert với seed data

# __PHẦN IX — Anti\-Patterns Cần Tránh__

__Anti\-Pattern__

__Vấn đề__

__Solution__

Agent query raw graph trực tiếp

Không có guardrails, có thể traversal vô hạn, expose sensitive data

Luôn đi qua Context Builder API

Overwrite manual relation bằng inferred

Mất audit trail, giảm độ tin cậy của graph

Tạo relation type riêng INFERRED\_DEPENDENCY

Xóa Requirement/Relation cũ

Phá vỡ audit trail, không thể time\-travel query

Dùng valid\_to và status = DEPRECATED

Sync đồng bộ giữa Neo4j và Postgres

Distributed transaction khó, dễ deadlock

Dùng Outbox Pattern \+ eventual consistency

Lưu toàn bộ thought chain không nén

Context window overflow, chi phí LLM cao

Compress sau mỗi N vòng lặp

Một rule engine chạy đồng bộ trong agent loop

Block agent, tăng latency

Rule Engine chạy async, trigger qua event

Dùng confidence = 1\.0 cho auto\-generated

Gây nhầm lẫn với manual data

Auto\-generated max confidence = 0\.85

# __PHẦN X — Pre\-Implementation Checklist__

## Infrastructure

- Neo4j 5\.x khởi động, APOC plugin cài đặt
- Postgres 16 khởi động, schema migration tool sẵn sàng \(Alembic\)
- Redis 7 với persistence enabled \(AOF\)
- Qdrant đang chạy, collection 'knowledge' đã tạo
- Môi trường Python với LangGraph, neo4j\-driver, redis, qdrant\-client

## Configuration

- Connection strings và secrets trong \.env \(KHÔNG commit lên git\)
- Neo4j credentials, database name = 'traceability'
- Redis password và TLS nếu production
- Qdrant API key nếu dùng cloud
- OpenAI API key cho embedding

## Seed Data

- Script tạo 50 Requirement mẫu \(mix FUNCTIONAL \+ NON\_FUNCTIONAL\)
- 20 UseCase, 30 API, 10 Component, 60 TestCase mẫu
- Relations với confidence đa dạng \(0\.5, 0\.7, 0\.9, 1\.0\)
- Ít nhất 5 requirement không có test \(để test coverage detection\)
- Ít nhất 2 cặp requirement conflict \(để test conflict detection\)

─────────────────────────────────────

__END OF DOCUMENT — v2\.0__

*Tài liệu này đủ để bắt đầu implement Sprint 1\.*

