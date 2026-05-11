---
id: TASK-STO-008
title: Implement Deployment & CI/CD Infrastructure
service: ov-storage
status: Done
---

# TASK-STO-008: Implement Deployment & CI/CD Infrastructure

## Objective
Prepare the `ov-storage` service for Kubernetes/production deployment in alignment with `runbook.md` and enterprise operations standards.

## Requirements
1. **Containerization**:
   - Write a multi-stage `Dockerfile` optimized for minimal binary size and security (distroless/scratch).
   - Provide a `docker-compose.yml` block for local developer testing.
2. **Build Tooling**:
   - Implement a robust `Makefile` with commands for `build`, `test`, `lint`, `mock`, and `generate` (for Protobufs/Wire).
3. **Health & Readiness**:
   - Ensure gRPC `HealthServer` accurately reflects internal dependencies (DB ping, Redis ping, NATS connection status).

## Acceptance Criteria
- [x] `Dockerfile` builds successfully and produces a minimal, secure image.
- [x] The `Makefile` exposes standardized developer commands.
- [x] `docker-compose up` cleanly starts the service.
- [x] The health check endpoint adheres strictly to the gRPC Health Checking Protocol.
