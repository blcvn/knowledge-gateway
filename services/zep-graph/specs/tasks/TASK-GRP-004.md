---
id: TASK-GRP-004
title: Implement Zep Graph Adapter Layer
service: zep-graph
status: Done
---

# Objective
Implement gRPC Handlers, NATS Subscriber, and LLM Client integrations.

# Requirements
1. **gRPC Server**: Expose manual fact override endpoints (`AddFact`, `DeleteFact`, `GetEpisodes`, `GetOntology`, `SetOntology`).
2. **NATS Subscriber**: Bind to `zep.memory.messages.ingested` to trigger asynchronous background extraction (target 10-20s duration).
3. **LLM Client**: Implement standard interface adapter for calling OpenAI/Anthropic models for extraction.
