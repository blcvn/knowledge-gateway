---
id: TASK-SEA-005
title: Implement Redis Cache + NATS Invalidation
feature: FEAT-SEA-005
status: Done
---

## Objective
Thực thi implement cache adapter với Redis và NATS listener dựa trên FEAT-SEA-005.

## Tasks
1. Tạo file `internal/adapter/cache/redis_cache.go`
   - Implement `CacheRepo` interface với Redis client.
   - Implement key creation scheme: `search:{group_id}:{sha256(query+methods+rerankers)}`.
   - Sử dụng Protobuf encoding/decoding lưu cache.
   - Thiết lập grace degradation khi Redis không available.

2. Tạo file `internal/adapter/event/nats_subscriber.go`
   - Khởi tạo listener NATS bắt event `graphiti.episode.ingested`.
   - Xóa bỏ các cache keys (`search:{group_id}:*`) cho `group_id` tương ứng khi nhận sự kiện.

3. Observability
   - Thêm cache hit ratio metric push về Prometheus.

4. Integration Tests
   - Test với Redis testcontainer và NATS testcontainer.
   - Coverage >= 80%.
