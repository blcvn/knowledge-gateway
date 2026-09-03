---
id: TASK-KNW-005
title: Implement Bifrost Embedder Adapter
feature: FEAT-KNW-005
status: Done
---

## Objective
Thực thi implement Bifrost embedder adapter dựa trên FEAT-KNW-005.

## Tasks
1. Tạo file `internal/adapter/embedder/bifrost_embedder.go`:
   - Implement `EmbedderClient` port.
   - Gửi POST request tới Bifrost `/v1/embeddings`.
   - Implement hàm `Embed()` (trả về float32 vector).
   - Implement hàm `EmbedBatch()`.

2. Logic validation và rules:
   - Batch size limiting: chia batch lớn thành các chunks kích thước `EMBEDDER_BATCH_SIZE`.
   - Dimension validation: trả về `ErrInvalidEmbeddingDimension` nếu sai dimension so với config.
   - Circuit breaker bảo vệ embedder.

3. Unit Tests:
   - Dùng mock HTTP server.
   - Đảm bảo coverage >= 80%.
