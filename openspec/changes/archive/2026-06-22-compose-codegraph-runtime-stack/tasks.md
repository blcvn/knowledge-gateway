# Tasks

- [x] **T1** — Chuẩn hóa Compose deployment path để boot stack CodeGraph validation với Postgres tương thích `pgvector`, Memgraph, Qdrant, migration/init containers, và `kg-service` chạy profile `qdrant-memgraph`.
- [x] **T2** — Cập nhật inventory/docs cấu hình để mô tả đầy đủ các biến embedding HTTP (`EMBEDDING_PROVIDER=http`, `EMBEDDING_URL`, `EMBEDDING_MODEL`, `EMBEDDING_API_KEY`) theo contract tham chiếu từ `tests/llm/embedding-vnp.txt`, dùng placeholder an toàn cho secrets.
- [x] **T3** — Thêm một script repo-owned chạy đủ flow CodeGraph local runtime: khởi động Compose, bootstrap tenant/app/domain/ontology, verify bootstrap, upsert CodeGraph KG data vào `kg-service`, và verify get/list, search, template/index behavior.
- [x] **T4** — Thiết kế rerun semantics cho script để có thể bỏ qua hoặc reuse an toàn các bước init/bootstrap đã chạy từ lần đầu, thay vì yêu cầu reset stack hoặc xóa dữ liệu.
