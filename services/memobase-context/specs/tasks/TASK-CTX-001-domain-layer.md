---
id: TASK-CTX-001
title: "Domain Layer Implementation"
status: Done
created: 2026-05-11
---

# Task: Implement Domain Layer

## 1. Objective
Implement the Domain Layer (Layer 1) for the `memobase-context` microservice following the 4-layer Clean Architecture. This layer defines the core entities, value objects, and repository interfaces without any external dependencies.

## 2. Requirements

### 2.1. Entities & Value Objects
- **Profile (`internal/domain/model/profile.go`)**: Read-only view structure containing `id`, `user_id`, `project_id`, `content`, `topic`, `sub_topic`, and `updated_at`.
- **EventGist (`internal/domain/model/event_gist.go`)**: Value object for vector search containing `gist_data`, `embedding`, and `created_at`.
- **ContextResult & PromptTemplate (`internal/domain/model/context.go`)**: Core domain models for context assembly and prompt formatting.
- **TruncationPolicy**: Configuration for truncation rules (`prefer_topics`, `only_topics`, `max_token_size`, `max_subtopic_size`).

### 2.2. Repository Interfaces
- **ProfileReadRepository (`internal/domain/repository/profile_repo.go`)**: Define read-only methods (`GetProfiles`, `SearchProfiles`).
- **EventGistSearchRepository (`internal/domain/repository/event_repo.go`)**: Define semantic search method (`SearchBySimilarity`).

## 3. Acceptance Criteria
- [x] All structs and interfaces defined in `internal/domain`.
- [x] No external packages imported in the domain layer.
- [x] Basic unit tests implemented for any domain logic (e.g., PromptTemplate formatting).
