---
id: TASK-STO-006
title: Neo4j Bulk Operations + Transaction Management
feature: FEAT-STO-006
status: Done
---

## Objective
Thực thi implement atomic bulk operations và transaction management wrapper cho Neo4j driver dựa trên FEAT-STO-006.

## Tasks
1. Tạo file `internal/adapter/driver/neo4j/transaction.go`:
   - Implement method `WithTransaction` dùng làm wrapper. Hỗ trợ auto-rollback khi gặp lỗi và commit khi success. Phải hỗ trợ nested operations với single commit point.

2. Tạo file `internal/adapter/driver/neo4j/bulk_repo.go`:
   - Cài đặt method `SaveBulk`: Bọc bên trong `WithTransaction`. Thực thi tuần tự (1) creates/updates entity nodes, (2) creates episode node, (3) creates RELATES_TO edges, (4) creates MENTIONS edges (episode → entities).
   - Cài đặt method `RollbackBulk`: Xóa tất cả artifact (MENTIONS, RELATES_TO, episode node) tạo ra bởi specific episode ID.
   - Cài đặt method `DeleteByGroupID`: Thực thi purge (xóa SẠCH toàn bộ data cho một tenant/group_id).

3. Integration Tests:
   - Dùng Neo4j testcontainer. Bơm failure (failure injection) để kiểm thử chức năng auto-rollback.
   - Đảm bảo coverage >= 80%.
