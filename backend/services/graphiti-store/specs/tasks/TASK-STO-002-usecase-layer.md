---
id: TASK-STO-002
title: Usecase Layer + Port Interfaces
feature: FEAT-STO-002
status: Done
---

## Objective
Thực thi implement usecase layer orchestration và port interfaces cho graphiti-store, sử dụng duy nhất một output port là GraphDriver interface, dựa trên FEAT-STO-002.

## Tasks
1. Tạo file `internal/usecase/node_ops.go`:
   - Implement `SaveNode`, `GetNode`, `DeleteNode`, `ListNodes`.

2. Tạo file `internal/usecase/edge_ops.go`:
   - Implement `SaveEdge`, `GetEdge`, `DeleteEdge`, `InvalidateEdge`, `GetEdgesInTimeRange`.

3. Tạo file `internal/usecase/community_ops.go`:
   - Implement `SaveCommunity`, `GetCommunity`, `DeleteCommunity`.

4. Tạo file `internal/usecase/bulk_ops.go`:
   - Implement `SaveBulk` (atomic) để gộp toàn bộ logic lưu trữ nodes, edges và episode node vào 1 transaction của driver.
   - Implement `RollbackBulk` để xóa toàn bộ artifacts tạo ra từ 1 episode.
   - Implement `DeleteByGroupID` để xóa toàn bộ tenant data.

5. Tạo file `internal/usecase/search_ops.go`:
   - Implement `CosineSimilarity`, `Fulltext`, `BFS` (chỉ delegate gọi driver, không xử lý business logic).

6. Tạo file `internal/usecase/index_ops.go`:
   - Implement `BuildIndices`, `DropIndices`, `ListIndices`.

7. Port interfaces & DTOs:
   - Viết các request/response mapping DTOs cho các usecase.
   - Implement logic validate input trước khi delegate cho driver.

8. Unit Tests:
   - Viết unit tests cho tất cả các usecases, dùng mock GraphDriver.
   - Đảm bảo coverage >= 80%.
