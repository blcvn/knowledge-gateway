# Proposal: Separate RelationshipDB Writes From Projection Sync

## Problem

Write path của `kg-service` hiện đã dùng `relationshipdb`/PostgreSQL làm source of truth và dùng outbox để
chiếu dữ liệu sang `graphdb` và `vectordb`. Tuy nhiên contract vận hành giữa hai phần này chưa được chốt
rõ ràng ở mức OpenSpec:

- request từ application chưa được phát biểu đủ mạnh là chỉ được commit source data vào `relationshipdb`
  cùng outbox event;
- phần sync sang `graphdb` và `vectordb` chưa được mô tả như một projection plane độc lập, do job/worker
  sở hữu SLA và retry;
- khi backlog hoặc lỗi projection xảy ra, rất dễ kéo nhầm trách nhiệm này vào request path hoặc làm mờ
  semantics phản hồi của API.

Điều này làm kiến trúc khó vận hành khi scale ingest: ứng dụng cần một write SLA rõ ràng cho source commit,
trong khi sync downstream phải được tối ưu và quan sát như một pipeline bất đồng bộ riêng.

Ngoài ra, boundary của read/search và graph identity cũng chưa được phát biểu đủ chặt:

- graph read theo `app_id` + `kg_id` chưa mô tả rõ khi nào đọc từ `graphdb` và khi nào fallback về
  `relationshipdb`;
- search chưa chốt rõ tuyến thực thi giữa `vectordb` và `fts db`;
- write path chưa buộc caller hoặc service phải mang một `graph identity`/`graph scope` đủ để tách các
  knowledge graph độc lập như `codegraph` theo `project`, `repo`, hoặc discriminator tương đương.

## Proposed Solution

Tách change này thành hai trách nhiệm được phát biểu rõ trong spec:

1. **Synchronous write plane**
   - request của application chỉ validate, ghi source records vào `relationshipdb`, và append outbox
     trong cùng transaction;
   - request thành công không chờ `graphdb`, `vectordb`, embedding, hay reconciliation hoàn tất.

2. **Asynchronous projection plane**
   - job/worker riêng consume outbox hoặc queue bền vững sau commit;
   - worker chịu trách nhiệm sync từ `relationshipdb` sang `graphdb` và `vectordb`, retry, theo dõi lag,
     và repair drift;
   - projection failure không rollback source commit đã thành công.

3. **Read and search routing**
   - graph read theo `app_id` + `kg_id` mặc định đi qua `graphdb`, nhưng có mode `realtime` để so phiên
     bản projection với source trước khi quyết định dùng `graphdb` hay fallback `relationshipdb`;
   - search semantic/hybrid đi qua `vectordb`, full-text đi qua `fts db`, và mỗi path phải phát biểu rõ
     contract backend của nó.

4. **Graph identity on writes**
   - write path phải persist hoặc dẫn xuất được `graph identity`/`graph scope discriminator` để phân biệt
     các logical graph độc lập;
   - với `codegraph`, identity đó phải phân biệt tối thiểu theo `project` hoặc `repo` boundary tương đương.

## Scope

### In scope

- siết lại contract write-path chỉ ghi `relationshipdb` + outbox
- siết lại contract sync-path là trách nhiệm của job/worker bất đồng bộ
- định nghĩa rõ boundary giữa request SLA và projection SLA
- làm rõ observability/failure semantics giữa source commit và downstream projection
- đặc tả read-mode `realtime` và `non-realtime` cho graph reads
- đặc tả routing của semantic search, hybrid search, và full-text search
- rà soát yêu cầu graph identity / graph scope trên write path

### Out of scope

- thay đổi business schema của node/relationship
- redesign adapter cụ thể cho Neo4j, Memgraph, Milvus, PgVector
- thay đổi API read/search ngoài các semantics liên quan đến projection lag
- thay thế outbox bằng broker khác trong change này
- áp đặt duy nhất một field vật lý tên `graph_id` nếu implementation chọn cơ chế tương đương khác

## Success Criteria

- spec `write-path` nói rõ application request chỉ commit vào `relationshipdb` và không trực tiếp sync
  `graphdb`/`vectordb`
- spec `sync-consistency` nói rõ projection sang `graphdb`/`vectordb` là trách nhiệm của job/worker
  bất đồng bộ
- failure ở projection plane không làm thay đổi kết quả commit của source write đã thành công
- worker lag/backlog được coi là trạng thái vận hành của projection plane, không phải lỗi của request path
- spec `read-templates` nói rõ rule `realtime` so version `graphdb` với `relationshipdb` trước khi đọc
- spec search nói rõ vector search dùng `vectordb`, full-text dùng `fts db`, hybrid hợp nhất hai path đó
- spec `write-path` buộc source write mang hoặc dẫn xuất được `graph identity` để tránh trộn các knowledge
  graph độc lập như `codegraph` khác `project`/`repo`
