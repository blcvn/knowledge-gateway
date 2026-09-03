# SOL-MB-001 — Solution: Blob Ingestion & Buffer Zone Service

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-001 |
| **CR** | CR-MB-001 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/memobase-ingestion` |

---

## 1. Giải pháp

Buffer Zone = incoming blobs stored in buffer before profile extraction to avoid write conflicts.

### `services/memobase-ingestion/internal/usecase/ingest.go` [MODIFY]

```go
func (u *IngestUseCase) IngestBlob(ctx context.Context, req *BlobRequest) (*Blob, error) {
    // 1. Validate + extract text (PDF/DOCX/txt)
    text, _ := u.extractor.Extract(ctx, req.Content, req.MimeType)
    
    // 2. Store in buffer zone (PostgreSQL temp table, TTL 24h)
    blobID, _ := u.bufferRepo.Store(ctx, &BufferEntry{
        TenantID: req.TenantID, UserID: req.UserID,
        Content: text, Source: req.Source,
    })
    
    // 3. Publish to NATS for async profile extraction
    u.pub.Publish(ctx, "memobase.blob.buffered", map[string]string{"blob_id": blobID})
    
    return &Blob{ID: blobID, Status: "buffered"}, nil
}
```

## 2. File Changes

| File | Action |
|---|---|
| `services/memobase-ingestion/internal/usecase/ingest.go` | MODIFY — buffer zone |
| `services/memobase-ingestion/internal/adapter/pg/buffer_repo.go` | NEW |
| `deployment/dev/migrations/0XX_memobase_buffer.sql` | NEW |

## 3. Acceptance Criteria

- [ ] Blob accepted in < 100ms (async extraction)
- [ ] Buffer TTL 24h (auto-cleanup)
- [ ] NATS event triggers profile extraction

