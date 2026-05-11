---
id: TASK-STO-001
title: Implement Domain Layer for ov-storage
service: ov-storage
status: Done
---

# TASK-STO-001: Implement Domain Layer

## Objective
Implement the Domain layer (entities, value objects, and core algorithm interfaces) for the consolidated `ov-storage` service, encompassing file system, cryptography, and resource management domains.

## Requirements
1. **File System (`fs`)**:
   - Define `File`, `Directory`, `VikingURI`, `TieredLevel` entities.
   - Implement the `PathLock` structure and algorithm interfaces (Point Lock, Subtree Lock, Move Lock) to handle concurrent filesystem operations.
2. **Cryptography (`crypto`)**:
   - Define `EncryptionKey` and `EnvelopeEncryption` entities.
   - Define the `KMSBackend` interface (Wrap/Unwrap).
   - Implement the OVE1 Transparent Envelope Encryption structures (DEK, IV generation logic, AES-256-GCM configurations).
3. **Resource (`resource`)**:
   - Define `Resource`, `ParseResult`, `WatchEvent`, `ParserType` entities.
   - Define interfaces for format detection and chunking.

## Acceptance Criteria
- [x] Domain models contain no external framework dependencies.
- [x] PathLock, EnvelopeEncryption, and Resource entities strictly follow definitions from `data-model.md` and `architecture.md`.
