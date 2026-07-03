# Design: Standardize KG Identities As UUID

## Overview

Change này đưa `kg-service` quay lại một identity model nhất quán:

- **internal canonical IDs** là UUID cho mọi entity do service sở hữu
- **external references** là khóa logic/idempotent do domain hoặc bridge cung cấp
- **backend projection IDs** không tự phát sinh format riêng, mà bám canonical UUID hoặc mapping dẫn xuất ổn định

Mục tiêu không phải chỉ sửa lỗi Qdrant hiện tại, mà là đóng lại đường workaround đã làm schema/code drift khỏi
contract gốc.

## Current Constraints

### 1. CodeGraph sync hiện dựa trên external identity ổn định

Bridge CodeGraph đang có một behavior đúng cần giữ lại: repeated sync dùng `external_ref` xác định cùng một symbol
logic. Đây là contract idempotency tốt và không nên thay bằng timestamp-based IDs.

Vấn đề nằm ở chỗ internal record IDs hiện được tạo theo chuỗi tự do thay vì UUID, khiến backend projection phải
gánh một identity format không tương thích.

### 2. Qdrant yêu cầu point id tương thích

Flow lỗi hiện tại cho thấy vector upsert fail ở `PUT /collections/kg_vectors/points` với `400 Bad Request` dù
embedding có `dims=1024` đúng collection config. Tín hiệu mạnh nhất là point id đang là chuỗi như
`node_20260622102817.068974468`, không phải UUID hay `uint64`.

### 3. Postgres schema đã bị nới kiểu dữ liệu để né lỗi runtime

Một số migration/schema gần đây đã đổi các cột như node/vector/outbox identity từ UUID sang `TEXT`. Đây chỉ là
workaround để unblock validate tạm thời, nhưng tạo lệch contract giữa:

- write path
- projection ledger
- adapters
- runtime validation/docs

## Proposed Shape

### Canonical identity contract

`kg-service` SHALL phân biệt rõ ba lớp định danh:

1. **Canonical internal ID**
   - UUID
   - dùng cho source-of-truth records, outbox aggregate IDs, replica projection IDs, read/write APIs

2. **Stable external reference**
   - string theo quy ước domain
   - dùng cho idempotent upsert và bridge lookup
   - không thay thế canonical UUID

3. **Backend-local lookup keys**
   - nếu backend chấp nhận UUID thì dùng trực tiếp canonical UUID
   - nếu cần key phụ thì key đó phải được dẫn xuất tất định từ canonical UUID, không phải timestamp string tự do

### Write path behavior

Write path phải có một nơi duy nhất sinh canonical UUID cho node/relationship/outbox aggregate identity.
Repeated upsert by `external_ref` sẽ có behavior:

- nếu chưa có record: tạo record mới với UUID mới
- nếu đã có `external_ref`: reuse UUID hiện có và update record hiện hữu

Điều này giữ được cả hai yêu cầu:

- internal IDs ổn định và hợp chuẩn
- external sync idempotent cho bridge

### Schema rollback and repair

Change này cần phục hồi những cột/service contract đã bị đổi sang `TEXT` chỉ để thích nghi với timestamp-string IDs.

Thiết kế mong muốn:

- source-of-truth identity columns quay lại UUID
- vector projection tables/ledger dùng UUID cho entity identity khi record là service-owned
- các trường đúng bản chất là external references hoặc domain payload mới giữ `TEXT`

Nếu dữ liệu tạm thời đã tồn tại dưới dạng chuỗi không phải UUID, migration/backfill phải mô tả cách chuyển đổi rõ:

- map record theo `external_ref` nếu có
- phát sinh UUID mới cho legacy rows không hợp lệ
- propagate mapping này sang relationship refs, outbox aggregate refs, projection ledger, vector docs, và sync state

### Bridge and sync updates

CodeGraph sync tool không còn được phép gửi hoặc kỳ vọng canonical IDs kiểu `node_*` / `rel_*`.

Bridge SHALL:

- dùng `external_ref` để tìm hoặc upsert logical symbol
- chấp nhận response identity từ `kg-service` như canonical UUID
- dùng canonical UUID đó khi tạo relationship endpoints hoặc khi replay/upsert tiếp theo
- lưu mapping `external_ref -> canonical UUID` trong state của sync tool nếu cần để giảm round-trips

### Backend adapters

Graph và vector adapters phải coi canonical UUID là identity chuẩn.

Đối với Qdrant:

- point id SHALL là canonical UUID hoặc một UUID-compatible representation của canonical UUID
- payload vẫn chứa `node_id`, `external_ref`, và metadata cần cho trace/debug

## Key Decisions

### 1. Fix the source contract, not only the adapter symptom

Chỉ bọc/chuyển đổi point id trong Qdrant adapter sẽ chữa triệu chứng nhưng vẫn để Postgres, outbox, và bridge chạy
trên contract identity lệch nhau. Change này chốt sửa từ source-of-truth đi ra ngoài.

### 2. Preserve `external_ref` as the idempotent key

Không dùng UUID để thay thế vai trò logical identity của CodeGraph symbol. `external_ref` vẫn là anchor cho repeated
sync, còn UUID là canonical persisted identity.

### 3. Revert temporary TEXT widening

Những thay đổi nới UUID sang `TEXT` sẽ được xem là tạm thời và phải được hoàn nguyên trong change này, cùng với code
phía trên để runtime không tái phát lỗi cũ.

## Risks

### Legacy data migration risk

Nếu local/dev stack đã có dữ liệu với ID không phải UUID, migration cần strategy backfill rõ để tránh làm gãy:

- relationship endpoints
- outbox replay
- projection versions
- vector search documents

### Bridge compatibility risk

Các script/bootstrap/validation hiện có có thể đang giả định format `node_*` trong log hoặc file state. Change này cần
đổi toàn bộ assumption đó sang UUID.

## Verification

Verification của change nên bao gồm:

1. create/update node by `external_ref` tạo hoặc reuse canonical UUID đúng cách
2. create/update relationship dùng endpoint UUID hợp lệ
3. outbox/projection ledger giữ UUID nhất quán
4. CodeGraph sync rerun không tạo duplicate logical node
5. Qdrant upsert thành công với runtime `qdrant-memgraph`
6. runtime validation `get`, `search`, `index`, và template traversal đều pass sau sync
