---
id: TASK-ING-005
title: Implement NATS Event Publisher
service: graphiti-ingestion
type: task
status: done
priority: P1
created: 2026-05-11
dependencies: [TASK-ING-002]
estimated_time: 3h
linked_feat: FEAT-ING-005
---

## Objective
Implement NATS JetStream publisher cho `graphiti.episode.ingested` events.

## Scope
- `internal/adapter/event/nats_publisher.go` — EventPublisher port implementation
- Publish on saga completion: episode_id, group_id, nodes_count, edges_count
- Retry 3x with backoff on transient failures

## Acceptance Criteria
- [x] Event published on saga completion
- [x] Retry on transient NATS failures
- [x] Events include group_id for downstream filtering
- [x] NATS unavailability doesn't block saga completion (async)

## Test Requirements
- Unit tests: Mock JetStream context
- Minimum coverage: 80%
