---
id: TASK-ING-005
title: Implement Background Workers and Resilience
service: memobase-ingestion
status: DONE
created: 2026-05-11
---

# Task: Implement Background Workers and Resilience

## Objective
Implement background processes to handle idle timeout flushes and dead-letter queues as outlined in the system capabilities and limitations.

## Requirements

1. **Idle Buffer Timeout Worker**:
   - Implement a cron/ticker-based background worker.
   - Periodically scan for `idle` buffer entries that have been waiting for `> 1h`.
   - Trigger the `FlushBufferUseCase` for these specific users/projects to ensure blobs do not get stuck indefinitely.

2. **Dead-Letter Queue (DLQ) Handling**:
   - Implement a mechanism or worker to handle permanently `failed` buffer entries.
   - Identify buffer entries that have failed processing multiple times.
   - Move or flag these entries appropriately so they do not block the system or can be manually inspected.

3. **Concurrency Mitigation**:
   - Ensure parallel flushes do not cause duplicate processing by strictly enforcing FSM status checks (`WHERE status='idle'`) during state transitions.

## Constraints
- The background workers should run asynchronously and not impact the latency of the main gRPC APIs.
- Must handle graceful shutdown properly when the service is terminated.
