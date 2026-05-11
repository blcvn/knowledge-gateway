---
id: TASK-RES-001
service: ov-resource
status: Done
---

# TASK-RES-001: Implement Domain Layer

## Objective
Define the core business entities, value objects, domain events, and interface definitions for the `ov-resource` service following the Clean Architecture pattern as specified in `architecture.md` and `tdd.md`.

## Requirements
1. **Models (`internal/domain/model/`)**:
   - Define `Resource` entity, `ResourceType`, and `IngestionResult` (`resource.go`).
   - Define `Chunk` entity and `ChunkMetadata` covering position, tokens, and AST node info (`chunk.go`).
   - Define `WatchTask` entity, `WatchEvent`, and `WatchStatus` (`watch.go`).
   - Define `ParserConfig` and `ParserType` enum (`parser.go`).
2. **Repositories Interfaces (`internal/domain/repository/`)**:
   - Define `ResourceRepository` (`resource_repo.go`).
   - Define `WatchRepository` (`watch_repo.go`).
3. **Events & Errors**:
   - Define `ResourceIngested` domain event (`event.go`).
   - Define standard domain errors (`errors.go`): `UnsupportedFormat`, `ParseFailed`, `IngestFailed`.
4. **Usecase Ports**:
   - Input ports (`internal/usecase/port/input.go`): `IngestUseCase`, `ParseUseCase`, `WatchUseCase`, `RefreshUseCase`.
   - Output ports (`internal/usecase/port/output.go`): `FileWriterPort`, `ParserPort`, `EventPublisherPort`.
5. **DTOs (`internal/usecase/dto/`)**:
   - Map strictly to the gRPC request/response models: `IngestRequest`, `IngestResponse` (chunks_count, total_tokens, path, parse_duration_ms), `ParseRequest` (chunk_size, chunk_overlap), `ParseResponse`, `WatchRequest`, `WatchEvent`, `RefreshRequest`, `RefreshResponse`.

## Dependencies
- Zero external dependencies.
