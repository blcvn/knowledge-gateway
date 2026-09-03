---
id: TASK-STO-004
title: Neo4j Edge Repository — Bi-Temporal Model
feature: FEAT-STO-004
status: Done
---

## Objective
Thực thi implement Neo4j edge repository với bi-temporal model, quản lý valid_at, invalid_at, và expired_at properties dựa trên FEAT-STO-004.

## Tasks
1. Tạo file `internal/adapter/driver/neo4j/edge_repo.go`.

2. Cài đặt các methods:
   - `SaveEdge`: sử dụng Cypher CREATE để thiết lập `RELATES_TO` relationship giữa Entity nodes và `MENTIONS` giữa Episodic → Entity. Cấu hình đúng các temporal properties (valid_at, invalid_at, expired_at, created_at).
   - `GetEdge`: truy vấn thông tin edge.
   - `DeleteEdge`: thực hiện xóa edge.
   - `InvalidateEdge`: set trường `invalid_at` mà không delete physical edge.
   - `GetEdgesInTimeRange`: tìm các edges theo temporal range queries với window intersection (`valid_at <= to` AND (`invalid_at IS NULL` OR `invalid_at >= from`)).

3. Điều kiện bắt buộc cho Repository:
   - All edge queries bắt buộc phải scope theo `group_id`.
   - Edge với `invalid_at = NULL` coi như "currently valid". `expired_at` là do newer version tồn tại.

4. Integration Tests:
   - Viết các temporal edge scenarios sử dụng Neo4j testcontainer.
   - Đảm bảo coverage >= 80%.
