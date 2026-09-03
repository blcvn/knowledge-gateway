---
id: TASK-SRC-004
title: Implement Zep Search Adapter Layer
service: zep-search
status: Done
---

# Objective
Implement the gRPC Server handlers.

# Requirements
1. **gRPC Server**: Implement `GraphSearch` and `SessionSearch` endpoints defined in the `.proto`.
2. Map incoming gRPC request parameters to Domain `Query` objects, including all reranking parameters and filters.
3. Map internal `ScoredResult` outputs to the gRPC `SearchResponse`.
