# ov-storage — OpenViking Unified Storage Service

> **Service**: `ov-storage` | **gRPC Port**: 9051 | **Health**: 9104  
> **Origin**: Consolidated from ov-fs + ov-crypto + ov-resource  
> **Status**: Proposed | **Version**: 0.1.0

---

## Purpose

Unified storage service for the OpenViking procedural context engine. Combines VikingFS (Go-native virtual filesystem), envelope encryption (AES-256-GCM with KMS), and resource ingestion (parser pipeline for markdown, PDF, code). Provides transparent encryption on all file operations and tiered context storage (L0/L1/L2).

## Business Capability

- **VikingFS**: Go-native virtual filesystem with hierarchical path operations (read, write, tree, grep, glob)
- **Envelope Encryption**: AES-256-GCM per-file encryption with KMS key rotation
- **Resource Ingestion**: Parse markdown/PDF/code (tree-sitter), extract sections, write to FS
- **Tiered Context**: L0 (hot, <10KB), L1 (warm, <100KB), L2 (cold, archival)
- **PathLock**: Point/subtree/move locking for concurrent access control
- **File Watching**: Watch directories for changes, emit events

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (3 services: OvFsService + OvCryptoService + OvResourceService) |
| Database | PostgreSQL 17 (metadata, path index) |
| Storage | VikingFS (Go-native, custom) |
| Async | NATS JetStream |

## Quick Start

```bash
cd services/ov-storage
go run cmd/server/main.go
# gRPC: :9051 | Health: :9104
```

## Links

- [Architecture](./architecture.md)
- [Changelog](./changelog.md)

## Owner

OpenViking Engine Team
