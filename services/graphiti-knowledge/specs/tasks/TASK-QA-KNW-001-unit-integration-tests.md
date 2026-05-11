---
id: TASK-QA-KNW-001
title: Unit and Integration Tests
solution: SOL-001
status: Done
---

## Objective
Viết và chạy các unit tests & integration tests cho toàn bộ graphiti-knowledge service theo SOL-001.

## Tasks
1. Đảm bảo overall unit test coverage toàn bộ service >= 80%.
2. Verify ExtractEntities trả về đúng entities từ LLM.
3. Verify ResolveEntities có deduplicate với >0.85 cosine similarity threshold.
4. Verify ExtractEdges trả về bi-temporal fact triples.
5. Verify ResolveEdges phát hiện được contradictions và invalidate các edges cũ.
6. Verify GenerateEmbedding trả về vectors đúng số dimension.
7. Verify UpdateCommunity chạy được label propagation và LLM summarization.
8. Kiểm thử lại bulkhead, giới hạn số concurrent LLM requests không vượt quá MAX_CONCURRENT.
9. Đảm bảo proto interface swap được trực tiếp vào cho graphiti-pipeline.
