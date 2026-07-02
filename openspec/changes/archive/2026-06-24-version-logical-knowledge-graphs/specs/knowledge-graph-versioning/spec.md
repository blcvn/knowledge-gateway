# knowledge-graph-versioning

## ADDED Requirements

### Requirement: Mỗi logical knowledge graph SHALL có một canonical identifier ổn định

Hệ thống SHALL quản lý mỗi logical knowledge graph của một cặp `owner_tenant_id` / `owner_app_id` bằng một
identifier ổn định. Caller MAY cung cấp `project_id` hoặc external graph key, nhưng service SHALL persist một
`identifier_id` canonical để dùng cho versioning và projection lineage.

#### Scenario: Repeated sync của cùng project reuse cùng graph identifier

- GIVEN một ingest pipeline sync dữ liệu cho `tenant-a`, `app-b`, và external key `project-x`
- AND lần sync đầu đã resolve thành `identifier_id = kg-123`
- WHEN pipeline sync lại cùng logical graph đó ở lần sau
- THEN service SHALL reuse `identifier_id = kg-123`
- AND service SHALL NOT tạo một logical graph mới chỉ vì phát sinh thêm entity mutations

#### Scenario: Hai logical graphs khác nhau trong cùng tenant/app không bị nhập nhầm

- GIVEN cùng một `tenant/app` có hai graphs khác nhau với external keys `project-a` và `project-b`
- WHEN hai graph này có node trùng tên hoặc trùng external references ở mức domain
- THEN service SHALL quản lý chúng bằng hai graph identifiers khác nhau
- AND projection lineage của graph này SHALL NOT ghi đè head version của graph kia

#### Scenario: Không có external key ổn định thì service không được tự động merge graph mới theo fallback mơ hồ

- GIVEN một caller ghi dữ liệu mới cho `tenant-a` / `app-b`
- AND request không mang `graph_id`, `_kg_graph_scope`, `project_id`, hay external graph key ổn định nào
- WHEN service cần resolve logical graph identifier
- THEN service SHALL NOT tự động suy ra rằng graph này phải reuse một graph hiện có chỉ dựa trên fallback `domain + tenant + app`
- AND service SHALL either cấp một `identifier_id` mới hoặc từ chối request theo contract của API

#### Scenario: Caller không có external key nhưng reuse canonical identifier đã được cấp trước đó

- GIVEN lần ghi đầu cho một logical graph không có `project_id` đã được service cấp `identifier_id = kg-123`
- WHEN caller gửi lần ghi sau và chỉ định lại canonical identifier đó theo contract được hỗ trợ
- THEN service SHALL reuse `identifier_id = kg-123`
- AND service SHALL NOT tạo thêm một logical graph mới

### Requirement: Mỗi write hoặc sync operation SHALL seal một graph version mới

Mỗi operation làm thay đổi một logical knowledge graph SHALL tạo ra một graph version mới với
`version_number` tăng đơn điệu trong phạm vi graph đó. Graph version SHALL luôn có `reference_id`
không null: caller MAY cung cấp giá trị (commit SHA, import batch id, snapshot id), hoặc nếu không
cung cấp thì service SHALL generate một `reference_id` và trả về cho caller trong response.

#### Scenario: Bulk sync gắn graph version với commit SHA

- GIVEN một code graph sync cho `identifier_id = kg-123`
- AND pipeline cung cấp `reference_id = git:7f3ab21`
- WHEN sync commit source mutations thành công
- THEN service SHALL tạo một graph version mới cho `kg-123`
- AND graph version đó SHALL lưu `reference_id = git:7f3ab21`
- AND `version_number` của graph đó SHALL lớn hơn head version trước đó

#### Scenario: Single write vẫn tạo được graph version hợp lệ

- GIVEN một application write chỉ cập nhật một node của logical graph `kg-123`
- WHEN transaction source commit thành công
- THEN service SHALL seal một graph version mới cho `kg-123`
- AND node bị đổi SHALL được link vào change set của version đó

#### Scenario: Chỉ một phần graph thay đổi — version_number vẫn tăng cho whole graph, change set là partial

- GIVEN logical graph `kg-123` có 10.000 nodes và 50.000 relationships
- AND operation chỉ cập nhật 3 nodes
- WHEN transaction source commit thành công
- THEN service SHALL seal một graph version mới với `version_number` lớn hơn head trước đó
- AND change set của version đó SHALL chứa đúng 3 entity entries tương ứng với 3 nodes bị đổi
- AND change set SHALL NOT chứa các entities không bị đổi trong operation đó
- AND worker SHALL có thể load đúng 3 entities từ change set mà không phải scan toàn bộ graph

#### Scenario: Write không có reference_id — service tự generate và trả về

- GIVEN một application write cập nhật một node của logical graph `kg-123`
- AND caller không cung cấp `reference_id`
- WHEN transaction source commit thành công
- THEN service SHALL tự generate một `reference_id` cho graph version đó
- AND `reference_id` được generate SHALL được persist cùng graph version
- AND response trả về cho caller SHALL chứa `reference_id` đã được generate
- AND `version_number` SHALL vẫn tăng đơn điệu bình thường so với head version trước đó

### Requirement: Version sealing SHALL là durable handoff cho async projection

Hệ thống SHALL phát sinh durable event hoặc outbox record ở mức graph version để worker có thể đồng bộ
`graphdb`, `vectordb`, và các projection backends khác theo change set của version đó.

#### Scenario: Worker nhận graph-version event sau source commit

- GIVEN một graph version mới của `kg-123` đã được seal trong cùng transaction với source mutations
- WHEN worker poll durable queue hoặc outbox
- THEN worker SHALL nhìn thấy event tham chiếu tới đúng `identifier_id`, `version_id`, và `version_number`
- AND worker SHALL có đủ thông tin để load change set của version đó từ `relationshipdb`

#### Scenario: Bulk sync lớn chỉ phát GRAPH_VERSION_SEALED sau khi change set đã materialize xong

- GIVEN một bulk sync tạo graph version mới cho `kg-123`
- AND hệ thống chọn flow `PENDING_ENTITIES` vì change set quá lớn để ghi trong transaction nguồn
- WHEN transaction nguồn commit nhưng `kg_graph_version_entities` chưa được ghi xong
- THEN version đó SHALL chưa được xem là `SEALED`
- AND outbox SHALL NOT phát `GRAPH_VERSION_SEALED` cho version đó
- WHEN finalizer durable ghi xong toàn bộ change set và chuyển version sang `SEALED`
- THEN hệ thống SHALL phát đúng một durable `GRAPH_VERSION_SEALED` event cho version đó

#### Scenario: Version pending finalization được recover nếu process chết giữa chừng

- GIVEN một graph version của `kg-123` đang ở trạng thái `PENDING_ENTITIES`
- AND process finalize change set bị dừng sau khi transaction nguồn đã commit
- WHEN hệ thống chạy durable finalizer hoặc reconciler cho các version pending quá hạn
- THEN hệ thống SHALL có thể tiếp tục materialize change set của version đó
- AND version đó SHALL either được chuyển sang `SEALED` và phát event, hoặc được đánh dấu failed để điều tra
- AND version đó SHALL NOT bị bỏ mồ côi như một handoff đã hoàn tất

#### Scenario: Backend heads advance theo graph version

- GIVEN graph backend của `kg-123` đã áp dụng tới `version_number = 17`
- AND worker đang xử lý graph version `18`
- WHEN graph projection hoàn tất
- THEN graph backend head của `kg-123` SHALL advance lên `18`
- AND vector backend head của `kg-123` MAY vẫn ở `17` nếu vector sync chưa hoàn tất

#### Scenario: GraphDB worker chỉ xử lý entities trong change set, không scan toàn bộ graph

- GIVEN graph version `19` của `kg-123` có change set gồm 2 nodes và 5 relationships
- WHEN graphDB worker xử lý version `19`
- THEN worker SHALL chỉ upsert/delete đúng 2 nodes và 5 relationships đó vào graphdb
- AND worker SHALL NOT re-sync các nodes và relationships không có trong change set của version `19`

#### Scenario: VectorDB worker chỉ xử lý nodes và embeddable_relationships, bỏ qua relationship thường

- GIVEN graph version `20` của `kg-123` có change set gồm 1 node UPSERT và 4 relationship UPSERT
- WHEN vectorDB worker xử lý version `20`
- THEN worker SHALL generate embedding và upsert đúng 1 node đó vào vectordb
- AND worker SHALL NOT tạo embedding cho 4 relationships có `entity_kind = relationship` trong change set
- AND vectorDB head của `kg-123` SHALL advance lên `20` sau khi node đó hoàn tất

#### Scenario: VectorDB worker advance head kể cả khi version chỉ có relationship-only mutations

- GIVEN graph version `22` của `kg-123` có change set gồm toàn bộ là relationship UPSERT, không có node nào
- WHEN vectorDB worker nhận `GRAPH_VERSION_SEALED` event cho version `22`
- THEN worker SHALL xác định filtered change set (node + embeddable_relationship) là rỗng
- AND worker SHALL advance vectorDB head của `kg-123` lên `22` như một no-op version
- AND worker SHALL NOT bỏ qua advance head chỉ vì không có gì để embed

#### Scenario: FTS PostgreSQL head advance cùng source write, không phải qua worker

- GIVEN graph version `21` của `kg-123` đang được seal với 5 node mutations
- WHEN source rows được ghi vào `kg_nodes` trong transaction
- THEN FTS index SHALL phản ánh nội dung mới ngay sau khi transaction commit
- AND FTS head của `kg-123` SHALL advance lên `21` trong cùng transaction với source write
- AND SHALL NOT cần một async worker riêng để cập nhật FTS PostgreSQL

#### Scenario: FTS head hợp lệ ở version PENDING_ENTITIES khi dùng bulk sync flow

- GIVEN bulk sync tạo graph version `25` cho `kg-123` với flow PENDING_ENTITIES
- AND source rows đã được commit vào `kg_nodes` ở transaction nguồn (bước 4)
- AND version `25` vẫn đang ở trạng thái `PENDING_ENTITIES` (chưa SEALED)
- WHEN monitoring đọc FTS head của `kg-123`
- THEN FTS head MAY ghi nhận `version_number = 25`
- AND trạng thái này SHALL được monitoring coi là hợp lệ, không phải lag
- AND FTS head ở version `25` SHALL NOT bị so sánh với graphdb/vectordb head bằng cùng lag threshold vì graphdb/vectordb chưa xử lý version `25`

#### Scenario: Graph head chỉ advance sau khi toàn bộ change set pass entity stale guard

- GIVEN worker đang xử lý graph version `18` của `kg-123` với change set gồm 3 entities
- AND entity stale guard pass cho 2/3 entities
- AND 1 entity còn lại bị reject bởi stale guard do event out-of-order
- WHEN worker xử lý version đó
- THEN graph backend head của `kg-123` SHALL NOT advance lên `18`
- AND worker SHALL retry hoặc nack event cho entity bị reject
- AND graph head SHALL chỉ advance lên `18` khi toàn bộ entities trong change set đã pass stale guard

#### Scenario: Duplicate replay không được chặn graph head advance nếu toàn bộ entity đã ở target version

- GIVEN graph version `18` của `kg-123` đã được áp dụng thành công cho 2 entities
- AND worker replay lại cùng `GRAPH_VERSION_SEALED` event do duplicate delivery
- WHEN stale guard xác định 2 entities đó đã ở backend version tương ứng hoặc cao hơn
- THEN mỗi entity đó SHALL được xem là `ALREADY_APPLIED`
- AND graph head của `kg-123` MAY vẫn advance hoặc giữ nguyên ở `18` mà không bị xem là lỗi blocking
- AND worker SHALL NOT retry vô hạn chỉ vì duplicate replay

### Requirement: GRAPH_VERSION_SEALED event SHALL chứa đủ thông tin để worker load change set

Durable event hoặc outbox record phát sinh khi seal một graph version SHALL chứa tối thiểu
`identifier_id`, `version_id`, `version_number`, `reference_id` (NOT NULL), và locator để worker
có thể load change set của version đó từ `relationshipdb` mà không cần scan toàn bộ source graph.

#### Scenario: Event payload đủ để worker load change set không cần full-graph scan

- GIVEN một graph version mới của `kg-123` đã được seal với change set gồm 150 entity mutations
- WHEN worker nhận `GRAPH_VERSION_SEALED` event từ outbox
- THEN event payload SHALL chứa `identifier_id`, `version_id`, và `version_number`
- AND event payload SHALL chứa danh sách entity ids hoặc pointer tới change set đủ để worker fetch đúng mutations
- AND worker SHALL NOT phải scan toàn bộ `kg_nodes` / `kg_relationships` để xác định entities thuộc version đó

#### Scenario: reference_id trong event phản ánh đúng giá trị caller cung cấp hoặc giá trị service generate

- GIVEN pipeline cung cấp `reference_id = git:7f3ab21` khi sync graph `kg-123`
- WHEN worker nhận `GRAPH_VERSION_SEALED` event cho version đó
- THEN event payload SHALL chứa `reference_id = git:7f3ab21`

- GIVEN một application write không cung cấp `reference_id`
- WHEN worker nhận `GRAPH_VERSION_SEALED` event cho version được seal từ write đó
- THEN event payload SHALL chứa `reference_id` do service generate
- AND `reference_id` đó SHALL khớp với giá trị đã trả về cho caller trong response của write đó

#### Scenario: GRAPH_VERSION_SEALED luôn ngụ ý change set đã sẵn sàng để claim

- GIVEN worker nhận một `GRAPH_VERSION_SEALED` event cho graph `kg-123`
- WHEN worker bắt đầu xử lý event đó
- THEN worker SHALL có thể load đầy đủ change set của version được tham chiếu
- AND worker SHALL NOT cần chờ thêm một bước materialization ngoài băng nào khác để xác định entities thuộc version đó

### Requirement: Graph versions SHALL hỗ trợ storage classes ONLINE và OFFLINE

Hệ thống SHALL cho phép version history của một logical graph được phân tầng lưu trữ giữa `ONLINE` và
`OFFLINE` mà không làm mất metadata lineage hoặc khả năng tra cứu phiên bản.

#### Scenario: Truy xuất online version ngay lập tức

- GIVEN graph version `42` của `kg-123` đang ở storage class `ONLINE`
- WHEN operator hoặc service yêu cầu đọc metadata và change manifest của version đó
- THEN hệ thống SHALL trả được version này ngay trên hot path

#### Scenario: Offline version vẫn truy hồi được qua metadata

- GIVEN graph version `7` của `kg-123` đã được archive sang storage class `OFFLINE`
- WHEN operator yêu cầu restore hoặc inspect version đó
- THEN hệ thống SHALL tìm được metadata lineage và storage locator của version `7`
- AND thời gian truy hồi MAY chậm hơn online path
- AND version `7` SHALL không bị xem là mất dữ liệu chỉ vì không còn hot snapshot

#### Scenario: manifest_locator đủ để locate offline payload deterministically

- GIVEN graph version `7` của `kg-123` đã được archive sang storage class `OFFLINE`
- WHEN hệ thống persist `manifest_locator`
- THEN `manifest_locator` SHALL follow một URI deterministic dạng `s3://<bucket>/kg/<identifier_id>/versions/<version_number>/<version_id>.json.zst`
- AND restore workflow SHALL locate đúng payload chỉ từ `manifest_locator` đó

### Requirement: RelationshipDB SHALL là canonical content store của versioned knowledge graphs

`relationshipdb` SHALL giữ canonical node và relationship content. Projection stores như `graphdb`,
`vectordb`, và FTS backend riêng (nếu có) SHALL ưu tiên lưu topology, index materialization, filter attrs,
sync metadata, và source pointers thay vì trở thành canonical payload store.

#### Scenario: Projection stores chỉ replicate filter attrs đã được chốt

- GIVEN một node có các property như `ten`, `mo_ta`, `so_hop_dong`, và `status_value`
- WHEN service project node đó sang `graphdb` hoặc `vectordb`
- THEN backend MAY replicate các field hợp lệ như `node_type`, `domain_id`, `owner_tenant_id`, `owner_app_id`, `acl_visible_to`, `visibility`, `status_value`, và `is_deleted`
- AND backend SHALL NOT coi các property content-rich như `ten` hoặc `mo_ta` là filter attrs contractually allowed để dùng như source of truth cho canonical content
- AND nếu caller cần full payload thì service SHALL hydrate từ `relationshipdb`

#### Scenario: Semantic search hit được hydrate từ source content

- GIVEN vector backend trả về candidate `node-abc` của graph `kg-123`
- WHEN service materialize search response cho caller
- THEN service SHALL có thể hydrate canonical content của `node-abc` từ `relationshipdb`
- AND vector backend SHALL NOT là nguồn canonical duy nhất của node payload đó

#### Scenario: PostgreSQL FTS path phù hợp với canonical content rule

- GIVEN full-text search đang chạy qua PostgreSQL trên `kg_nodes`
- WHEN service trả kết quả FTS
- THEN canonical searchable content SHALL được đọc từ source-backed records
- AND implementation này SHALL được xem là tương thích với rule `relationshipdb` là canonical content store

#### Scenario: PostgreSQL FTS không còn phụ thuộc vào async worker indexing sau cutover

- GIVEN deployment dùng `FTS_ADAPTER=postgres`
- WHEN source write commit một graph version mới
- THEN PostgreSQL FTS SHALL phản ánh dữ liệu mới từ source-backed records mà không cần worker gọi `Index/Delete`
- AND `backend_kind=fts` head của graph đó SHALL được advance theo synchronous write path
- AND rollout SHALL có cơ chế phân biệt mode này với external FTS async mode trong monitoring

#### Scenario: Graph traversal trả topology từ graphdb nhưng full payload từ source

- GIVEN graph query tìm được một tập node qua `graphdb`
- WHEN caller cần full node payload của các node đó
- THEN service MAY dùng `graphdb` cho traversal và candidate discovery
- AND service SHALL có thể hydrate full canonical properties từ `relationshipdb` trước khi trả response
