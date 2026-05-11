---
id: TASK-PIP-004
title: "Cross-Service Integration Tests"
service: cross-service
status: Done
priority: P1
linked_feat: QA-001
---

## Objective
Establish end-to-end integration tests confirming data flows seamlessly from ingestion, through cognify, and into search results.

## Scope
1. **Test Infrastructure**:
   - Set up Docker Compose / Testcontainers to launch PostgreSQL, Neo4j, MinIO, NATS, and the services.
   - Prepare a mock Bifrost LLM server with deterministic responses for entity extraction/chunking.
2. **Test Scenarios**:
   - **Happy Path**: Upload a PDF via ingestion API, wait for local cognify pipeline to finish, query via search API, and expect correct results.
   - **Event Flow**: Ensure `cognee.pipeline.completed` is published and triggers search indexing.
   - **Resilience**: Restart `cognee-pipeline` mid-processing and verify the pipeline job resumes correctly.
   - **Isolation**: Test multi-tenancy by ingesting data as Tenant A and ensuring Tenant B yields zero results on search.
   - **Errors**: Upload unsupported files and assert the correct propagation of errors up to the gRPC client.

## Acceptance Criteria
- [x] All E2E test scenarios pass using Docker Compose environments.
- [x] Both standalone (if deployed) and consolidated pipeline modes satisfy the same test suites.
- [x] Zero cross-tenant data leakage is verified via automated tests.
