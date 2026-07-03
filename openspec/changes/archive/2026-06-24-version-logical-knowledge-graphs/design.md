# Design: Version Logical Knowledge Graphs

## Overview

Change này đưa versioning từ mức entity lên mức **logical knowledge graph**.

Mục tiêu là để `kg-service` có thể quản lý:

- một graph cụ thể của một `tenant/app` là graph nào;
- graph đó đang ở version số mấy;
- version đó đến từ `reference_id` nào;
- worker đã sync version đó sang `graphdb` / `vectordb` đến đâu;
- version nào còn online, version nào đã archive offline;
- và đâu là canonical source content so với projection/index payload.

## Current State

Qua rà soát code hiện tại:

### 1. Graph scope đã tồn tại nhưng chưa là first-class identity

Write path đã có `deriveGraphScope(...)` và ưu tiên:

1. `graph_id` / `_kg_graph_scope`
2. `repo_id` / `repository_id`
3. `project_id`
4. fallback theo `domain + tenant + app`

Điều này tốt cho tránh trộn graph khác nhau, nhưng chưa tạo ra một registry chính thức để trả lời:

- graph nào là graph canonical của `project-x`?
- graph đó có identifier nội bộ nào?
- head version của graph đó là gì?

### 2. Versioning hiện tại là per-entity, không phải per-graph

Code hiện có hai lớp version chính:

- `domain_version` trên node/relationship source rows
- `kg_projection_versions` với `SourceVersion`, `GraphVersion`, `VectorVersion`

Hai lớp này giải bài toán monotonicity và replica lag ở mức entity rất tốt, nhưng chưa diễn đạt được:

- version 42 của cả graph gồm những entity nào;
- version đó gắn với `reference_id` nào;
- rollback hoặc replay theo whole-graph revision.

### 3. Outbox hiện handoff theo entity mutation

Pipeline hiện tại bám theo `kg_outbox_events` và mutation của từng node/relationship.
Đây là durable handoff đúng cho projection correctness, nhưng chưa có orchestration ở mức graph revision.

### 4. Projection stores còn mang nhiều canonical payload

- graph projection đang ghi `Properties` khá đầy đủ lên graph adapter;
- vector documents đang mang `DomainProps` để search path có thể trả content trực tiếp;
- FTS PostgreSQL hiện lại không cần projection riêng mà query thẳng `kg_nodes`.

Điều này cho thấy hệ thống đang ở trạng thái pha trộn:

- một số path đã sống theo source-of-truth model;
- một số path vẫn nhân bản nội dung đầy đủ sang projection stores.

## Goals

### Identity goals

- mỗi logical KG có một identifier ổn định trong scope `tenant/app`
- caller có thể dùng `project_id` như external key, nhưng service vẫn có `identifier_id` canonical

### Versioning goals

- mỗi sync/write tương ứng với một graph version tăng đơn điệu
- version SHALL có `reference_id` không null — caller-provided hoặc service-generated
- version có thể được archive nhưng vẫn giữ lineage đầy đủ

### Projection goals

- worker đồng bộ theo graph version lineage, không chỉ theo raw entity events
- mỗi backend replica có head version riêng theo graph
- stale writes tiếp tục bị chặn ở mức entity như hiện tại

### Storage goals

- `relationshipdb` là nơi giữ full canonical content
- `graphdb` / `vectordb` / FTS backend riêng (nếu có) chỉ giữ phần cần cho topology, index, filtering, và lookup

## Proposed Model

## 1. Graph identifier registry

Thêm một first-class registry, ví dụ:

- `kg_graph_identifiers`

Các field chính:

- `identifier_id` (UUID canonical)
- `owner_tenant_id`
- `owner_app_id`
- `external_project_id` hoặc external graph key caller cung cấp
- `graph_scope`
- `status`
- `head_version_id`
- `created_at`, `updated_at`

### Canonical identity decision

Change này chốt quy tắc:

- **external key** như `project_id` là optional domain-facing identifier;
- **identifier_id** là canonical key nội bộ;
- repeated writes/syncs của cùng logical graph SHALL resolve về cùng `identifier_id`.

#### Trường hợp không có external key ổn định

Không phải caller nào cũng có `project_id`, nhưng service vẫn không được tự ý nhập nhầm hai logical graph khác nhau chỉ vì
cùng `tenant/app/domain`.

Quy tắc chốt thêm:

- nếu caller muốn **reuse một logical graph đã tồn tại**, request phải mang một stable graph key đủ mạnh, ví dụ:
  `graph_id`, `_kg_graph_scope`, `project_id`, hoặc một external graph key khác đã được registry chấp nhận;
- nếu caller **không có stable external key**, service MAY tạo một `identifier_id` mới và trả lại cho caller để các lần
  ghi sau dùng lại chính identifier đó;
- fallback `domain + tenant + app` từ `deriveGraphScope(...)` chỉ nên được dùng cho **legacy backfill / single-graph
  compatibility mode**, không được coi là bằng chứng đủ mạnh để tự động merge nhiều logical graphs mới.

Hệ quả là contract của T1 phải tách rõ:

- **reuse path**: cần stable key hoặc canonical identifier đã biết trước;
- **create-new path**: service cấp identifier mới;
- **legacy fallback path**: chỉ dùng trong migration/backfill với ràng buộc vận hành rõ ràng.

Điều này phù hợp với hiện trạng đã có `graph_scope`, nhưng thay vì chỉ derive để validate relationship endpoints,
service sẽ persist kết quả đó thành graph registry thực sự.

## 2. Graph version chain

Thêm một version ledger, ví dụ:

- `kg_graph_versions`

Các field chính:

- `version_id` (UUID)
- `identifier_id`
- `version_number` (monotonic trong phạm vi một graph)
- `reference_id` (NOT NULL — commit SHA, import batch id, snapshot id, hoặc service-generated ID nếu caller không cung cấp)
- `parent_version_id`
- `storage_class` (`ONLINE` | `OFFLINE`)
- `sealed_at`
- `created_by`
- `change_summary`
- `manifest_locator`

### Version granularity

Để tránh nổ version quá mức, change này chọn semantic:

- **mỗi write/sync operation seal đúng một graph version**
- single-node write là trường hợp suy biến của một operation
- bulk ingest/sync có thể gom nhiều entity mutation vào cùng một graph version

Nói cách khác, graph version đi theo **change set boundary**, không bắt buộc one-version-per-entity.

### Whole-graph version number vs partial change set

`version_number` là định danh snapshot của **toàn bộ logical graph** tại một thời điểm, tăng đơn điệu mỗi khi có operation. Khi chỉ một phần graph thay đổi:

- `version_number` của graph **vẫn tăng** — nó đại diện cho state mới nhất của whole graph
- `kg_graph_version_entities` **chỉ chứa entities thực sự bị đổi** trong operation đó (partial delta)
- worker **không bao giờ phải scan toàn bộ graph** để xử lý một version — chỉ load change set từ `kg_graph_version_entities`

Quan hệ này tương đương với git: commit SHA đại diện cho toàn bộ repo tại một điểm, nhưng diff chỉ chứa files thay đổi. "Version N của graph X" có nghĩa là "toàn bộ graph X, với delta được áp dụng từ version N-1."

### Version number generation

`version_number` là monotonic integer trong phạm vi một `identifier_id`. Cơ chế đề xuất:

- **Counter row với `SELECT ... FOR UPDATE`**: giữ một counter column trên row `kg_graph_identifiers`, tăng trong cùng transaction write. Đây là cách đơn giản nhất và không cần DDL sequence riêng per-graph.
- **Sequence per graph**: mỗi graph có một Postgres sequence riêng. Tránh row lock nhưng cần quản lý lifecycle sequence khi graph bị archive/delete.

T2 phải chốt cơ chế. Nếu dùng counter row, implementation cần retry path cho conflict khi nhiều writer đồng thời cùng graph. Nếu dùng sequence, T2 phải mô tả cách tạo và dọn sequence.

**Concurrent writers trên cùng graph**: hai callers cùng ghi vào graph X đồng thời sẽ race trên version counter. Hành vi phải được chốt rõ trong T2:
- counter row với `SELECT ... FOR UPDATE`: caller B bị block cho đến khi caller A release lock; B chờ theo timeout cấu hình, không fail ngay; nếu hết timeout thì trả lỗi cho B
- sequence per graph: không block nhưng cần xử lý gap trong `version_number` nếu một transaction rollback sau khi đã lấy sequence value

## 3. Entity-to-version linkage

Thêm linkage table, ví dụ:

- `kg_graph_version_entities`

Các field chính:

- `version_id`
- `entity_kind` (`node` | `relationship` | `embeddable_relationship`)
- `entity_id`
- `source_version`
- `change_kind` (`UPSERT` | `DELETE`)

`entity_kind` enum values:

- `node` — node thông thường, được graphdb và vectordb xử lý
- `relationship` — relationship thông thường, chỉ được graphdb xử lý; vectordb bỏ qua
- `embeddable_relationship` — relationship mang content có thể embed (ví dụ: relationship chứa description hoặc metadata tìm kiếm được); được cả graphdb và vectordb xử lý

Phân biệt `relationship` vs `embeddable_relationship` là trách nhiệm của write path khi ghi linkage row.

Table này cho phép:

- biết version X chạm những entity nào;
- worker load đúng change set để project;
- audit/replay một graph version mà không phải scan toàn bộ source graph.

### Atomicity scope cho bulk sync

Với single write, toàn bộ 6 bước trong section 4 có thể nằm trong một transaction mà không tốn kém.

Với bulk sync lớn (hàng nghìn entity), ghi toàn bộ `kg_graph_version_entities` trong cùng transaction sẽ tạo lock window lớn và tăng nguy cơ deadlock.

Cách giải quyết đề xuất:

- thêm `version_status` column trên `kg_graph_versions`: `PENDING_ENTITIES` → `SEALED` → `FAILED_FINALIZATION` (nếu cần điều tra)
- transaction chính ghi source rows và tạo version với `status = PENDING_ENTITIES`
- **không phát `GRAPH_VERSION_SEALED` trong transaction đầu** nếu change set chưa materialize xong
- sau khi commit, một finalizer bền vững ghi `kg_graph_version_entities` theo batch
- chỉ khi toàn bộ linkage đã được persist thành công, finalizer mới:
  - chuyển version sang `SEALED`
  - ghi durable outbox event `GRAPH_VERSION_SEALED`
- worker chỉ xử lý `GRAPH_VERSION_SEALED` cho version đã ở trạng thái `SEALED`

Điểm cần giữ chặt:

- durable handoff cho async projection chỉ hoàn tất ở **bước seal cuối cùng**, không phải lúc source transaction đầu tiên commit;
- nếu process chết giữa chừng, hệ thống phải có durable finalization work item hoặc scanner/reconciler để tìm lại các version
  `PENDING_ENTITIES` quá hạn và tiếp tục finalize thay vì để version mồ côi;
- `PENDING_ENTITIES` không được xuất hiện như một version đã sẵn sàng cho worker.

T3 phải chốt cách tiếp cận này hoặc lý do chọn cách khác.

## 4. Version-aware outbox and projection heads

Write path hiện vẫn cần outbox durability, nhưng event contract nên nâng lên mức graph version:

1. ghi source rows vào `relationshipdb`
2. resolve/create `identifier_id`
3. cấp `version_number` mới cho graph đó
4. nếu change set đủ nhỏ: ghi linkage entity -> version trong cùng transaction
5. append outbox event `GRAPH_VERSION_SEALED`
6. commit transaction

Hoặc với bulk sync lớn:

1. ghi source rows vào `relationshipdb`
2. resolve/create `identifier_id`
3. cấp `version_number` mới cho graph đó với `status = PENDING_ENTITIES`
4. commit transaction nguồn
5. finalizer ghi linkage entity -> version theo batch
6. finalizer chuyển version sang `SEALED` và append durable outbox event `GRAPH_VERSION_SEALED`

Outbox payload của event này nên chứa tối thiểu:

- `identifier_id`
- `version_id`
- `version_number`
- `reference_id` (luôn có giá trị — caller-provided hoặc service-generated)
- danh sách entity ids hoặc locator tới change set

`GRAPH_VERSION_SEALED` luôn có nghĩa là:

- change set đã materialize xong;
- worker có thể load change set mà không cần scan toàn bộ graph;
- version đã sẵn sàng để projection backends claim.

### Replica heads

Bên cạnh `kg_projection_versions` ở mức entity, nên có thêm head table ở mức graph/backend, ví dụ:

- `kg_graph_projection_heads`

field chính:

- `identifier_id`
- `backend_kind` (`graph`, `vector`, `fts`)
- `applied_version_number`
- `applied_version_id`
- `applied_at`
- `status`

Lý do:

- entity ledger vẫn cần cho stale guard và drift debug;
- graph head mới là thứ trả lời "backend này đã sync tới version mấy của graph này rồi?"

### Per-backend sync protocol

Mỗi backend có cơ chế sync khác nhau. Đây là phần design cần phân biệt rõ.

#### GraphDB (async projection)

Worker nhận `GRAPH_VERSION_SEALED` event và:

1. load change set từ `kg_graph_version_entities` cho `version_id` đó
2. với mỗi entity trong change set:
   - **Node UPSERT**: upsert node id, labels, graph identifier, sync version, filter attrs vào graphdb
   - **Relationship UPSERT**: upsert edge (source id, target id, type, filter attrs, sync version) vào graphdb
   - **Node/Relationship DELETE**: remove khỏi graphdb
3. mỗi entity phải pass entity stale guard trước khi ghi
4. sau khi toàn bộ change set pass stale guard → advance `kg_graph_projection_heads` cho `backend_kind=graph` lên version này

GraphDB xử lý **cả node và relationship** vì topology là thứ graphdb cần để traversal.

#### VectorDB (async projection, node và embeddable_relationship)

Worker nhận `GRAPH_VERSION_SEALED` event và:

1. load change set, **lọc chỉ lấy entities có `entity_kind` là `node` hoặc `embeddable_relationship`**
2. với mỗi entity trong filtered change set:
   - **UPSERT**: load canonical content từ `relationshipdb`, generate embedding, upsert (embedding, entity_id, graph_identifier, sync_version, filter_attrs) vào vectordb
   - **DELETE**: remove khỏi vectordb
3. mỗi entity phải pass entity stale guard trước khi ghi
4. sau khi toàn bộ filtered entities hoàn tất → advance head cho `backend_kind=vector`

**Khi filtered change set rỗng** (version chỉ có `relationship` mutations thuần túy): worker không có gì để ghi nhưng **vẫn phải advance head** lên version đó. Đây là no-op advance hợp lệ — bỏ qua advance sẽ khiến head lag sau mỗi relationship-only version dù không có vấn đề thực sự.

**Thứ tự xử lý**: worker advance từng version theo thứ tự tăng dần, kể cả no-op versions. Không skip version. Lý do: skip yêu cầu query "version này có entities cho vectordb không?" trước khi advance — thêm complexity mà lợi ích nhỏ, vì no-op version advance là O(1).

VectorDB worker có thể chạy độc lập với graphDB worker — head của chúng advance riêng biệt.

#### FTS PostgreSQL (synchronous với source write)

FTS PostgreSQL có cơ chế **khác cơ bản** so với graphdb và vectordb: nó **không phải async projection**.

Vì FTS query đọc trực tiếp từ `kg_nodes` (source table), index FTS được cập nhật tự động khi source rows được ghi (qua tsvector column, index expression, hoặc trigger). Không có worker riêng cho FTS PostgreSQL.

Hệ quả:
- FTS head advance **cùng transaction với source write**, không phải sau outbox event
- `kg_graph_projection_heads` cho `backend_kind=fts` được advance ngay trong write path, không qua worker
- FTS PostgreSQL luôn reflect đúng current head của graph tại thời điểm read

**Lưu ý với PENDING_ENTITIES flow**: khi bulk sync dùng flow hai bước, source rows được commit ở transaction đầu (bước 4) trước khi version được SEALED (bước 6). FTS head advance tại bước 4, nên sẽ ghi `version_number = N` trong khi version N vẫn đang ở `PENDING_ENTITIES`.

Đây là hành vi đúng về correctness (FTS content đã cập nhật), nhưng monitoring cần biết:
- FTS head có thể ở version N khi version N chưa SEALED
- So sánh FTS head với graphdb/vectordb head trong trạng thái này không phải lag — đây là đặc tính của synchronous FTS path
- Monitoring cho FTS PostgreSQL **không được** dùng cùng lag threshold với async backends

#### FTS External (async projection, nếu có)

Nếu service dùng FTS backend riêng (Elasticsearch, OpenSearch, ...), cơ chế tương tự vectordb:

1. load change set, lọc entities có searchable content
2. sync inverted index cho nodes UPSERT/DELETE
3. advance head sau khi hoàn tất

FTS external **không được** trở thành canonical content store — chỉ giữ searchable materialization và source pointer.

## 5. Online and offline version storage

### ONLINE

`ONLINE` là version có thể lấy ngay với latency vận hành bình thường.

Nên giữ online:

- current head
- một cửa sổ recent history cấu hình được
- manifests và change sets cần cho replay ngắn hạn

### OFFLINE

`OFFLINE` là version không còn giữ warm snapshot, nhưng vẫn còn:

- metadata trong Postgres
- manifest locator
- lineage pointer tới parent/head

Offline payload có thể nằm ở:

- object storage snapshot
- compressed manifest blob
- archival table/partition

Change này chốt contract:

- offline version vẫn phải tra được
- restore phải deterministic
- restore có thể chậm hơn online path

`manifest_locator` SHALL dùng URI có dạng deterministic:

- `s3://<bucket>/kg/<identifier_id>/versions/<version_number>/<version_id>.json.zst`

Các provider khác MAY dùng cùng cấu trúc path logic nhưng đổi scheme theo backend của họ.
Điểm quan trọng là chỉ cần `manifest_locator` là đủ để locate payload mà không scan toàn bộ history.

## 6. Canonical content boundary

Đây là quyết định kiến trúc quan trọng nhất của change.

### RelationshipDB

`relationshipdb` là canonical content store.
Nó giữ:

- full node properties
- full relationship properties
- ownership/access metadata
- graph identifier resolution metadata
- version lineage metadata

### GraphDB

`graphdb` nên giữ:

- topology
- node/relationship ids
- graph identifier
- sync version metadata
- filter attrs (xem định nghĩa bên dưới)
- optional lightweight summary fields thực sự cần cho query planning

`graphdb` không nên là nơi giữ full arbitrary node payload nếu payload đó đã có trong `relationshipdb`.

**Định nghĩa filter attrs:** một property được phép tồn tại trong `graphdb` hoặc `vectordb` là "filter attr" nếu thoả một trong ba điều kiện:

1. property đó được khai báo trong ontology traversal policy là traversal filter (ví dụ: `node_type`, `status`, `access_scope`);
2. property đó được index ở mức graph query cho access control hoặc multi-tenancy (ví dụ: `tenant_id`, `app_id`);
3. property đó là ANN filter metadata cần cho pre-filter hoặc post-filter trong vector search.

Change này chốt danh sách cụ thể được phép replicate:

- `node_type`
- `domain_id`
- `owner_tenant_id`
- `owner_app_id`
- `acl_visible_to`
- `visibility`
- `status_value`
- `is_deleted`
- `rel_type` cho relationship topology trong `graphdb`

Các field khác, kể cả content-rich properties, không được xem là filter attrs. Nếu backend cần trả payload đầy đủ cho caller thì service SHALL hydrate từ `relationshipdb` thay vì dựa vào projection store.

### VectorDB

`vectordb` nên giữ:

- embedding
- node id
- graph identifier
- sync version
- filter attrs cần cho ANN filtering
- optional snippet/hash/preview nếu cần debug

Search hit sau khi chọn candidate nên có thể rehydrate canonical content từ `relationshipdb`.
Nếu backend giữ thêm metadata nội bộ cho scoring hoặc debug, metadata đó SHALL không được coi là canonical content hoặc filter attr contractually visible.

### FTS

Với PostgreSQL FTS hiện tại, hệ thống đã gần như đúng đích vì search query đọc thẳng từ `kg_nodes`.

Nếu sau này dùng FTS backend riêng:

- backend đó nên chỉ giữ inverted index / searchable materialization / source pointer;
- response content canonical vẫn nên được rehydrate từ `relationshipdb` khi cần.

## Read and Search Implications

## Graph reads

Graph traversal vẫn nên tận dụng `graphdb` cho relationship traversal và candidate discovery.
Tuy nhiên node payload trả về cho caller nên có thể đi qua bước hydrate từ `relationshipdb` nếu query cần full content.

## Semantic search

Semantic/vector search nên tách làm hai bước:

1. ANN backend trả candidate ids + score + filter attrs tối thiểu
2. service hydrate canonical node content từ `relationshipdb`

Điều này tăng một round-trip source lookup nhưng giảm duplicated payload và làm version rollback/archive nhất quán hơn.

## Full-text search

FTS PostgreSQL hiện đã đọc source trực tiếp nên phần lớn đã phù hợp.
Nếu service dùng hybrid search, candidate merge nên diễn ra trên `node_id`, sau đó hydrate canonical payload một lần.

## Integrity Rules

### 1. Version order per graph

Trong cùng một `identifier_id`, `version_number` phải tăng đơn điệu.

### 2. Projection order per backend

Mỗi backend chỉ được advance graph head theo chiều tăng version.

### 3. Entity stale guard stays mandatory

Dù worker sync theo graph version, stale guard per-entity hiện có vẫn phải giữ để tránh:

- event out-of-order
- partial replay
- duplicate delivery

### 4. Version sealing is the async trigger

Source write chỉ được xem là hoàn tất handoff khi graph version đã được seal và outbox event version-aware đã được persist.

### 5. Graph head advances only after full change set passes entity stale guard

Hai ledger cùng tồn tại:

- `kg_projection_versions` (per-entity): stale guard, drift debug, out-of-order protection
- `kg_graph_projection_heads` (per-graph per-backend): trả lời "backend này đã sync tới version mấy?"

Ordering rule: graph head của một backend **chỉ được advance** lên `version_number = N` khi toàn bộ entities trong change set của version N đạt một trong hai trạng thái:

- **APPLIED**: entity được backend áp dụng trong lần xử lý hiện tại;
- **ALREADY_APPLIED**: backend đã ở source/backend version tương ứng hoặc cao hơn do replay/duplicate delivery, nên entity này được xem là idempotent success.

Chỉ các trạng thái như:

- missing dependency,
- source version chưa tới,
- entity stale guard cho thấy backend đang thiếu một bước trung gian bắt buộc,
- lỗi ghi backend thực sự

mới được xem là blocking failure khiến worker phải retry/nack và chưa được advance graph head.

## Rollout Strategy

1. thêm registry và version tables mà chưa đổi read/write path public ngay
2. resolve `graph_scope` hiện có sang `identifier_id`
3. bắt đầu ghi graph versions song song với entity outbox hiện tại
4. thêm worker path đọc `GRAPH_VERSION_SEALED`
5. thêm graph projection heads
6. với `FTS_ADAPTER=postgres`, cắt worker-owned FTS indexing:
   - `ftsAdapter.Index/Delete` ở worker được no-op hoặc bypass rõ ràng;
   - `backend_kind=fts` head được backfill từ source truth và từ đây chỉ advance trong write transaction;
   - metric/monitoring phải phân biệt `fts-postgres` (sync) với external FTS (async)
7. chuyển search/read sang hydrate canonical content sau candidate selection
8. giới thiệu archive policy cho version cũ

## Risks

### Risk: write amplification ở graph version layer

Nếu mỗi mutation nhỏ đều thành một graph version, số version có thể tăng rất nhanh.

Giảm thiểu:

- version gắn với operation boundary
- bulk sync gom entity vào một change set
- cho phép ingest pipeline có `reference_id` và batching rõ

### Risk: đọc/search phải hydrate thêm từ source

Latency có thể tăng do phải đọc lại `relationshipdb`.

Giảm thiểu:

- hydrate theo batch ids
- cache recent node payloads
- chỉ hydrate full content khi response thực sự cần

### Risk: archive làm replay phức tạp hơn

Giảm thiểu:

- metadata version luôn online
- offline payload có locator chuẩn hóa
- restore workflow được test từ đầu

## Spec Impact

Change này thêm capability mới `knowledge-graph-versioning` để chốt:

- graph identifier
- graph version chain
- version-aware sync handoff
- online/offline archival
- canonical content boundary giữa source và projection stores

Các capability hiện có như `write-path`, `sync-consistency`, `semantic-search`, và `full-text-search` có thể cần
điều chỉnh implementation về sau, nhưng change này trước hết chốt contract tổng thể cho versioned logical KG.
