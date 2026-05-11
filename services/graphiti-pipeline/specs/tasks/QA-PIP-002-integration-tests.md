---
id: QA-PIP-002
title: Saga End-to-End Integration Tests
feature: SOL-001
status: Done
---

## Objective
Xác nhận tính toàn vẹn của nghiệp vụ và tích hợp hệ thống qua end-to-end integration tests (Saga).

## Tasks
1. Môi trường Test
   - Setup Docker Compose `docker compose up graphiti-pipeline` cho E2E environment (Postgres, Neo4j, NATS, Bifrost Mock).
   - Tích hợp testcontainers cho Golang tests.

2. Test Saga Hoàn Chỉnh (Happy Path)
   - Gọi `IngestEpisode` RPC.
   - Kiểm tra các bước thực thi thành công: extraction, resolution, embedding, lưu database và publish NATS event `graphiti.episode.ingested`.

3. Test Saga Compensation (Failure Path)
   - Giả lập lỗi ở bước `SaveBulk`.
   - Kiểm tra `RollbackBulk` được gọi và state được khôi phục chính xác.
   - Kiểm tra NATS publish sự kiện `graphiti.episode.failed`.

4. Test Edge Cases
   - Concurrent episodes với cùng `group_id` bị serialize chính xác (GroupLock).
   - Bulk ingestion xử lý dedup và retry.
   - LLM transient errors bị retry thành công (Circuit Breaker / Bulkhead).
