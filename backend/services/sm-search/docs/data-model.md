---
id: DOC-S04
service: sm-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-search — Data Model

> **Database**: PostgreSQL + pgvector (read from sm-document and sm-memory tables)

## Search Indexes (on shared tables)

### Document Chunk Search (v3)

Uses `chunks` table from sm-document:

| Index | Type | Config | Purpose |
|-------|------|--------|---------|
| idx_chunk_embedding | HNSW | lists=100, m=16, ef_construction=200 | Primary vector search |
| idx_chunk_matryoshka | HNSW | lists=50 | Fast pre-filter (256-dim) |
| idx_chunk_fulltext | GIN | tsvector(content) | PostgreSQL fulltext search |
| idx_chunk_doc_id | B-tree | document_id | Doc-scoped search |

### Memory Search (v4)

Uses `memory_entries` table from sm-memory:

| Index | Type | Config | Purpose |
|-------|------|--------|---------|
| idx_memory_embedding | HNSW | lists=100, m=16 | Memory vector search |
| idx_memory_space_latest | B-tree | (space_id, is_latest, is_forgotten) | Active memory filter |

### Document Summary Search

Uses `documents` table from sm-document:

| Index | Type | Purpose |
|-------|------|---------|
| idx_doc_summary_emb | HNSW | Document-level similarity |
| idx_doc_metadata | GIN | Metadata filter queries |

## Search Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| HNSW ef_search | 64 | Search quality vs speed tradeoff |
| RRF k-constant | 60 | Reciprocal rank fusion constant |
| Context window | ±1 chunk | Adjacent chunks for context |
| Max results | 100 | Hard limit per search |

## Note

sm-search is **stateless** for its own data — it queries shared tables owned by sm-document and sm-memory. No sm-search-specific tables exist.
