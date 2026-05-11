---
id: TASK-COG-002
title: Implement Usecase Ports and DTOs
feature: FEAT-COG-001
status: Done
---
# Task: Implement Usecase Ports and DTOs

## Objective
Define the Data Transfer Objects (DTOs) and port interfaces for the Usecase layer.

## Files to Create/Modify
- `internal/usecase/dto/request.go`
- `internal/usecase/dto/response.go`
- `internal/usecase/port/input.go`
- `internal/usecase/port/output.go`

## Requirements
- `request.go`: Define `TriggerCognifyReq` and `CognifyConfig`.
- `response.go`: Define `CognifyJobResult` and `PipelineMetrics`.
- `input.go`: Define the inbound ports: `CognifyUseCase` and `JobManager` interfaces.
- `output.go`: Define the outbound ports: `GraphRepository`, `VectorRepository`, `LLMClient`, `EmbedderClient`, `JobRepository`, and `EventPublisher` interfaces.
- **Constraint**: Port interfaces must depend exclusively on domain entities and DTOs, completely decoupled from infrastructure or third-party implementations.
