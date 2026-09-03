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

### Option B: Dual-deployment with 18 consolidated services (selected)
- **Pro**: Flexible deployment (compact for dev, scale for prod), shared domain code, unified vector interface
- **Con**: 2 deployment configs to maintain, migration effort ~8 weeks

### Option C: Consolidate to 6 "mega-services" (one per engine)
- **Pro**: Maximum simplification
- **Con**: Loses independent scaling (search vs ingestion), monolith anti-pattern within engines

## Decision

**Option B — Build 18 consolidated services alongside 35 individual services** using 4 merge patterns:

| Pattern | Description | New Services |
|---------|-------------|-------------------|
| Pipeline Merge | Ingestion + Processing → single binary | cognee-pipeline, graphiti-pipeline, memobase-pipeline |
| Functional Merge | Tightly coupled CRUD → single binary | zep-core, ov-storage, sm-engine |
| Platform Unification | Admin/Auth/Event → single platform service | vnp-platform |
| Gateway Absorption | sm-mcp tools → vnp-gateway | (integrated) |

**Deployment**: Two modes — `compact` (18 consolidated) for dev/staging, `scale` (35 individual) for production.

**Vector Storage**: Dual backends — Qdrant (Cognee high-throughput ANN) + pgvector (all other engines, co-located with metadata).

## Consequences

### Positive
- Flexible deployment: compact mode for dev/staging, scale mode for production
- Shared domain code between consolidated and individual services
- Eliminated network hops within pipelines (compact mode)
- Single admin/auth service for all engines (vnp-platform)
- Proto backward compatibility maintained (multiple gRPC services per binary)
- Unified VectorStore interface (Qdrant + pgvector)

### Negative
- 2 deployment configurations to maintain (docker-compose.consolidated + docker-compose)
- Merged services require bulkhead pattern for LLM isolation in compact mode
- Slightly more complex CI/CD (build both modes)

### Risks
- Performance regression in merged services → mitigate with bulkhead + benchmarking
- Configuration drift between modes → mitigate with shared env config and E2E tests for both modes
