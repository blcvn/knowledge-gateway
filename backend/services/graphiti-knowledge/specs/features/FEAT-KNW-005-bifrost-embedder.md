---
id: FEAT-KNW-005
title: Bifrost Embedder Adapter
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement embedding generation adapter via Bifrost gateway implementing EmbedderClient port.

## Scope

- `internal/adapter/embedder/bifrost_embedder.go` — Embed single + batch texts
- POST to Bifrost /v1/embeddings endpoint
- Batch size limiting (max EMBEDDER_BATCH_SIZE per request)
- Dimension validation against config

## Acceptance Criteria

- [ ] AC-1: Embed() returns float32 vector of correct dimension
- [ ] AC-2: EmbedBatch() splits large batches into EMBEDDER_BATCH_SIZE chunks
- [ ] AC-3: Circuit breaker protects against embedder failures
- [ ] AC-4: Dimension mismatch returns ErrInvalidEmbeddingDimension

## Test Requirements
- **Unit tests**: Mock HTTP server
- **Minimum coverage**: 80%
