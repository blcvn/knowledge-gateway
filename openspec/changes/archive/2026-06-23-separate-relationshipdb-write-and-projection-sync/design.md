# Design: Separate RelationshipDB Writes From Projection Sync

## Overview

Change này chốt một boundary kiến trúc đơn giản nhưng quan trọng:

- `relationshipdb` là write plane đồng bộ mà application request được phép tác động trực tiếp;
- `graphdb` và `vectordb` là projection plane bất đồng bộ, chỉ được cập nhật bởi job/worker sau khi source
  commit thành công.

Mục tiêu là làm rõ ownership, SLA, và failure semantics thay vì mở rộng tính năng mới.

Change này cũng chốt thêm boundary cho read/search routing và graph identity để write/read/search cùng
nói chung một ngôn ngữ vận hành.

## Architecture boundary

### 1. Synchronous write plane

Request path của application chỉ làm bốn việc:

1. xác thực quyền truy cập và validate payload;
2. ghi node/relationship mutation vào `relationshipdb`;
3. ghi outbox event hoặc durable sync trigger trong cùng transaction;
4. trả response dựa trên kết quả source commit.

Request path không được:

- gọi trực tiếp `graphdb`;
- gọi trực tiếp `vectordb`;
- chờ embedding generation;
- chờ worker reconcile hoặc projection completion.

Nói cách khác, request SLA chỉ bao phủ `relationshipdb` commit và durable handoff sang async pipeline.

### 2. Asynchronous projection plane

Job/worker là thành phần duy nhất sở hữu việc:

- claim source changes đã commit;
- sync node/relationship sang `graphdb`;
- sync searchable/vector representation sang `vectordb`;
- retry các lỗi tạm thời;
- theo dõi lag, backlog, drift, và repair khi cần.

Projection plane có SLA vận hành riêng, tách khỏi response time của API write.

### 3. Read plane over graph projections

Graph read theo `app_id` + `kg_id` có hai mode vận hành:

- `non-realtime`: luôn đọc từ `graphdb` qua graph query/Cypher path;
- `realtime`: cố gắng đọc từ `graphdb`, nhưng phải so trạng thái projection của entity trong `graphdb`
  với source version ở `relationshipdb` trước.

Thuật toán cho `realtime` mode:

1. xác định source record trong `relationshipdb` theo `kg_id` và scope truy cập của caller;
2. đọc `graphdb` projection version tương ứng của entity;
3. lấy source version từ `relationshipdb` hoặc projection ledger authoritative;
4. nếu `graph_version == source_version` thì trả kết quả từ `graphdb`;
5. nếu `graph_version < source_version` hoặc projection chưa tồn tại thì fallback đọc từ `relationshipdb`.

Điều này giữ được hai tính chất:

- `non-realtime` tối ưu latency và query power của `graphdb`;
- `realtime` ưu tiên độ tươi dữ liệu khi projection còn lag.

### 4. Search plane routing

Search được tách theo backend projection:

- semantic search: truy vấn `vectordb`
- RAG retrieval: dùng cùng vector-backed retrieval plane, có thể thêm expansion/rerank phía sau
- full-text search: truy vấn `fts db`
- hybrid search: hợp nhất kết quả từ semantic/vector và full-text/FTS theo policy xếp hạng của service

Request path của search không được đọc semantic candidates trực tiếp từ `relationshipdb`, trừ khi có một
repair/admin workflow ngoài scope change này.

### 5. Graph identity on writes

Source write không chỉ cần `domain_id` và entity identity, mà còn cần một `graph identity` hoặc
`graph scope discriminator` để tránh trộn dữ liệu thuộc các logical graph khác nhau.

Discriminator này có thể được:

- caller truyền tường minh; hoặc
- service dẫn xuất tất định từ metadata nguồn như `project_id`, `repo_id`, tenant/app scope, hoặc tổ hợp
  tương đương.

Yêu cầu semantic quan trọng là:

- cùng một logical graph thì discriminator phải ổn định giữa các lần sync;
- khác logical graph thì discriminator phải khác nhau;
- relationship write phải resolve endpoint trong đúng graph scope, trừ khi ontology/domain cho phép
  cross-graph relation một cách tường minh.

Với `codegraph`, graph identity phải phân biệt tối thiểu theo `project` hoặc `repo` boundary tương đương,
để symbol giống tên ở hai project/repo khác nhau không bị nhập nhầm vào cùng một knowledge graph.

## Data flow

```text
Application request
  -> write validation
  -> relationshipdb transaction
       - persist source rows
       - append outbox event
  -> return success/accepted to caller

Background job / worker
  -> claim committed outbox events
  -> load source state from relationshipdb
  -> project to graphdb
  -> project to vectordb
  -> update worker status / projection ledger / retry state

Read request (non-realtime)
  -> graph query / Cypher against graphdb
  -> return graph projection result

Read request (realtime)
  -> lookup source version in relationshipdb
  -> lookup graph projection version in graphdb / projection ledger
  -> if versions match: return graphdb result
  -> else: fallback to relationshipdb result

Semantic search / RAG
  -> vectordb query

Full-text search
  -> fts db query

Hybrid search
  -> vectordb query + fts db query
  -> merge/rerank in service
```

## Failure semantics

### Source write failures

Nếu validation hoặc transaction fail thì:

- source rows không được commit;
- outbox event không được để lại;
- request trả lỗi write-path chuẩn.

### Projection failures

Nếu sync sang `graphdb` hoặc `vectordb` fail sau khi source commit thành công thì:

- source state trong `relationshipdb` vẫn là canonical state;
- request write trước đó vẫn được xem là thành công;
- worker ghi nhận retryable failure, backlog, hoặc drift để vận hành xử lý tiếp.

### Read freshness mismatch

Nếu `realtime` read phát hiện `graphdb` đang chậm hơn `relationshipdb` cho cùng entity thì:

- request read không bị coi là lỗi projection;
- service trả dữ liệu từ `relationshipdb`;
- lag đó vẫn phải được phản ánh ở projection observability/integrity plane.

## Operational implications

### Ownership

- application team sở hữu contract của request payload, validation, và source commit
- worker/job runtime sở hữu throughput, retry, concurrency, và replay của projection pipeline

### Observability

Để boundary này hữu ích trong vận hành, hệ thống cần phân biệt ít nhất:

- write commit success/failure trên `relationshipdb`
- outbox backlog hoặc sync queue backlog
- projection lag cho `graphdb`
- projection lag cho `vectordb`
- dead-letter hoặc retry exhaustion của worker
- realtime read fallback rate từ `graphdb` sang `relationshipdb`
- search backend path đang dùng cho từng loại truy vấn khi debug hoặc audit
- distribution của graph identity / graph scope conflicts nếu write path phát hiện trùng sai scope

### Client-facing behavior

Client không được suy luận rằng dữ liệu đã có mặt ngay trong `graphdb` hoặc `vectordb` chỉ vì request write
trả thành công. Contract của response chỉ đảm bảo source write đã durable và async sync đã được lên lịch qua
outbox/durable trigger.

Tương tự, client không được suy luận rằng mọi graph read đều đọc từ cùng một backend: `non-realtime` có thể
đọc thẳng từ `graphdb`, còn `realtime` có thể fallback về `relationshipdb` khi projection chưa bắt kịp.

## Spec impact

Change này sửa hai vùng spec:

- `write-path`: làm rõ request path chỉ ghi `relationshipdb` + outbox và không trực tiếp sync projections
- `sync-consistency`: làm rõ worker/job là thành phần độc quyền thực hiện sync sang `graphdb`/`vectordb`
- `read-templates`: làm rõ routing và fallback rule cho graph reads theo mode
- `semantic-search` và `full-text-search`: làm rõ backend nào phục vụ từng search mode

Không cần tạo spec mới vì yêu cầu là siết boundary của hai capability đã tồn tại.
