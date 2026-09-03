---
id: TASK-STO-002
title: Implement Usecase Layer for ov-storage
service: ov-storage
status: Done
---

# TASK-STO-002: Implement Usecase Layer

## Objective
Implement the business logic orchestration (Usecase layer) for the `ov-storage` service, seamlessly integrating FS, Crypto, and Resource workflows without cross-service network calls.

## Requirements
1. **FS Usecases**:
   - `ReadFile`: Retrieve file, locally call crypto to decrypt.
   - `WriteFile`: Locally call crypto to encrypt, then write.
   - `Tree`, `Grep`, `Glob`, `Move`: Implement using `PathLock` concurrency controls.
2. **Crypto Usecases**:
   - `Encrypt`/`Decrypt`: Execute AES-256-GCM logic with OVE1 envelopes.
   - `RotateKey`: Implement Key Rotation Algorithm (re-wrap DEK without re-encrypting ciphertext).
3. **Resource Usecases**:
   - `Ingest`: Orchestrate format detection -> parsing (tree-sitter/markdown/pdf) -> local FS write (which encrypts automatically).
   - Dispatch `ov.resource.ingested` NATS event upon successful ingestion.

## Acceptance Criteria
- [x] Cross-domain calls (`fs` calling `crypto`, `resource` calling `fs`) occur in-memory within the Usecase layer.
- [x] Key Rotation algorithm successfully updates OVE1 headers only.
- [x] Resource ingestion pipeline works end-to-end logically.
