---
id: TASK-STO-007
title: Neo4j Index Management
feature: FEAT-STO-007
status: Done
---

## Objective
Thực thi implement index management adapter cho Neo4j, bao gồm vector indexes, fulltext indexes, composite indexes, và range indexes cho bi-temporal queries dựa trên FEAT-STO-007.

## Tasks
1. Tạo file `internal/adapter/driver/neo4j/index_repo.go`.

2. Cài đặt các methods quản lý indexes:
   - `BuildIndices`: Tạo idempotently (IF NOT EXISTS) 6 loại indexes:
     - `entity_name_embedding` (Vector/cosine, node: Entity, field: name_embedding)
     - `edge_fact_embedding` (Vector/cosine, edge: RELATES_TO, field: fact_embedding)
     - `entity_name_fulltext` (Fulltext, node: Entity, fields: name, summary)
     - `edge_fact_fulltext` (Fulltext, edge: RELATES_TO, fields: name, fact)
     - `entity_group_id` (Range, node: Entity, field: group_id)
     - `edge_temporal` (Composite, edge: RELATES_TO, fields: group_id, valid_at, invalid_at)
   - Bảm đảm vector index dùng đúng dimension cấu hình trong config.
   - `DropIndices`: Xóa bỏ toàn bộ indexes định nghĩa cho group đó để dọn dẹp (cleanup).
   - `ListIndices`: Trả về danh sách current index definitions đang được cài.

3. Integration Tests:
   - Viết test tạo và xoá index qua Neo4j testcontainer.
   - Đảm bảo coverage >= 70%.
