# Tasks

- [x] **T1** — Cập nhật spec `write-path` để chốt rằng application request chỉ được persist source records
  vào `relationshipdb` và append outbox/durable sync trigger trong cùng transaction.

- [x] **T2** — Bổ sung scenario cho `write-path` khẳng định request thành công không được phụ thuộc vào
  `graphdb`, `vectordb`, embedding, hay worker reconcile completion.

- [x] **T3** — Cập nhật spec `sync-consistency` để chốt rằng sync từ `relationshipdb` sang `graphdb` và
  `vectordb` là trách nhiệm của background job/worker, không phải request handler.

- [x] **T4** — Bổ sung scenario cho `sync-consistency` mô tả projection failure hoặc backlog chỉ ảnh hưởng
  projection plane và không rollback source commit đã hoàn tất.

- [x] **T5** — Rà soát các tài liệu implementation hoặc runtime follow-up để bảo đảm tên gọi và observability
  tách bạch giữa `relationshipdb` write SLA và projection worker SLA.

- [x] **T6** — Bổ sung spec `read-templates` cho graph read theo `app_id` + `kg_id`, gồm hai mode
  `realtime` và `non-realtime`, cùng rule so sánh `graphdb` version với source version trong
  `relationshipdb` trước khi quyết định fallback.

- [x] **T7** — Bổ sung spec search để chốt semantic/RAG đi qua `vectordb`, full-text đi qua `fts db`,
  và hybrid hợp nhất hai pipeline đó mà không làm mờ backend contract.

- [x] **T8** — Mở rộng spec `write-path` để buộc source writes mang hoặc dẫn xuất được `graph identity`
  / `graph scope discriminator` đủ để phân biệt các logical graph độc lập.

- [x] **T9** — Bổ sung scenario cụ thể cho `codegraph` hoặc các knowledge graph tương tự, trong đó graph
  identity phải phân biệt tối thiểu theo `project` hoặc `repo` boundary tương đương.
