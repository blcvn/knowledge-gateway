# ADR-001: CQRS 3-Mode Architecture

**Status**: Accepted  
**Date**: 2026-06-17  
**Deciders**: KG Service team

---

## Context

KG Service cần phục vụ 3 loại access pattern hoàn toàn khác nhau:
1. **Write** — transactional ACID, schema validation, audit trail
2. **Graph traversal** — pattern matching, relationship-aware queries
3. **Semantic search** — vector similarity, full-text, hybrid

Không có một database nào tối ưu tốt cho cả 3 pattern này đồng thời.

## Decision

Áp dụng **CQRS 3-Mode** với 3 store tách biệt:

| Mode | Store | Optimized for |
|:---|:---|:---|
| Write | PostgreSQL (source of truth) | ACID transaction, RLS, schema validation |
| Read | Graph DB (read replica) | Graph traversal, pattern queries via compiled DSL |
| Search | Vector DB (read replica) | Semantic search, ANN, payload filtering |

**PostgreSQL là source of truth duy nhất.** Graph DB và Vector DB là read replicas được sync qua outbox pattern — không nhận write trực tiếp.

## Consequences

**Positive**:
- Mỗi store làm tốt đúng một việc → performance optimal cho từng access pattern
- PostgreSQL RLS làm lớp bảo vệ thứ 2
- Write path không bị chậm bởi graph/vector write latency
- Graph và Vector backend có thể thay thế mà không ảnh hưởng write path

**Negative/Trade-offs**:
- **Eventual consistency**: dữ liệu mới write sẽ có độ trễ < 2s (P95) trước khi xuất hiện trong Graph/Vector
- Cần Realtime read fallback mode cho trường hợp cần consistency tuyệt đối
- Tăng complexity vận hành: 3 stores cần monitor, reconcile
- AccessGrant change phải sync `acl_visible_to` tới cả Graph và Vector

**Mitigations**:
- `?mode=realtime` parameter: fallback to PostgreSQL read nếu graph version lag
- Hourly reconciliation job + drift alerting
- Outbox pattern đảm bảo at-least-once delivery

---

# ADR-002: Query Pattern DSL thay thế Cypher thô

**Status**: Accepted  
**Date**: 2026-06-17

---

## Context

KG Service cần cho phép domain owner đăng ký query templates. Hai options:

**Option A**: Domain owner nộp Cypher thô → lưu và execute
**Option B**: Domain owner nộp JSON DSL → service compile sang Cypher

## Decision

Chọn **Option B: Query Pattern DSL**.

DSL format:
```json
{
  "start": { "node_type": "...", "match": { ... } },
  "hops": [{ "rel_type": "...", "to_node_type": "...", "filter_status": "valid_only" }],
  "return_fields": [...]
}
```

`QueryTemplateCompiler` compile DSL → Cypher, **luôn tự inject ACL filter ở mọi hop**.

## Rationale

| Criterion | Cypher raw | Query Pattern DSL |
|:---|:---|:---|
| Thêm domain mới | Phải sửa code service | Chỉ gọi Ontology API |
| Domain có thể bỏ qua ACL? | Có thể | Không — compiler always injects |
| Service biết tên label cụ thể? | Có → vi phạm P1 | Không — chỉ thấy string |
| Kiểm soát độ sâu | Khó | Limit 5 hops, enforced at registration |

## Consequences

**Positive**:
- Domain agnostic (P1): service không biết tên label/relationship cụ thể nào
- Security (P4): ACL không thể bị bypass kể cả domain owner
- Runtime extensibility: thêm domain = gọi API, không deploy

**Negative/Trade-offs**:
- DSL có expressive power hạn chế hơn Cypher (max 5 hops, không có subquery)
- Domain owner phải học DSL format (không phải Cypher quen thuộc)
- Compiler complexity (tuy nhỏ, nhưng cần maintain)

---

# ADR-003: ACL Denormalization qua `acl_visible_to`

**Status**: Accepted  
**Date**: 2026-06-17

---

## Context

Graph DB và Vector DB không có native multi-tenant Row Level Security tương đương PostgreSQL. Cần cơ chế để enforce ACL tại graph/vector layer.

Options:
- **Option A**: Roundtrip sang PG để check ACL cho mỗi node returned
- **Option B**: Denormalize ACL thành field `acl_visible_to` trên mỗi node

## Decision

Chọn **Option B: Denormalization**.

Mỗi node trong Graph DB và Vector DB có field:
```
acl_visible_to: ["tenant-A:app-X", "tenant-A:*", "platform:*"]
```

Mọi query inject filter:
```cypher
WHERE ANY(tok IN n.acl_visible_to WHERE tok IN $acl_tokens)
```

## Consequences

**Positive**:
- Filter tại graph/vector layer → không cần roundtrip sang PG
- Performance: Qdrant payload-indexed array filtering efficient
- Single source of ACL truth: PG → sync workers → denormalized field

**Negative/Trade-offs**:
- **Write amplification**: AccessGrant change → phải update `acl_visible_to` trên N nodes
- Data duplication (nhỏ, chỉ string arrays)
- Rate-limit grant changes: max 10 changes/domain/hour để tránh thrashing

---

# ADR-004: Outbox Pattern cho Cross-Store Sync

**Status**: Accepted  
**Date**: 2026-06-17

---

## Context

Write path: PostgreSQL write thành công → cần sync sang Graph DB + Vector DB. Cần đảm bảo:
- Không mất event nếu Graph/Vector tạm thời down
- Write API response không block bởi graph/vector write latency
- At-least-once delivery

Options:
- **Option A**: Sync call trong transaction (blocking)
- **Option B**: Kafka/Redis Streams message queue
- **Option C**: Outbox pattern (event table trong PG)

## Decision

Chọn **Option C: Outbox Pattern** với `kg_outbox_events` table.

```
PostgreSQL transaction:
  INSERT kg_nodes
  INSERT kg_outbox_events (atomically)
  COMMIT

→ Sync workers poll outbox
→ On success: UPDATE status = DONE
→ On failure: retry up to 5 times → DEAD_LETTER
```

## Consequences

**Positive**:
- Atomic: node insert và event tạo trong cùng transaction → không mất event
- Non-blocking: Write API returns 202 immediately
- Simple: không cần external message broker
- Idempotent: workers use MERGE + upsert

**Negative/Trade-offs**:
- Polling latency (configurable, typically 100-500ms)
- PostgreSQL là bottleneck nếu volume rất cao → Phase D: migrate sang Kafka
- Outbox table cần cleanup (DONE events xóa sau N days)

---

# ADR-005: Pluggable Backends qua Adapter Interface

**Status**: Accepted  
**Date**: 2026-06-17

---

## Context

Cần flexibility để chạy với nhiều graph/vector database backends khác nhau (dev vs production), và vendor lock-in phải tránh.

## Decision

**Adapter pattern** với interface chung:

```go
type GraphStore interface {
    MergeNode(...) error
    MergeRelationship(...) error
    QueryTemplate(...) ([]map[string]any, error)
    // ...
}
```

Implementations: `memory`, `neo4j`, `surreal` (Memgraph via Bolt), `nebula_real`.

Backend selected via `KG_RUNTIME_PROFILE` env → no code change needed.

## Consequences

**Positive**:
- Dev/test: in-memory backend → zero external deps
- Production: swap backend without code change
- Conformance tests (graphstore/conformance_test.go) verify all impls

**Negative/Trade-offs**:
- Interface must be conservative (lowest common denominator)
- Some backend-specific optimizations not exposed

---

# ADR-006: Domain-Agnostic Service với Ontology Plane

**Status**: Accepted  
**Date**: 2026-06-17

---

## Context

Yêu cầu phục vụ nhiều domain nghiệp vụ (legal, payment, HR, product...) mà không cần fork/modify service code.

## Decision

**Ontology Plane** as configuration runtime:

- Node types, relationship types → `node_type_schemas`, `rel_type_schemas` (JSONB in PG)
- Query templates → `domain_query_templates` (Pattern DSL in PG)  
- Lifecycle rules → `domain_status_field_configs` (JSONB in PG)
- Cross-domain constraints → `cross_domain_rel_rules` (JSONB in PG)

Service code contains **zero domain-specific constants**. Everything is config.

## Consequences

**Positive**:
- New domain = gọi Ontology API, không cần deploy
- Multi-tenant: mỗi tenant có domain riêng, không share schema
- StatusGate là no-op cho domain không có lifecycle (truly generic)

**Negative/Trade-offs**:
- Ontology API learning curve cho tenant admin
- Schema validation lỗi phức tạp hơn hardcode schema
- Performance: nhiều PG lookups cho ontology (mitigated by ontology caching)
