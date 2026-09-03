---
id: FEAT-ING-005
title: NATS Event Publisher
service: graphiti-ingestion
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement NATS JetStream publisher cho `graphiti.episode.ingested` events.

## Scope

- `internal/adapter/event/nats_publisher.go` — EventPublisher port implementation
- Publish on saga completion: episode_id, group_id, nodes_count, edges_count
- Retry 3x with backoff on transient failures

## Acceptance Criteria

- [ ] AC-1: Event published on saga completion
- [ ] AC-2: Retry on transient NATS failures
- [ ] AC-3: Events include group_id for downstream filtering
- [ ] AC-4: NATS unavailability doesn't block saga completion (async)

## Test Requirements
- **Unit tests**: Mock JetStream context
- **Minimum coverage**: 80%
