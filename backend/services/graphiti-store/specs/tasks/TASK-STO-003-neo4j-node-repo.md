---
id: TASK-STO-003
title: Neo4j Node Repository Adapter
feature: FEAT-STO-003
status: Done
---

## Objective
Thực thi implement Neo4j node repository adapter cho các loại node types bằng cách sử dụng Neo4j Go driver v5 và Cypher queries, dựa trên FEAT-STO-003.

## Tasks
1. Tạo file `internal/adapter/driver/neo4j/node_repo.go`.

2. Cài đặt các methods:
   - `SaveNode`: dùng Cypher MERGE để upsert EntityNode, EpisodicNode, CommunityNode, SagaNode. Set properties đầy đủ.
   - `GetNode`, `GetNodeByName`: truy vấn node theo UUID hoặc Name.
   - `DeleteNode`: xóa node và toàn bộ relationships liên quan của nó.
   - `ListNodes`: liệt kê các nodes có hỗ trợ cursor-based pagination.

3. Điều kiện bắt buộc cho Repository:
   - Phải sử dụng parameterized Cypher queries để chống injection.
   - Multi-tenant scoping: Mọi queries bắt buộc phải có điều kiện filter `group_id`.
   - Metrics: Expose connection pool metrics cho Prometheus.

4. Integration Tests:
   - Cài đặt tests kết nối đến Neo4j testcontainer.
   - Đảm bảo coverage >= 80%.
