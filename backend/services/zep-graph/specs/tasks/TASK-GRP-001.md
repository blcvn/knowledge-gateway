---
id: TASK-GRP-001
title: Implement Zep Graph Domain Layer
service: zep-graph
status: Done
---

# Objective
Implement the Domain layer for Knowledge Graph entities and Ontology rules.

# Requirements
1. **Node Entity**: Represent Graphiti entities with proper classification.
2. **Node Priority Hierarchy**: Implement logic to enforce the 9-level priority (User > Assistant > Preference ... > Object).
3. **Edge Entity**: Include `valid_at` and `invalid_at` fields, standard types (`LOCATED_AT`, `OCCURRED_AT`).
4. **Episode Entity**: Represent temporal chunks of messages.
5. Create Interfaces: `GraphRepository`, `OntologyManager`, and `LLMExtractor`.
