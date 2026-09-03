---
id: TASK-ING-05
title: Implement Text Extractors
service: cognee-ingestion
feature: FEAT-ING-002
status: Done
---

## Objective
Implement text extractors for various supported MIME types.

## Files to Create/Update
- `internal/adapter/extractor/registry.go`: Implement MimeType to Extractor routing logic.
- `internal/adapter/extractor/pdf.go`: PDF text extraction.
- `internal/adapter/extractor/docx.go`: DOCX extraction.
- `internal/adapter/extractor/pptx.go`: PPTX extraction.
- `internal/adapter/extractor/csv.go`: CSV/TSV parsing.
- `internal/adapter/extractor/html.go`: HTML to text processing.
- `internal/adapter/extractor/text.go`: Plain text passthrough.
- Related `*_test.go` files with fixture files.

## Acceptance Criteria
- Text extractors correctly parse their respective file formats as per FEAT-ING-002.
- Registry routes correctly based on MimeType.
- Tests with fixture files pass with >= 80% coverage.
