# Task: Usecase Layer - Health Aggregation Fan-Out (TASK-ADM-005)

**Status:** DONE

## Description
Implement the gRPC health check fan-out mechanism to monitor active OV services.

## Requirements
- Define `HealthCheckerPort` in `internal/usecase/port/output.go`.
- Implement `HealthClient` adapter in `internal/adapter/client/health_client.go` to communicate with `ov-storage`, `ov-search`, and `ov-session`.
- Implement `HealthUseCase` in `internal/usecase/health_ops.go`.
- Use `errgroup` to make parallel gRPC Health Check calls with a configurable 5-second timeout.
- Aggregate responses and mark status as degraded if any critical service is `SERVING_STATUS_UNKNOWN` or `SERVING_STATUS_NOT_SERVING`.
