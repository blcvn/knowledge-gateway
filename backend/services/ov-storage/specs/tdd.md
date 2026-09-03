---
id: TDD-ov-storage
title: Technical Design Document — ov-storage
service: ov-storage
version: 1.1.0
status: Ready
created: 2026-05-10
updated: 2026-05-11
linked_sol: SOL-001
linked_adr: ADR-0001
---

# ov-storage — Technical Design Document

## 1. Overview

The `ov-storage` service consolidates the formerly independent `ov-fs`, `ov-crypto`, and `ov-resource` services into a single monolithic storage pipeline. This eliminates synchronous gRPC calls between these tightly coupled domains, allowing file reads, transparent envelope encryption, and resource ingestion to happen seamlessly in-memory.

## 2. Core Algorithms

Based on the OpenViking reference implementation, `ov-storage` implements the following key algorithms:

### 2.1. VikingFS PathLock Algorithm

To support concurrent file system access:
- **Point Lock**: Lock a specific file for Write/Delete.
- **Subtree Lock**: Lock a directory and all its descendants recursively for tree operations (e.g., recursive delete, tree traversal).
- **Move Lock (mv)**: Atomic lock of both source (Subtree) and destination (Subtree) to prevent cycles and ensure consistency during `Move` operations.

### 2.2. Transparent Envelope Encryption (OVE1)

All files written to `ov-storage` are encrypted at rest using envelope encryption:
1. **File Key Generation**: Generate a cryptographically secure 32-byte Data Encryption Key (DEK) and a 12-byte IV for the specific file.
2. **File Encryption**: Encrypt the file content using AES-256-GCM with the DEK.
3. **Key Wrapping**: Call the active KMS backend (Local, Vault, AWS KMS) to encrypt the DEK using the tenant's Root Key.
4. **Envelope Assembly**: Prepend the `OVE1` header, version, KMS provider type, and the wrapped DEK to the ciphertext.
5. **Decryption**: Read the envelope, ask KMS to unwrap the DEK, and decrypt the AES-256-GCM ciphertext.

### 2.3. Key Rotation Algorithm

When a root key is rotated, the system performs a re-wrap without re-encrypting the file contents:
1. Scan for all files wrapped with the old root key version.
2. For each file, decrypt the DEK using the old root key.
3. Re-encrypt the DEK using the new root key.
4. Rewrite the file header with the new wrapped DEK, leaving the AES-256-GCM ciphertext untouched.

### 2.4. Resource Ingestion & Parsing Pipeline

When a new resource (document/code) is submitted:
1. **Format Detection**: Identify the file format (e.g., Go, Python, Markdown, PDF).
2. **Parsing**:
   - *Code*: Use `tree-sitter` to parse the AST. Chunk by logical blocks (functions, classes).
   - *Markdown*: Chunk by section headers.
   - *PDF/Word*: Parse using Go libraries, chunk by page with configurable token overlap.
3. **Write & Encrypt**: The parsed chunks are written securely to VikingFS using the Envelope Encryption algorithm.
4. **Emit Event**: Publish an `ov.resource.ingested` NATS event (which contains the `viking://` URI) so that `ov-search` can generate embeddings and index it.

## 3. Architecture Layer Mapping

- **Domain Layer**: 
  - `fs`: Defines `VikingURI`, `DirectoryTree`, `PathLock`.
  - `crypto`: Defines `EnvelopeEncryption`, `KMSBackend` interface.
  - `resource`: Defines `ParseResult`, `IngestionTask`.
- **Usecase Layer**:
  - `fs`: Implements `ReadFile`, `WriteFile` (delegates locally to `crypto`), `Tree`, `Grep`, `Glob`.
  - `crypto`: Implements `Encrypt`, `Decrypt`, `RotateKey`.
  - `resource`: Implements `Ingest`, `Parse`, `Watch`.
- **Adapter Layer**: 
  - **gRPC**: Exposes 3 distinct services (`OvFsService`, `OvCryptoService`, `OvResourceService`) over a single listener port.
  - **PostgreSQL / SurrealDB**: Metadata persistence.
  - **NATS**: Emits `ov.content.written` and `ov.content.deleted`.
- **Infrastructure Layer**: KMS Backend SDKs (Vault, AWS).

## 4. Acceptance Criteria

- [ ] AC-1: VikingFS operations (read/write/tree/grep/glob) functional.
- [ ] AC-2: Envelope encryption is transparent on all file operations.
- [ ] AC-3: Key rotation functional across all encrypted files without re-encrypting ciphertext.
- [ ] AC-4: Resource ingestion pipeline (parse → write → emit) works end-to-end.
- [ ] AC-5: PathLock (point/subtree/mv) concurrency control correctly prevents races.
- [ ] AC-6: `ov.content.written` and `ov.content.deleted` events emitted to `ov-search`.
