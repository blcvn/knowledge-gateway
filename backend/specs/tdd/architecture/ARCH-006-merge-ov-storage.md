---
id: ARCH-006
title: Merge ov-fs + ov-crypto + ov-resource → ov-storage
service: ov-storage
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

`ov-fs` gọi `ov-crypto` cho mọi file read/write (transparent envelope encryption). `ov-resource` gọi `ov-fs` để write parsed content. 3 services tightly coupled via synchronous gRPC, running 3 separate binaries.

## Kiến Trúc Mới

```
services/ov-storage/
├── internal/
│   ├── domain/
│   │   ├── fs/         # File, Directory, PathLock, VikingURI
│   │   ├── crypto/     # EncryptionKey, EnvelopeEncryption, KMSBackend
│   │   └── resource/   # Resource, ParseResult, WatchEvent
│   ├── usecase/
│   │   ├── fs/         # ReadFile, WriteFile (calls crypto locally), Tree, Grep, Glob
│   │   ├── crypto/     # Encrypt, Decrypt, RotateKey
│   │   └── resource/   # Ingest, Parse (tree-sitter/markdown/PDF), Watch, Refresh
│   ├── adapter/grpc/
│   │   ├── fs_handler.go        # OvFsService
│   │   ├── crypto_handler.go    # OvCryptoService
│   │   └── resource_handler.go  # OvResourceService
│   └── infra/
```

**Key**: Envelope encryption becomes local call within fs usecase. `ov.crypto.key.rotated` NATS event becomes internal (no cross-service). External events: `ov.content.written`, `ov.content.deleted`, `ov.resource.ingested` → ov-search.

## Acceptance Criteria

- [ ] AC-1: VikingFS operations (read/write/tree/grep/glob) functional
- [ ] AC-2: Envelope encryption transparent on all file operations
- [ ] AC-3: Key rotation functional across all encrypted files
- [ ] AC-4: Resource ingestion pipeline (parse → write → emit) works end-to-end
- [ ] AC-5: PathLock (point/subtree/mv) concurrent access control preserved
- [ ] AC-6: `ov.content.written` and `ov.content.deleted` events emitted to ov-search
