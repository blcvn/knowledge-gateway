---
id: TASK-PRO-003
title: Data Models & Repositories
service: sm-profile
status: Done
priority: P0
created: 2026-05-11
---

# Data Models & Repositories

## Objective
Implement the storage and persistence adapters.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-profile
title: Technical Design — sm-profile
service: sm-profile
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-profile

> **Group**: Supermemory | **gRPC Port**: 9074 | **Health Port**: 9119

> **🚨 DEPRECATION NOTICE**: This specification is obsolete. The service has been merged into `sm-engine` (Ref: [ARCH-007-merge-sm-engine]).


## 1. Service Overview

User profiles (static preferences + dynamic learned traits) and organization settings. Auto-updated from memory events to build personalization context.

## 2. Domain Layer

- **OrganizationSettings**: id, org_id, should_llm_filter, filter_prompt, include_items[], exclude_items[], google_drive_custom_key (client_id/secret), notion_custom_key, onedrive_custom_key, updated_at
- **UserProfile**: org_id, user_id, static_preferences (JSONB), dynamic_traits (JSONB), updated_at
- **ResetResult**: deleted_connections, deleted_document_batches, deleted_documents_approx, deleted_memory_rows, deleted_extra_spaces, cleared_default_space_context, settings_reset

## 3. gRPC API

```protobuf
service SmProfileService {
  rpc GetProfile(GetProfileRequest) returns (Profile);
  rpc UpdateProfile(UpdateProfileRequest) returns (Profile);
  rpc GetDynamicTraits(GetTraitsRequest) returns (TraitsResponse);
  rpc GetSettings(GetSettingsRequest) returns (SettingsResponse);
  rpc UpdateSettings(UpdateSettingsRequest) returns (SettingsResponse);
  rpc ResetSettings(ResetRequest) returns (ResetResponse);
}
```

## 4. NATS Events

| Direction | Subject | Purpose |
|-----------|---------|---------|
| Subscribe | `sm.memory.created` | Update dynamic traits from new memories |
| Subscribe | `sm.memory.forgotten` | Adjust trait scores when memories decay |

## 5. Data Model

### Tables
- `organization_settings`: id(PK), org_id(UNIQUE), should_llm_filter, filter_prompt, include_items(TEXT[]), exclude_items(TEXT[]), google_drive_client_id, google_drive_client_secret, notion_client_id, notion_client_secret, onedrive_client_id, onedrive_client_secret, updated_at
- `user_profiles`: org_id, user_id — composite PK, static_preferences(JSONB), dynamic_traits(JSONB), trait_version, updated_at

## 6. Observability

- **Metrics**: profile_get_total, settings_update_total, dynamic_trait_update_total
- **Health**: gRPC + HTTP /healthz on port 9119

---

> **Next Steps**: FEAT-001 (Profile CRUD), FEAT-002 (Dynamic Trait Learning), FEAT-003 (Settings Reset with cascade)

```

## Acceptance Criteria
- [x] Database schema / migrations created.
- [x] Repository implementations accurately query the data models.
