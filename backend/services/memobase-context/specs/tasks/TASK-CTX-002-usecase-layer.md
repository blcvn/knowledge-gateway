---
id: TASK-CTX-002
title: "Usecase Layer Implementation"
status: Done
created: 2026-05-11
---

# Task: Implement Usecase Layer

## 1. Objective
Implement the Usecase Layer (Layer 2) for `memobase-context`. This layer handles the core context assembly algorithm, profile truncation, and orchestrates data retrieval from caches and databases.

## 2. Requirements

### 2.1. Ports & DTOs
- Define Input and Output ports in `internal/usecase/port`.
- Define Request and Response DTOs in `internal/usecase/dto` aligning with gRPC structures.

### 2.2. Use Cases Implementation
- **GetProfilesUseCase (`internal/usecase/get_profiles.go`)**:
  - Implement cache-first retrieval logic (Redis fallback to DB).
  - Implement **Truncation Algorithm**:
    - Sort profiles by `updated_at` DESC.
    - Reorder to prioritize `prefer_topics`.
    - Filter by `only_topics`.
    - Apply `max_subtopic_size` limit and accumulate tokens up to `max_token_size`.
- **SearchProfilesUseCase (`internal/usecase/search_profiles.go`)**:
  - Handle semantic profile search with optional chat-aware filtering.
- **GetContextUseCase (`internal/usecase/get_context.go`)**:
  - Load and apply fallbacks for configuration: `DEFAULT_MAX_TOKEN_SIZE` (default 500) and `PROFILE_EVENT_RATIO` (default 0.7).
  - Calculate token budget (e.g., 0.7 ratio for profile).
  - Use `errgroup` for parallel retrieval of truncated profiles and event gists.
  - Assemble the final prompt-ready string using `PromptTemplate`.

## 3. Acceptance Criteria
- [x] All usecases implemented in `internal/usecase`.
- [x] Truncation logic is thoroughly unit tested (target >80% coverage).
- [x] Token budget logic and parallel execution (`errgroup`) implemented correctly without race conditions.
