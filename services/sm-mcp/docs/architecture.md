---
id: DOC-S03
service: sm-mcp
version: 1.1.0
status: Deprecated
created: 2026-05-09
updated: 2026-05-10
---

# sm-mcp — Service Architecture

> **Group**: Supermemory | **Pattern**: 4-layer Clean Architecture

> **🚨 DEPRECATION NOTICE**: This architecture is obsolete. The service has been merged into `vnp-gateway` (Ref: [ARCH-008-absorb-sm-mcp-to-gateway]).


## Layer Structure

```
services/sm-mcp/
├── cmd/server/main.go
├── internal/
│   ├── domain/           # Layer 1: entities, value objects, events, errors
│   ├── usecase/          # Layer 2: business logic + port interfaces
│   ├── adapter/          # Layer 3: gRPC handlers, DB repos, NATS pub/sub
│   └── infra/            # Layer 4: config, server, wire DI
```

## External Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| sm-memory | gRPC (9072) | Memory CRUD |
| sm-search | gRPC (9073) | Search + RAG |
| sm-profile | gRPC (9074) | Profile retrieval |
| sm-document | gRPC (9071) | Document listing |

## Storage

None — stateless MCP proxy
