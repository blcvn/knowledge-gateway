---
id: TASK-SRC-001
title: Implement Zep Search Domain Layer
service: zep-search
status: Done
---

# Objective
Implement the Domain layer representing Search Intents and Results.

# Requirements
1. **Query Entity**: Define structured representation of search intent including `scope` (`edges`, `nodes`, `episodes`), `project_uuid`, `user_uuid`, `limit`.
2. **Filters**: Define constraints for `node_labels` and `edge_types`.
3. **Thresholds**: Include parameters for `min_fact_rating` and `mmr_lambda`.
4. **ScoredResult**: Agnostic result container with a normalized `float32` score.
