---
id: DOC-S07
service: zep-graph
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-graph — Changelog

## [1.1.0] - 2026-05-10

### Added
- Complete gRPC API with 13 RPCs (fact CRUD, graph data ops, MCP-compatible queries)
- Async entity extraction via NATS consumer (messages.ingested)
- Graphiti HTTP client integration with retry support
- 9-type node ontology with priority hierarchy
- Temporal annotations (valid_at, invalid_at, expired_at)
- MCP-compatible graph exploration APIs
- Dual-group extraction (session + user graphs)

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold
- Neo4j connection setup
- Basic Graphiti client
