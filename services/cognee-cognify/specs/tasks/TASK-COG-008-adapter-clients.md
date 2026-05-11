---
id: TASK-COG-008
title: Implement External Client Adapters
feature: FEAT-COG-002
status: Done
---
# Task: Implement External Client Adapters

## Objective
Implement adapters for interacting with external services, specifically LLM/Embedding services via Bifrost, and gRPC calls to the `cognee-ingestion` service.

## Files to Create/Modify
- `internal/adapter/client/llm_client.go`
- `internal/adapter/client/embedder_client.go`
- `internal/adapter/client/ingestion_client.go`

## Requirements
- `LLMClient`: Integrate with the Bifrost Gateway to generate structured JSON outputs (`CompleteStructured`) and raw completions. Add context timeouts and standard retries.
- `EmbedderClient`: Integrate with Bifrost to generate vector embeddings (e.g., using `text-embedding-3-large`).
- `IngestionClient`: Setup a gRPC client to communicate with `cognee-ingestion`, retrieving raw dataset items based on `dataset_id`.
- Inject proper OpenTelemetry tracing contexts into all outbound requests.
