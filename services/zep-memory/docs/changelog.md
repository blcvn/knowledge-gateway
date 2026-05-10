---

## [DEPRECATED] - 2026-05-10


id: DOC-S07
service: zep-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-memory — Changelog

## [1.1.0] - 2026-05-10

### Added
- Complete gRPC API with 7 RPCs (PutMemory, GetMemory, DeleteMemory, GetMessagesForSession, GetMessage, UpdateMessageMetadata, GetUserContext)
- PutMemory critical path with session upsert and async graph extraction
- GetMemory context assembly with graceful degradation
- NATS events: messages.ingested, memory.deleted
- Inter-service gRPC clients for zep-thread and zep-search
- Complete data model with role_type_enum and partial indexes

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold
- Messages table schema
- Basic PutMemory/GetMemory flows
