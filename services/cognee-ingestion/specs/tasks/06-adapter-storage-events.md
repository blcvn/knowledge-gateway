---
id: TASK-ING-06
title: Implement Storage, Event, and Scraper Adapters
service: cognee-ingestion
feature: FEAT-ING-002
status: Done
---

## Objective
Implement MinIO file storage, NATS publisher, and URL scraper adapters.

## Files to Create/Update
- `internal/adapter/storage/minio_adapter.go`: Implement `FileStorage` using MinIO/S3.
- `internal/adapter/event/nats_publisher.go`: Implement `EventPublisher` using NATS JetStream.
- `internal/adapter/scraper/url_scraper.go`: Implement URL scraping (e.g., colly or rod).
- Related `*_test.go` files.

## Acceptance Criteria
- FileStorage correctly handles object uploads and deletions.
- EventPublisher successfully publishes the `cognee.data.ingested` event.
- UrlScraper successfully fetches and extracts page content as text.
- Unit/Integration tests pass with >= 80% coverage.
