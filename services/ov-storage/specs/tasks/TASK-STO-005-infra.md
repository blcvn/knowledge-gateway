---
id: TASK-STO-005
title: Implement Infrastructure & Bootstrap for ov-storage
service: ov-storage
status: Done
---

# TASK-STO-005: Implement Infrastructure & Bootstrap

## Objective
Implement external infrastructure integrations (KMS, Parsers) and bootstrap the monolithic `ov-storage` service.

## Requirements
1. **KMS Backends**:
   - Implement integrations for Vault, AWS KMS, and Local KMS backends matching the `KMSBackend` interface.
2. **Parsers Integration**:
   - Wrap internal/external parsers (`tree-sitter` for code, heading-chunker for markdown, page-chunker for PDF) to comply with the Resource Domain interfaces.
3. **Bootstrap (Wire & Server)**:
   - Configure dependency injection using Google Wire (`infra/wire/wire.go`).
   - Register `OvFsService`, `OvCryptoService`, and `OvResourceService` onto a single gRPC server listening on port `9051`.
   - Setup configuration loading (`config.go`).

## Acceptance Criteria
- [x] Wire successfully generates dependency graphs connecting Domain, Usecase, and Adapter layers.
- [x] The service starts up, connects to PostgreSQL, NATS, and exposes port `9051` successfully.
- [x] Parsers and KMS backends are correctly initialized based on configuration.
