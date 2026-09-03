---
id: TASK-ING-07
title: Implement gRPC Handler and Mapper
service: cognee-ingestion
feature: FEAT-ING-002
status: Done
---

## Objective
Implement the gRPC server handler and Proto-to-Domain mappers.

## Files to Create/Update
- `internal/adapter/grpc/handler.go`: Implement `CogneeIngestionServiceServer`.
- `internal/adapter/grpc/mapper.go`: Map Proto messages to Domain entities/DTOs.
- Related `*_test.go` files.

## Acceptance Criteria
- Handler successfully integrates with Usecase ports.
- Mapper conversions result in zero data loss.
- Streaming AddData endpoint operates efficiently.
- Integration/unit tests using mock usecases pass with >= 80% coverage.
