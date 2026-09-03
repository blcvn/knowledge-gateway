---
id: TASK-GRP-003
title: Implement Zep Graph Temporal Resolver Usecase
service: zep-graph
status: Done
---

# Objective
Implement the temporal contradiction resolution logic.

# Requirements
1. **Temporal Resolver**: When a newly extracted fact contradicts an existing fact, instead of a hard delete, update the existing fact's `invalid_at` timestamp.
2. New facts should have `valid_at` set appropriately based on the episode's timestamp.
3. Include unit tests validating temporal progression and invalidation rules.
