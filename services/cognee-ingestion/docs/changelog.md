---

## [DEPRECATED] - 2026-05-10


id: DOC-S07
service: cognee-ingestion
version: 1.0.0
status: Active
created: 2026-05-09
updated: 2026-05-09
format: Keep a Changelog (keepachangelog.com)
---

# Changelog — cognee-ingestion

All notable changes to this service will be documented in this file.

## [Unreleased]

### Added
- Complete service documentation (README, API, architecture, data model, configuration, runbook)
- TDD spec with Clean Architecture layer definitions

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold with 4-layer Clean Architecture
- gRPC server skeleton on port 9011
- Health check endpoint on port 9091
- Dataset CRUD operations (CreateDataset, DeleteDataset, ListDatasets, GetDatasetStatus)
- File upload via streaming gRPC (AddData)
- Text ingestion (AddText)
- URL scraping (AddUrl)
- NATS event publishing (cognee.data.ingested)
- PostgreSQL dataset/data_item persistence
- MinIO/S3 raw file storage
- Redis upload progress cache
- Multi-tenant isolation via x-tenant-id metadata
