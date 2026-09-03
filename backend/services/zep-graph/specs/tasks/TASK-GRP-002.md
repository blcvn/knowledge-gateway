---
id: TASK-GRP-002
title: Implement Zep Graph Extractor Usecase
service: zep-graph
status: Done
---

# Objective
Implement the LLM-driven Graphiti extraction logic.

# Requirements
1. **Graphiti Extractor**: Receives message chunks, formats prompts, and calls the LLM interface to identify nodes and edges.
2. **Group ID Strategy**: Group messages by session ID. Prefix episode UUIDs with `{groupID}-{messageUUID}` if `addGroupIDPrefix=true`.
3. Provide high coverage unit tests with mocked LLM responses validating the correct parsing of facts.
