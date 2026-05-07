---
skill_id: SKILL-001
version: 1.0.0
status: active
priority: P1
group: Kiến trúc & Thiết kế
created_at: 2026-04-24
---

# SKILL-001 · Software Architecture & System Design

## Mô tả

Bộ kỹ năng thiết kế tổng thể hệ thống phần mềm — từ phân tích yêu cầu đến lựa chọn kiến trúc, thiết kế API contract, và đưa ra Architecture Decision Records (ADR).

## Agents sử dụng

Tất cả agents (nền tảng thiết kế)

## Tài liệu liên kết

- `docs/adr/`
- `docs/product/architecture.md`

---

## Năng lực cốt lõi

### 1. Microservices Architecture

#### Service Boundary Principles

```
Nguyên tắc định nghĩa service boundary:
├── Single Responsibility: Mỗi service sở hữu 1 domain rõ ràng
├── Loose Coupling: Service communicate qua API/events, không share DB
├── High Cohesion: Tất cả logic liên quan nằm trong 1 service
└── Independent Deployability: Deploy từng service không ảnh hưởng service khác
```

#### Knowledge Gateway Service Map

```
┌─────────────────────────────────────────────────────────────┐
│                     API Gateway                              │
│              (Auth, Rate Limit, Routing)                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ BA Knowledge │ │ KGS Platform │ │ BA Knowledge │
│   Service    │ │  (Neo4j)     │ │   Worker     │
│ (Golang/gRPC)│ │ (Golang/gRPC)│ │ (Async jobs) │
└──────────────┘ └──────────────┘ └──────────────┘
        │               │
        ▼               ▼
┌──────────────┐ ┌──────────────┐
│  PostgreSQL  │ │    Neo4j     │
│  (metadata)  │ │ (graph data) │
└──────────────┘ └──────────────┘
```

#### Inter-service Communication

| Pattern | Khi nào dùng | Technology |
|---------|-------------|------------|
| Synchronous RPC | Real-time request/response | gRPC |
| Async Event | Fire-and-forget, decoupled | Kafka / RabbitMQ |
| REST | External clients, simple CRUD | HTTP/JSON |
| Streaming | Large data transfer | gRPC streaming |

### 2. Multi-stage Pipeline Design

```
Pipeline: PRD Text → Knowledge Graph → UI Schema
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Stage 1: Document Ingestion
  Input:  Raw PRD text (Markdown / PDF)
  Output: Normalized text + metadata
  Checkpoint: doc_ingestion_completed

Stage 2: NLP Preprocessing
  Input:  Normalized text
  Output: Tokenized sentences + paragraph classification
  Checkpoint: nlp_preprocessing_completed

Stage 3: LLM Entity Extraction
  Input:  Classified paragraphs
  Output: Entities (Actors, Actions, Objects) + Relations
  Checkpoint: llm_extraction_completed

Stage 4: Knowledge Graph Build
  Input:  Extracted entities + relations
  Output: Neo4j graph populated
  Checkpoint: kg_build_completed

Stage 5: UI Schema Generation
  Input:  Knowledge Graph queries
  Output: JSON Schema for UI rendering
  Checkpoint: ui_schema_generated

Error Recovery:
- Each stage saves checkpoint before completing
- On failure: restart from last checkpoint
- Dead-letter queue for permanently failed jobs
```

### 3. API Contract Design

```yaml
# OpenAPI Contract Template (chuẩn cho tất cả services)
openapi: 3.1.0
info:
  title: "Knowledge Gateway API"
  version: "v1.0.0"
  
# Versioning Strategy: URL-based (/v1/, /v2/)
# Breaking changes → new major version
# Non-breaking additions → same version

paths:
  /v1/documents:
    post:
      operationId: CreateDocument
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateDocumentRequest'
      responses:
        '201':
          description: Document created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Document'
        '400':
          $ref: '#/components/responses/BadRequest'
        '422':
          $ref: '#/components/responses/UnprocessableEntity'
```

### 4. Data Flow Architecture

```
PRD Text Input
    │
    ▼ [Ingest]
Document Store (PostgreSQL)
    │
    ▼ [Parse + NLP]
NLP Processor → Classified Paragraphs
    │
    ▼ [LLM Extract]
Entity Extractor → {Actors, Actions, Entities, Relations}
    │
    ▼ [Graph Build]
Neo4j Knowledge Graph ←→ Graph Query Engine
    │
    ▼ [Schema Gen]
UI Schema Generator → JSON Schema
    │
    ▼ [Render]
React UI Renderer → Visual Prototype
```

### 5. Scalability Patterns

```go
// Horizontal scaling checklist:
// ✅ Stateless services (no in-memory state between requests)
// ✅ DB connection pooling (pgxpool, Neo4j driver pool)
// ✅ Redis caching for expensive computations
// ✅ Worker pool for CPU-bound NLP tasks
// ✅ Graceful shutdown (SIGTERM handling)

// Caching strategy
type CacheKey struct {
    DocID     string
    Stage     string  // "nlp", "extraction", "kg"
    Version   string  // hash of input content
}

// Cache invalidation: content-addressed (hash of input)
// TTL: 24h for extraction results, 1h for schema generation
```

### 6. Architecture Decision Records (ADR)

```markdown
# ADR-{number}: {title}

## Status
Proposed | Accepted | Deprecated | Superseded by ADR-{n}

## Context
[Mô tả vấn đề / bối cảnh dẫn đến quyết định]

## Decision
[Quyết định được đưa ra]

## Rationale
[Lý do: tại sao chọn option này]

## Alternatives Considered
1. **Option A**: [description] — Rejected because [reason]
2. **Option B**: [description] — Rejected because [reason]

## Consequences
**Positive:**
- [benefit 1]

**Negative (trade-offs):**
- [trade-off 1]

## References
- [link to relevant docs]
```

---

## Checklist Architecture Review

- [ ] Service boundaries đã được định nghĩa theo domain (không theo team)
- [ ] Mỗi service có 1 database riêng (không share DB giữa services)
- [ ] Communication pattern đã được chọn (sync vs async) cho mỗi inter-service call
- [ ] API contracts đã được document bằng OpenAPI spec
- [ ] Pipeline stages đã có checkpoint và error recovery
- [ ] Scalability bottlenecks đã được identify và có plan
- [ ] ADR đã được viết cho mọi quyết định kiến trúc quan trọng
- [ ] Data flow diagram đã được cập nhật tại `docs/product/architecture.md`
