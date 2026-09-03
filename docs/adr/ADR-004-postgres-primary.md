# ADR-004 — PostgreSQL + pgvector làm Primary Data Store

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-01 |
| **Deciders** | Platform Team |
| **Feature** | Tất cả memory engines, F10 (Hybrid Search) |

---

## Context

VNP Memory cần lưu trữ:
1. Relational entities (users, tenants, sessions, memories, profiles)
2. Vector embeddings (semantic search)
3. Audit trail (immutable logs)
4. Multi-tenant isolation

Câu hỏi: **Bao nhiêu database systems là đủ?** Minimize infrastructure complexity.

---

## Decision

**PostgreSQL 17 + pgvector extension làm primary data store cho tất cả relational + vector data.**

Neo4j được dùng thêm chỉ cho graph traversal (knowledge graphs).

```sql
-- pgvector: vector similarity search in PostgreSQL
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE memories ADD COLUMN embedding vector(1536);

-- Cosine similarity search (pgvector)
SELECT id, content, 1 - (embedding <=> $1) AS score
FROM memories
WHERE tenant_id = $2 AND user_id = $3
ORDER BY embedding <=> $1
LIMIT 10;

-- IVFFlat index for approximate nearest neighbor
CREATE INDEX ON memories USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
```

**TenantID trên mọi table** — không dùng schema separation:

```sql
-- Every table has tenant_id
CREATE TABLE memories (
    tenant_id TEXT NOT NULL,
    -- ... other columns
);

-- Mandatory composite index
CREATE INDEX ON memories(tenant_id, user_id);
-- Row-level security (alternative considered but not chosen — see Alternatives)
```

---

## Consequences

**Positive:**
- **1 database system** thay vì postgres + separate vector DB → giảm operational complexity
- **ACID transactions** cho critical operations (API key creation, tenant isolation)
- **pgvector** performance đủ tốt cho scale hiện tại (< 1M vectors per tenant)
- **Familiar tooling:** PostgreSQL ecosystem (pg_dump, pgAdmin, monitoring)
- **pgvector IVFFlat index** giảm search time từ O(n) → O(sqrt(n))

**Negative:**
- pgvector không scale bằng Qdrant cho extremely large vector datasets (> 100M vectors)
- Qdrant cần được thêm vào khi tenant scale lớn
- pgvector cosine search chậm hơn Qdrant's HNSW ~3-5x tại scale

**Mitigation:**
- Qdrant là optional secondary vector store (configurable)
- Migration path documented khi pgvector không đủ

---

## Alternatives Considered

### A1 — Separate Vector DB (Qdrant từ đầu)
- **Rejected:** Thêm 1 database để operate; pgvector đủ cho current scale; premature optimization

### A2 — Row-Level Security (PostgreSQL RLS)
- **Considered:** RLS tự động enforce tenant isolation
- **Rejected:** Performance overhead với nhiều tenant (RLS check mỗi row); complexity khi debug; khó override trong admin operations

### A3 — Separate schema per tenant
- **Rejected:** Schema explosion với nhiều tenants; migration khó; cross-tenant query impossible
