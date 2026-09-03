---
id: DOC-S07
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-search — Changelog

## [1.1.0] - 2026-05-10

### Added
- Complete gRPC API with 3 RPCs (GraphSearch, SearchSessions, GetRelevantFacts)
- 5 reranking strategies (RRF, MMR, CrossEncoder, NodeDistance, EpisodeMentions)
- Redis-backed search result caching with 30s TTL
- NATS consumer for cache invalidation (graph extraction events)
- Multi-scope search (edges, nodes, episodes, all)

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold
- Basic Graphiti search client
