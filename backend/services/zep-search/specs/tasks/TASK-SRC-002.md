---
id: TASK-SRC-002
title: Implement Zep Search Retrievers
service: zep-search
status: Done
---

# Objective
Implement retrieving logic for both PostgreSQL and Neo4j.

# Requirements
1. **VectorRetriever**: Implement `pgvector` similarity search logic for querying session history.
2. **GraphRetriever**: Implement Cypher queries against Neo4j to find 1-hop and 2-hop linked entities and temporal facts.
3. Define strict interfaces for these retrievers to allow mocking.
4. Implement unit tests using mocked DB connections.
