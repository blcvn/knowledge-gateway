---
id: TASK-COG-009
title: Implement NATS Messaging and gRPC Server Handlers
feature: FEAT-COG-002
status: Done
---
# Task: Implement Entrypoints (NATS & gRPC)

## Objective
Implement event-driven entry/exit points (NATS publisher/subscriber) and incoming API handlers (gRPC).

## Files to Create/Modify
- `internal/adapter/nats/subscriber.go`
- `internal/adapter/nats/publisher.go`
- `internal/adapter/grpc/handler.go`
- `internal/adapter/grpc/mapper.go`

## Requirements
- `subscriber.go`: Subscribe to `cognee.data.ingested`. Upon receipt, fetch dataset items via `IngestionClient` and trigger `CognifyUseCase`.
- `publisher.go`: Publish `cognee.pipeline.completed` with `PipelineMetrics` when a job successfully finishes.
- `handler.go`: Implement the `CogneeCognifyServiceServer` protocol handlers: `TriggerCognify`, `GetJobStatus`, and `CancelJob`.
- `mapper.go`: Decouple Protobuf definitions from domain models by implementing dedicated mapping functions.
