# Proposal: Version Logical Knowledge Graphs

## Problem

Hiện tại `kg-service` đã có một số lớp versioning, nhưng chúng đang nằm ở sai abstraction level cho bài toán
quản lý version của cả một logical knowledge graph:

- write path có `graph_scope` dẫn xuất từ `graph_id` / `repo_id` / `project_id`, nhưng đây mới là string suy
  diễn chứ chưa phải một thực thể versionable hạng nhất;
- source records dùng `domain_version` theo entity, còn worker ledger dùng `kg_projection_versions` theo
  entity-replica, nên chưa biểu diễn được "version số mấy của cả graph này";
- outbox hiện kích hoạt projection theo mutation/event của từng entity, chưa có event ở mức `graph version`
  để backend replicas bám theo;
- `graphdb` và `vectordb` hiện đang mang khá nhiều payload lặp lại từ `relationshipdb`, khiến rollback,
  archival, và truy xuất version lịch sử của whole-graph trở nên nặng và khó kiểm soát;
- riêng FTS PostgreSQL hiện thực tế đã đọc trực tiếp từ `kg_nodes`, cho thấy boundary "source giữ content,
  projection giữ index" là hướng khả thi nhưng chưa được chuẩn hóa thành contract chung.

Hệ quả là:

- khó trả lời chính xác một `tenant/app` đang có những knowledge graph nào và mỗi graph đang ở version nào;
- khó gắn một lần sync với một `reference_id` rõ ràng như commit SHA, import batch id, hay external snapshot id;
- khó replay, archive, hoặc khôi phục phiên bản cũ của một graph mà không quét lại toàn bộ source data;
- projection pipeline vẫn đúng ở mức entity, nhưng chưa có orchestration chuẩn ở mức whole-graph revision.

## Proposed Solution

Đưa logical knowledge graph trở thành first-class object có định danh ổn định và có version chain riêng.

1. **Thêm first-class graph identifier cho mỗi logical KG**
   - mỗi KG thuộc một cặp `owner_tenant_id` / `owner_app_id` SHALL được gắn với đúng một identifier ổn định;
   - identifier này có thể được caller cung cấp dưới dạng `project_id`, nhưng service SHALL persist một
     `identifier_id` canonical để dùng nội bộ và làm khóa versioning;
   - repeated writes/syncs của cùng logical graph SHALL reuse cùng identifier đó.

2. **Thêm graph version chain gắn với `reference_id`**
   - mỗi lần sync hoặc write commit SHALL tạo ra một graph version mới có `version_number` tăng đơn điệu;
   - version mới SHALL liên kết với `reference_id` để trace về commit, snapshot, import batch, hay nguồn khác; nếu caller không cung cấp, service SHALL tự generate một `reference_id` và trả về cho caller;
   - các entity bị ảnh hưởng SHALL được link tới graph version mới như một change set.

3. **Đổi durable handoff từ entity-only event sang graph-version-aware event**
   - write path vẫn commit source rows vào `relationshipdb`, nhưng đồng thời SHALL seal một graph version;
   - outbox/event pipeline SHALL cho phép worker sync theo `graph version` và change set của version đó;
   - backend projection heads SHALL theo dõi version đã áp dụng cho từng graph thay vì chỉ nhìn entity rời rạc.

4. **Chuẩn hóa lưu trữ version theo hai tier online/offline**
   - version mới và recent history SHALL ở trạng thái `ONLINE` để đọc/replay ngay;
   - version cũ hơn MAY được archive sang `OFFLINE` storage class, nhưng metadata và pointer truy hồi SHALL
     luôn còn trong hệ thống;
   - restore từ offline version SHALL chậm hơn nhưng không làm mất lineage.

5. **Chuẩn hóa boundary dữ liệu giữa source và projection stores**
   - `relationshipdb` SHALL là canonical content store cho node/relationship payload đầy đủ;
   - `graphdb` SHALL ưu tiên giữ topology, replica version, filter attrs, và source pointers;
   - `vectordb` SHALL ưu tiên giữ embeddings, filter attrs, sync version, và source pointers;
   - FTS PostgreSQL hiện có SHALL được xem là baseline cho mô hình "index/read từ source", còn nếu dùng FTS
     backend riêng thì backend đó cũng chỉ nên giữ search index cùng source pointer thay vì full canonical payload.

## Scope

### In scope

- logical KG identity theo `tenant/app + project_id/identifier_id`
- graph version numbering và `reference_id`
- write/sync handoff từ source rows sang version-aware outbox/projection events
- online/offline storage policy cho graph versions
- content boundary giữa `relationshipdb`, `graphdb`, `vectordb`, và FTS
- migration/rollout plan ở mức contract và data model

### Out of scope

- implement ngay UI hay admin workflow để duyệt version history
- redesign ontology/domain schema của node và relationship
- thay đổi ranking algorithm của semantic/hybrid search
- chọn cụ thể object storage vendor cho offline archive
- full historical query language cho "time travel graph traversal"

## Success Criteria

- hệ thống có thể định danh ổn định từng logical KG của một `tenant/app`
- mỗi sync/write có thể truy ra `graph identifier`, `version_number`, và `reference_id`
- worker/projection pipeline có contract rõ ràng để sync theo graph version lineage
- version history có policy rõ cho `ONLINE` và `OFFLINE`
- boundary "source giữ canonical content, projections giữ topology/index/pointer" được chốt đủ rõ để triển khai
