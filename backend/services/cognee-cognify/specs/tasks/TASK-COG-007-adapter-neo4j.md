---
id: TASK-COG-007
title: Implement Neo4j Adapter
feature: FEAT-COG-002
status: Done
---
# Task: Implement Neo4j Repository

## Objective
Implement the graph database adapter for creating and querying the Knowledge Graph nodes and edges.

## Files to Create/Modify
- `internal/adapter/repository/neo4j/graph_repo.go`
- `internal/adapter/repository/neo4j/queries.go`

## Requirements
- `queries.go`: Store parameterized Cypher queries. Nodes must be strictly isolated by tenant, using labels such as `Tenant_{tenant_id}`.
- `GraphRepository`: Execute graph operations including node creation (Entities), edge creation (Relationships), and community detection leveraging Graph Data Science (GDS).
- Ensure all graph mutating operations execute safely within Neo4j transactions.
- Write integration tests verifying queries against a Neo4j testcontainer.
