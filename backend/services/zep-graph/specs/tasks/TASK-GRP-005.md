---
id: TASK-GRP-005
title: Implement Zep Graph Infrastructure & Bootstrap
service: zep-graph
status: Done
---

# Objective
Implement the Neo4j 5.x persistence logic and bootstrap the service.

# Requirements
1. **Neo4j Repository**: Implement graph storage with vector and temporal indexes.
2. Ensure queries optimize for filtering on `valid_at` / `invalid_at`.
3. **Redis**: Implement rate-limiting and exponential backoff for the LLM extraction pipeline.
4. **Bootstrap**: Setup Wire DI, configuration, OTel observability tracing for long-running async tasks.
5. Create E2E tests validating the NATS -> LLM -> Neo4j pipeline.
