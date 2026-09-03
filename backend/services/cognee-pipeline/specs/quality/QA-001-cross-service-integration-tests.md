---
id: QA-001
title: Cross-Service Integration Tests — Cognee Domain
service: cross-service
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
type: Coverage
---

## Vấn Đề Chất Lượng Hiện Tại

No integration tests exist across Cognee domain services. Need to verify end-to-end data flow: ingestion → cognify → search.

## Mục Tiêu Sau Cải Thiện

- E2E flow verified: upload file → cognify → search returns results
- NATS event flow: `cognee.data.ingested` → `cognee.pipeline.completed`
- Multi-tenant isolation: tenant A data invisible to tenant B
- Graceful error handling across service boundaries

## Phạm Vi Công Việc

### Test Scenarios

1. **Happy Path E2E**: Upload PDF → auto-cognify → search by entity name
2. **NATS Event Flow**: Verify ingestion publishes event → cognify subscribes and processes
3. **Pipeline Resume**: Kill cognify mid-pipeline → restart → verify job resumes
4. **Multi-Tenant Isolation**: Ingest data as tenant A → search as tenant B → zero results
5. **Error Propagation**: Upload unsupported file → verify error propagates to gateway
6. **Consolidated Mode**: Same tests against cognee-pipeline (single binary)

### Test Infrastructure

- Docker Compose with all services + infra (PostgreSQL, Neo4j, MinIO, NATS)
- Testcontainers for isolated per-test environments
- Mock Bifrost LLM with deterministic responses

## Acceptance Criteria

- [ ] AC-1: E2E happy path passes with real services (Docker Compose)
- [ ] AC-2: NATS event flow verified with message trace
- [ ] AC-3: Tenant isolation confirmed (zero cross-tenant leakage)
- [ ] AC-4: Pipeline resume works after service restart
- [ ] AC-5: Both standalone and consolidated modes pass same test suite

## Không Được Làm

- Do not modify service business logic
- Do not bypass gRPC interfaces (test via public APIs only)
