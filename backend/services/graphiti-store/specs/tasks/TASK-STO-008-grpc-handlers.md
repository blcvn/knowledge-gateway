---
id: TASK-STO-008
title: gRPC Handler Adapters
feature: FEAT-STO-008
status: Done
---

## Objective
Thực thi implement gRPC handler adapters cho GraphitiStoreService (15 RPCs) và Proto-to-Domain mapping logic dựa trên FEAT-STO-008.

## Tasks
1. Tạo file `internal/adapter/grpc/handler.go`:
   - Khai báo và implement interface GraphitiStoreService cho 15 RPCs:
     - Node: SaveNode, GetNode, DeleteNode
     - Edge: SaveEdge, GetEdge, DeleteEdge, InvalidateEdge
     - Bulk: SaveBulk, RollbackBulk, DeleteByGroupID
     - Search: CosineSimilaritySearch, FulltextSearch, BFSSearch
     - Index: BuildIndices, DropIndices
   - Tất cả các handlers phải delegate sang usecase layer (không làm business logic ở đây).
   - Extract thông tin tenant từ gRPC metadata (`x-tenant-id`) và gắn vào dưới dạng `GroupID`.
   - Cấu hình khởi tạo 1 OTel span cho mỗi request RPC tới.
   - Trả về đúng mã gRPC status: `NOT_FOUND` (nếu thiếu), `INVALID_ARGUMENT` (nếu input hỏng).

2. Tạo file `internal/adapter/grpc/mapper.go`:
   - Code các mapping logic 2 chiều giữa Proto model và Domain entity. Mapper phải lossless (không mất mát properties).

3. Unit Tests:
   - Viết test handler với mock usecase layer.
   - Viết test round-trip cho proto-domain mapper.
   - Đảm bảo coverage >= 80%.
