---
id: TASK-PIP-006
title: Implement NATS Event Publisher Adapter
feature: FEAT-PIP-006
status: Done
---

## Objective
Thực thi implement NATS JetStream publisher adapter dựa trên FEAT-PIP-006.

## Tasks
1. Tạo file `internal/adapter/event/nats_publisher.go`
   - Implement `EventPublisher` port.
   - Cấu hình NATS JetStream stream (`graphiti` stream).

2. Implement event publishing
   - Publish `graphiti.episode.ingested` khi Saga hoàn thành.
   - Publish `graphiti.entity.resolved` khi merge entities.
   - Publish `graphiti.community.rebuilt` sau khi cập nhật community.
   - Đảm bảo payloads chứa `group_id`.
   - Serialize bằng Protobuf hoặc JSON.

3. Resiliency
   - Retry logic cho transient NATS failures (3x exponential backoff).
   - Deduplication qua NATS dedup window.

4. Tests
   - Viết unit tests cho publisher sử dụng mock JetStream context.
   - Viết integration tests cho NATS publish/subscribe roundtrip (testcontainer).
   - Đảm bảo coverage >= 80%.
