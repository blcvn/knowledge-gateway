---
id: TASK-ENG-001
title: Implement Domain Layer Models and Interfaces
service: memobase-engine
layer: Domain (Layer 1)
status: Done
---

# Task: Implement Domain Layer Models and Interfaces

## Objective
Implement the enterprise-grade Domain layer (Layer 1) for `memobase-engine` according to the Clean Architecture standard defined in `architecture.md` and `tdd.md`. This layer must have ZERO external dependencies.

## Requirements
1. **Entities and Value Objects (`internal/domain/model/`)**:
   - `profile.go`: Define `Profile` entity (id, user_id, project_id, topic, sub_topic, content, attributes, updated_at).
   - `event.go`: Define `UserEvent` (id, user_id, project_id, event_data, embedding, created_at) and `EventGist` (id, event_id, gist_data, embedding) entities.
   - `blob.go`: Define `Blob` value objects.
   - `pipeline.go`: Define `PipelineResult` (profiles_delta, events_created, tokens_consumed) and `MergeDecision` (add[], update[], delete[] - YOLO merge output).
   - `config.go`: Define `ProfileConfig` (language, profile_strict_mode, additional_profiles, event_tags) and `EventTagDef`.

2. **Repository Interfaces (`internal/domain/repository/`)**:
   - `profile_repo.go`: Define `ProfileRepository` interface.
   - `event_repo.go`: Define `EventRepository` interface.
   - `blob_repo.go`: Define `BlobRepository` interface for reading blobs.

## Constraints
- ZERO external imports in the Domain layer.
- Exact mapping to the structures in `data-model.md` and `tdd.md`.
