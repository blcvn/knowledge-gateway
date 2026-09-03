---
id: QA-001
title: End-to-end Integration Testing for Consolidated Architecture
service: cross-service
version: 1.0.0
status: Ready
priority: P2
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
type: Coverage
---

## Vấn Đề Chất Lượng Hiện Tại

After consolidating 35 → 18 services, need to verify:
- All gRPC service endpoints remain accessible via original proto paths
- NATS event flows function correctly with reduced subject set
- Cross-engine search (vnp-search-hub) returns results from all 6 engines
- No performance regression on critical hot paths (zep-core PutMemory sub-200ms)
- Multi-tenant isolation maintained across consolidated services

## Mục Tiêu Sau Cải Thiện

- 100% proto endpoint coverage verified
- All NATS event chains tested (17 subjects)
- Cross-engine search Recall returns ≥1 result from each engine
- PutMemory p95 ≤ 200ms
- Zero cross-tenant data leakage

## Phạm Vi Công Việc

### Test Suites

1. **Proto Endpoint Verification**: For each of 18 services, call every gRPC method, verify response format
2. **NATS Event Chain Tests**:
   - cognee-pipeline → cognee.pipeline.completed → cognee-search
   - graphiti-pipeline → graphiti.episode.completed → graphiti-search
   - memobase-pipeline → memobase.pipeline.completed → memobase-context
   - ov-storage → ov.content.written → ov-search
   - zep-core → zep.memory.messages.ingested → zep-graph → zep-search
   - sm-engine → sm.engine.document.created → sm-search
   - sm-connector → sm.connector.synced → sm-engine
   - vnp-platform → admin.tenant.created → all engines
3. **Cross-Engine Search**: vnp-search-hub.Recall() with data in all 6 engines
4. **Performance**: PutMemory benchmark (p50, p95, p99)
5. **Multi-Tenant**: Create 2 tenants, verify data isolation across all services

## Acceptance Criteria

- [ ] AC-1: All proto endpoints respond with correct status codes
- [ ] AC-2: NATS event chains complete end-to-end (all 17 subjects tested)
- [ ] AC-3: vnp-search-hub.Recall returns results from ≥ 5/6 engines
- [ ] AC-4: zep-core PutMemory p95 ≤ 200ms
- [ ] AC-5: Zero cross-tenant data leakage in any engine
- [ ] AC-6: docker-compose dev environment fully functional with 18 containers

## Không Được Làm

- Modify proto definitions
- Change business logic behavior
- Introduce new external dependencies
