---
id: TASK-PIP-003
title: "Implement Infrastructure & Bootstrap"
service: cognee-pipeline
status: Done
priority: P1
linked_feat: FEAT-PIP-001
---

## Objective
Provide the infrastructure plumbing, dual gRPC service registration, and dependency injection to bootstrap the single `cognee-pipeline` binary.

## Scope
1. **Configuration & Server Setup**:
   - Define a combined configuration schema in `internal/infra/config/config.go`.
   - Setup `internal/infra/server/grpc.go` to register both `CogneeIngestionService` and `CogneeCognifyService` on the exact same port (e.g., 9011).
2. **Dependency Injection**:
   - Create a combined Google Wire configuration in `internal/infra/wire/wire.go` that provides all usecases, shared repositories, and handlers.
3. **Entry Point & Deployment**:
   - Implement `cmd/server/main.go` as the bootstrap execution point.
   - Combine health checks into a single endpoint representing both subsystems.
   - Create a unified `Dockerfile` and `Makefile` tailored for the pipeline service (target image size ≤50MB).
4. **Telemetry**:
   - Integrate OTel tracing and Prometheus metrics spanning both ingestion and cognify logic.

## Acceptance Criteria
- [x] `main.go` successfully bootstraps both gRPC services on the same port.
- [x] Single binary correctly resolves all DI dependencies via Wire.
- [x] Unified health endpoint accurately reports the status of shared DBs and message queues.
