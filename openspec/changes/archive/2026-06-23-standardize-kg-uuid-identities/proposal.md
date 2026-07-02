# Proposal: Standardize KG Identities As UUID

## Problem

Trong quá trình validate runtime `CodeGraph -> kg-service`, flow sync hiện tạo `node_id`, `relationship_id`,
`outbox event aggregate_id`, và một số ID phụ trợ ở dạng chuỗi tự do như `node_20260622102817...`.

Điều này gây lệch contract giữa các lớp:

- `Qdrant` chỉ chấp nhận point id kiểu `uint64` hoặc `UUID`, nên projection vector trả `400 Bad Request`
  dù embedding hợp lệ
- một số migration/schema gốc của `kg-service` được thiết kế quanh UUID nhưng gần đây đã bị nới lỏng sang
  `TEXT` để tạm thời đi vòng lỗi runtime
- bridge, sync ledger, và ref lookup đang pha trộn giữa:
  - internal service identity
  - external reference identity
  - backend-specific projection identity

Hệ quả là hệ thống chạy được một phần nhưng mất tính nhất quán về định danh:

- Postgres schema không còn phản ánh contract ban đầu
- graph/vector adapters phải xử lý nhiều loại ID khác nhau
- sync/debug khó phân biệt đâu là canonical ID, đâu là stable external reference
- các workaround `TEXT` làm tăng nguy cơ drift và lỗi tương thích backend về sau

## Proposed Solution

Chuẩn hóa lại toàn bộ identity contract của KG quanh UUID:

1. **Canonical UUID identities**
   - mọi `node.id`, `relationship.id`, `aggregate_id`, và các service-owned identity liên quan SHALL là UUID
   - ID được sinh theo một cơ chế chuẩn và nhất quán trong write path thay vì chuỗi timestamp tự do

2. **Stable external references remain external**
   - `external_ref` tiếp tục là khóa domain-level/idempotent cho CodeGraph sync
   - bridge dùng `external_ref` để upsert logic node/relationship
   - internal persisted ID không còn bị overload làm backend point id hay domain ref

3. **Revert temporary TEXT widening**
   - phục hồi lại các cột/migration/schema vừa bị đổi từ UUID sang `TEXT`
   - sửa code bootstrap/sync/write path để luôn cấp UUID hợp lệ thay vì nới kiểu dữ liệu

4. **Backend identity normalization**
   - graph store dùng canonical UUID làm node/relationship identity
   - vector store dùng canonical UUID-compatible point identity
   - mọi ref payload tiếp tục giữ `external_ref` và `node_id`/`relationship_id` tương ứng để truy vết

## Scope

### In scope

- Chuẩn hóa service-owned IDs về UUID trong write path, outbox, projection ledger, bridge sync state
- Revert các thay đổi schema/code gần đây đổi UUID sang `TEXT`
- Cập nhật CodeGraph sync/bootstrap để resolve hoặc sinh canonical UUID đúng contract
- Cập nhật test, validation scripts, và runtime verification theo contract UUID mới

### Out of scope

- thay đổi semantics domain của `external_ref`
- thay Qdrant bằng backend khác
- đổi shape API ngoài phạm vi identity contract cần thiết

## Success Criteria

- Runtime sync `CodeGraph -> kg-service` upsert thành công vào cả graph store và Qdrant mà không cần workaround `TEXT`
- Canonical `node_id` và `relationship_id` trong source-of-truth là UUID hợp lệ
- `external_ref` vẫn giữ vai trò idempotent logical key cho repeated sync
- Các migration/schema tạm thời nới `TEXT` được phục hồi về contract UUID nhất quán
- Validation scripts và integration tests xác nhận được get/search/index hoạt động trên contract identity mới
