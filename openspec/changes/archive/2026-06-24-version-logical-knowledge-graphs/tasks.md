# Tasks

- [x] **T1** — Chuẩn hóa canonical identity cho logical knowledge graph.
  Yêu cầu:
  - xác định contract giữa `project_id` do caller cung cấp và `identifier_id` do service quản lý;
  - quy định rule resolve/reuse identifier cho repeated writes hoặc repeated syncs;
  - làm rõ quan hệ giữa `graph_scope` hiện tại và graph identifier registry mới.
  - **chốt rule cho request không có stable external key**: khi nào service cấp `identifier_id` mới, khi nào caller phải gửi lại canonical identifier đã được cấp trước đó, và khi nào phải từ chối thay vì tự merge theo fallback `domain + tenant + app`.

- [x] **T2** — Thiết kế schema cho graph registry, version chain, và entity change set.
  Yêu cầu:
  - đề xuất tables cho graph identifiers, graph versions, entity-to-version linkage, và projection heads;
  - xác định khóa unique, sequence/version counter, và index phục vụ lookup head/replay/archive;
  - mô tả metadata tối thiểu cần giữ cho `reference_id`, storage class, và lineage;
  - **chốt cơ chế sinh `version_number`**: counter row trên `kg_graph_identifiers` với `SELECT ... FOR UPDATE`, hay Postgres sequence per graph; mô tả retry path cho conflict;
  - **chốt hành vi concurrent writers trên cùng graph**: caller B bị block hay fail ngay khi caller A đang giữ lock; timeout bao lâu; nếu dùng sequence thì xử lý gap version_number từ rollback transaction như thế nào.

- [x] **T3** — Thiết kế write/sync handoff theo graph version.
  Yêu cầu:
  - source write vào `relationshipdb` vẫn phải atomically seal một graph version;
  - outbox/event payload phải mang đủ `identifier_id`, `version_id`, `version_number`, `reference_id` (NOT NULL — service generate nếu caller không cung cấp);
  - single write và bulk sync đều phải map được về một graph version operation boundary;
  - **xác định atomicity scope cho bulk sync**: với change set lớn, ghi toàn bộ `kg_graph_version_entities` trong cùng transaction có thể không scalable; chốt cách tiếp cận (ví dụ: `version_status = PENDING_ENTITIES → SEALED`) và mô tả điều kiện worker được phép xử lý event.
  - nếu dùng `PENDING_ENTITIES`, **chốt durable finalization protocol**: ai ghi change set, khi nào mới được phát `GRAPH_VERSION_SEALED`, và cơ chế recover/reconcile cho các version pending bị kẹt do crash giữa chừng.

- [x] **T4** — Thiết kế projection runtime dựa trên graph-version-aware events.
  Yêu cầu:
  - worker phải có khả năng claim và sync theo graph version lineage;
  - mỗi backend (`graph`, `vector`, `fts`) cần head version riêng ở mức graph;
  - stale guard per-entity hiện có phải tiếp tục được giữ như lớp bảo vệ bắt buộc;
  - **chốt ordering constraint**: graph head chỉ được advance lên `version_number = N` sau khi toàn bộ entities trong change set của version N đã pass entity stale guard trên backend đó; mô tả cách worker detect và xử lý khi một entity bị reject;
  - **mô tả per-backend sync protocol**:
    - graphdb worker: xử lý nodes + relationships từ change set, upsert topology + filter attrs;
    - vectordb worker: xử lý `node` và `embeddable_relationship` từ change set; advance head kể cả khi filtered change set rỗng (no-op advance); process từng version theo thứ tự tăng dần, không skip;
    - FTS PostgreSQL: advance head trong transaction source write, không qua async worker; head hợp lệ ở version PENDING_ENTITIES — monitoring không được so sánh FTS head với async backend head bằng cùng lag threshold;
    - FTS external (nếu có): async tương tự vectordb, chỉ sync searchable nodes.
  - **chốt `entity_kind` enum**: liệt kê rõ `node` / `relationship` / `embeddable_relationship` và tiêu chí write path dùng để phân loại entity vào đúng kind khi ghi `kg_graph_version_entities`;
  - **chốt retry-safe stale guard semantics**: phân biệt `APPLIED` / `ALREADY_APPLIED` / blocking failure để duplicate replay không làm graph head bị kẹt vô hạn.

- [x] **T5** — Chốt policy online/offline cho version storage.
  Yêu cầu:
  - định nghĩa version nào bắt buộc `ONLINE`, version nào có thể `OFFLINE`;
  - metadata online phải đủ để tra cứu và restore version offline;
  - rollout phải mô tả archive, restore, và retention policy ở mức contract;
  - **chốt format của `manifest_locator`** trước khi implement restore workflow: ví dụ S3 URI, blob reference key, archival table partition key; format này phải deterministic và đủ để locate offline payload mà không cần scan toàn bộ lịch sử.

- [x] **T6** — Chốt boundary "canonical content ở relationshipdb, projection stores giữ pointer/index/topology".
  Yêu cầu:
  - graphdb chỉ giữ dữ liệu cần cho traversal, filtering, và version tracking;
  - vectordb chỉ giữ embedding cùng source pointer và filter attrs;
  - FTS PostgreSQL hiện tại phải được ghi nhận là baseline source-backed path, và FTS backend riêng trong tương lai không được trở thành canonical content store;
  - **liệt kê danh sách cụ thể "filter attrs" được phép nhân bản vào graphdb và vectordb** dựa trên ba tiêu chí đã chốt trong design (ontology traversal policy, access control index, ANN filter metadata); property nào không thoả phải ở lại relationshipdb.

- [x] **T7** — Lập kế hoạch migration và rollout an toàn.
  Yêu cầu:
  - backfill graph identifiers từ `graph_scope` hiện có;
  - xác định cách sinh initial graph head/version cho dữ liệu đang chạy;
  - mô tả cách vận hành song song old entity-only projection ledger với graph-version ledger trong giai đoạn chuyển tiếp.
  - mô tả **cutover riêng cho `FTS_ADAPTER=postgres`**: vô hiệu hoá worker-owned FTS indexing, backfill head synchronous, và tách monitoring/lag semantics giữa Postgres FTS với external FTS async mode.
