---
id: TASK-SEA-006
title: Implement Reranker Strategies
feature: FEAT-SEA-006
status: Done
---

## Objective
Thực thi implement 5 chiến lược Rerank dựa trên FEAT-SEA-006.

## Tasks
1. Tạo file `internal/usecase/reranker/interface.go`
   - Định nghĩa `Reranker` interface.

2. Tạo file `internal/usecase/reranker/rrf.go`
   - Implement Reciprocal Rank Fusion (k=60 default).

3. Tạo file `internal/usecase/reranker/mmr.go`
   - Implement Maximal Marginal Relevance (lambda=0.7 default).

4. Tạo file `internal/usecase/reranker/cross_encoder.go`
   - Implement Cross-Encoder reranker, delegate sang graphiti-pipeline qua gRPC.

5. Tạo file `internal/usecase/reranker/node_distance.go`
   - Implement Node Distance strategy sử dụng BFS distance.

6. Tạo file `internal/usecase/reranker/episode_mentions.go`
   - Implement Episode Mentions strategy dưa trên số lần count.

7. Unit Tests
   - Viết test với inputs biết trước và so sánh ranking.
   - Cho phép combine rerankers theo chain.
   - Coverage >= 85%.
