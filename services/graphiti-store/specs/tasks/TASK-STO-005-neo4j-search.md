---
id: TASK-STO-005
title: Neo4j Search Primitives — Cosine, Fulltext, BFS
feature: FEAT-STO-005
status: Done
---

## Objective
Thực thi implement các search primitives trong Neo4j (cosine similarity, fulltext search, BFS traversal) dựa trên FEAT-STO-005.

## Tasks
1. Tạo file `internal/adapter/driver/neo4j/search_repo.go`.

2. Cài đặt các methods:
   - `CosineSimilaritySearch`: thực thi vector index query (gọi `db.index.vector.queryNodes`) trên trường `name_embedding` hoặc `fact_embedding`. Sắp xếp top-K entities theo score.
   - `FulltextSearch`: thực thi BM25 index query (gọi `db.index.fulltext.queryNodes`) trên `name`, `summary`, `fact`. Sắp xếp theo score.
   - `BFSSearch`: thực thi query variable-length path traversal cho graph để duyệt subgraph tới depth cụ thể. Sắp xếp kết quả theo distance.

3. Điều kiện bắt buộc cho Repository:
   - Toàn bộ kết quả tìm kiếm phải bao gồm score/distance để phục vụ ranking ở higher layers.
   - Bắt buộc phải thực hiện group_id scoping filter đối với tất cả searches.

4. Integration Tests:
   - Viết test scenarios kiểm tra searches bằng Neo4j testcontainer chứa pre-loaded data và pre-created indexes.
   - Đảm bảo coverage >= 80%.
