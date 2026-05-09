---
id: DOC-S01
service: sm-profile
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-profile

> **Group**: Supermemory | **gRPC Port**: 9074 | **Health Port**: 9119 | **Origin**: Supermemory

## Purpose

User profile management with **static preferences** (explicitly set by users) and **dynamic learned traits** (inferred from memory events). Updated automatically when new memories are created, providing personalization context for search and AI interactions.

### Business Capability

- **Static Preferences**: User-set settings (excludeItems, includeItems, filterPrompt, shouldLLMFilter)
- **Dynamic Traits**: Auto-learned from memory patterns (interests, expertise areas, communication style)
- **Organization Settings**: Custom OAuth keys for connectors (Google Drive, Notion, OneDrive)
- **LLM Filtering**: Per-org settings to filter/include specific content types
- **Settings Reset**: Full account data purge with confirmation

## API Surface

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

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Profile + settings persistence |
| sm-memory | NATS sub | `sm.memory.created` → update dynamic traits |
| Redis | Cache | Profile cache for fast retrieval |

## Owner

- **Team**: VNP Memory — Supermemory
