---
id: TASK-SEA-008
title: QA Tests (Unit & Integration)
feature: QA-SEA-001
status: Done
---

## Objective
Thực thi quality assurance, unit test và integration tests cho hybrid search service dựa trên SOL-001 và QA-SEA-001.

## Tasks
1. Integration Test Flow
   - Verify gRPC HybridSearch trả về kết quả combine (cosine, bm25, bfs).
   - Kiểm tra reranking results hợp lệ.
   - Validate caching < 10ms đối với cache hit.
   - Validate invalidation logic qua NATS.

2. Traces & Metrics
   - Đảm bảo metrics Prometheus và OTel traces output chính xác.

3. Tenant Isolation
   - Test query từ Tenant A không overlap với Data của Tenant B.

4. Test Coverage Validation
   - Chạy `go test -cover` đảm bảo overall project >= 80%.
