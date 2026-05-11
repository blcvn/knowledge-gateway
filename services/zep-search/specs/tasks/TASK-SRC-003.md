---
id: TASK-SRC-003
title: Implement Zep Search Reranker Pipeline
service: zep-search
status: Done
---

# Objective
Implement the 5 multi-modal reranking algorithms for merging search results.

# Requirements
1. **RRF**: Implement Reciprocal Rank Fusion to merge Graph and Vector results.
2. **MMR**: Implement Maximal Marginal Relevance algorithm for diversity.
3. **Graph Specific Rerankers**: Implement `node_distance`, `episode_mentions`.
4. **Temporal Decay**: Implement algorithm to smoothly downweight older facts.
5. **Cross Encoder**: Provide interface for an external ML cross-encoder model reranking.
6. Provide extensive test suite validating reranking mathematical correctness.
