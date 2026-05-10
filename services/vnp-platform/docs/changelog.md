# vnp-platform — Changelog

All notable changes to this service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Initial service structure from consolidation of vnp-admin + vnp-event + ov-admin + zep-admin + sm-auth + sm-analytics + sm-project
- 7 gRPC service definitions on single port :9050
- Unified tenant management across all engines
- Cross-domain event timeline with pgvector semantic search
- JWT + API Key authentication with RBAC
- Usage analytics and token economics tracking
- Project/Space management with container tags
