# ADR-0001: Service Consolidation — 35 → 18 Services

- **Date**: 2026-05-10
- **Status**: Proposed
- **Deciders**: Software Architect, Tech Lead
- **Linked**: `docs/service-consolidation-proposal.md`, `docs/service-compatibility-matrix.md`

## Context

VNP Memory monorepo hiện có 35 domain services + 1 gateway, hợp nhất từ 6 engine (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory) + 3 platform services. Phân tích compatibility matrix cho thấy overlap đáng kể ở 6 functional areas (entity extraction, vector search, reranking, admin/auth, profile management, ingestion pipeline). Số lượng service quá lớn gây:

- **Operational overhead**: 35 Docker containers, 35 gRPC ports, 7 NATS streams / 29 subjects
- **Development friction**: tight gRPC coupling giữa services trong cùng engine (ov-fs ↔ ov-crypto, zep-memory → zep-thread)
- **Infrastructure cost**: 6 database backends (PostgreSQL, Neo4j, Qdrant, Redis, MinIO, VikingFS)

## Considered Options

### Option A: Keep 35 services (status quo)
- **Pro**: No migration effort, maximum isolation
- **Con**: Operational overhead, tight coupling hidden behind gRPC, 6 separate admin services doing same thing

### Option B: Consolidate to 18 services (selected)
- **Pro**: 48.6% reduction, eliminates tight gRPC coupling, unifies admin/auth, drops Qdrant
- **Con**: Larger binaries, more complex internal structure, migration effort ~8 weeks

### Option C: Consolidate to 6 "mega-services" (one per engine)
- **Pro**: Maximum simplification
- **Con**: Loses independent scaling (search vs ingestion), monolith anti-pattern within engines

## Decision

**Option B — Consolidate to 18 services** using 4 merge patterns:

| Pattern | Description | Services Eliminated |
|---------|-------------|-------------------|
| Pipeline Merge | Ingestion + Processing → single binary | 3 (cognee, graphiti, memobase) |
| Functional Merge | Tightly coupled CRUD → single binary | 6 (zep, ov, sm) |
| Platform Unification | Admin/Auth/Event → single platform service | 6 (ov-admin, zep-admin, sm-auth, sm-analytics, sm-project, vnp-event) |
| Gateway Absorption | sm-mcp → vnp-gateway | 1 |

**Infrastructure**: Drop Qdrant, migrate Cognee embeddings to pgvector (already used by 4 other engines).

## Consequences

### Positive
- 48.6% fewer services to deploy, monitor, maintain
- 41% fewer NATS subjects — simpler event topology
- Eliminated network hops within pipelines
- Single admin/auth service for all engines
- Proto backward compatibility maintained (multiple gRPC services per binary)

### Negative
- 8-week migration effort across 4 phases
- Qdrant → pgvector migration needs benchmarking
- Merged services require bulkhead pattern for LLM isolation
- Temporary dual-routing during transition

### Risks
- Performance regression in merged services → mitigate with bulkhead + benchmarking
- Memory footprint increase → mitigate with shared connection pools (net reduction expected)
