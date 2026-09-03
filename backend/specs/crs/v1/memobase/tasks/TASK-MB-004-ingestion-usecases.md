# TASK-MB-004 — `services/memobase-ingestion` Use Cases, NATS & gRPC Server

**Wave:** 1 (Data Layer)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-003 (domain + repositories)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-MB-001 §5, §6, §7, §8](../solutions/SOL-MB-001-Blob-Ingestion-Buffer-Zone.md)
**Trạng thái:** ✅ Implemented

---

## Mục tiêu

Hoàn thiện `services/memobase-ingestion/` với: InsertBlob use case (token counting + auto-flush), FlushBuffer use case (atomic lock + NATS), AutoFlushScheduler, NATS subscriber (engine events), gRPC handler, và monolith bootstrap.

---

## Các file cần tạo

### 1. `internal/usecase/insert_blob.go` — Insert Blob

```go
type InsertBlobConfig struct {
    MaxChatBlobTokenSize int           // default: 1024
    PersistentChatBlobs  bool          // default: false
}

type InsertBlobUseCase struct {
    blobRepo   port.BlobRepository
    bufferRepo port.BufferRepository
    tokenizer  tokenizer.Tokenizer
    publisher  port.EventPublisher
    flushUC    *FlushBufferUseCase  // injected (lazy to avoid circular)
    config     InsertBlobConfig
}

func (uc *InsertBlobUseCase) Execute(ctx context.Context, req InsertBlobRequest) (*InsertBlobResponse, error) {
    // 1. Validate blob data
    if err := req.BlobData.Validate(); err != nil {
        return nil, err
    }

    // 2. Persist blob to general_blobs
    blob := &domain.Blob{
        UserID: req.UserID, ProjectID: req.ProjectID,
        BlobType: req.BlobData.BlobType(), BlobData: req.BlobData,
        AdditionalFields: req.AdditionalFields,
    }
    saved, err := uc.blobRepo.Save(ctx, blob)
    if err != nil { return nil, err }

    // 3. Count tokens
    tokenCount := uc.tokenizer.CountBlob(req.BlobData, string(req.BlobData.BlobType()))

    // 4. Insert buffer zone (idle)
    entry := &domain.BufferZone{
        UserID: req.UserID, ProjectID: req.ProjectID,
        BlobID: saved.ID, BlobType: saved.BlobType,
        TokenSize: tokenCount, Status: domain.BufferStatusIdle,
    }
    if _, err := uc.bufferRepo.Save(ctx, entry); err != nil { return nil, err }

    // 5. Check total idle tokens
    totalTokens, _ := uc.bufferRepo.GetTotalIdleTokens(ctx, req.UserID, req.ProjectID)

    // 6. Auto-flush if threshold exceeded (non-blocking!)
    flushTriggered := false
    if totalTokens >= uc.config.MaxChatBlobTokenSize {
        flushTriggered = true
        go func() {
            bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
            defer cancel()
            uc.flushUC.Execute(bgCtx, FlushBufferRequest{
                UserID: req.UserID, ProjectID: req.ProjectID,
                BlobType: string(req.BlobData.BlobType()), Sync: false,
            })
        }()
    }

    return &InsertBlobResponse{BlobID: saved.ID.String(), FlushTriggered: flushTriggered}, nil
}
```

### 2. `internal/usecase/flush_buffer.go` — Flush Buffer

```go
type FlushBufferUseCase struct {
    bufferRepo port.BufferRepository
    blobRepo   port.BlobRepository
    publisher  port.EventPublisher
    config     FlushBufferConfig
}

type FlushBufferRequest struct {
    UserID    uuid.UUID
    ProjectID string
    BlobType  string
    Sync      bool
}

func (uc *FlushBufferUseCase) Execute(ctx context.Context, req FlushBufferRequest) (*FlushBufferResponse, error) {
    blobType := domain.BlobType(req.BlobType)

    // 1. Atomic status lock (idempotent — returns empty if another flush running)
    acquired, err := uc.bufferRepo.AcquireProcessingLock(ctx, req.UserID, req.ProjectID, blobType)
    if err != nil { return nil, err }
    if len(acquired) == 0 {
        return &FlushBufferResponse{Skipped: true}, nil  // Another flush in progress
    }

    bufferIDs := extractIDs(acquired)

    if req.Sync {
        // Sync mode: direct gRPC call to engine (for testing / force flush)
        // This path is rarely used in production
        return &FlushBufferResponse{BlobsFlushed: len(acquired)}, nil
    }

    // Async mode: publish NATS event (engine picks up)
    payload := port.BufferReadyPayload{
        UserID:    req.UserID.String(),
        ProjectID: req.ProjectID,
        BufferIDs: uuidsToStrings(bufferIDs),
        BlobType:  string(blobType),
    }
    if err := uc.publisher.PublishBufferReady(ctx, payload); err != nil {
        // Rollback: reset status to idle
        uc.bufferRepo.MarkFailed(ctx, bufferIDs, req.ProjectID, "nats publish failed")
        return nil, fmt.Errorf("publish buffer ready: %w", err)
    }

    return &FlushBufferResponse{BlobsFlushed: len(acquired)}, nil
}
```

### 3. `internal/usecase/auto_flush.go` — Background Scheduler

```go
type AutoFlushScheduler struct {
    bufferRepo port.BufferRepository
    flushUC    *FlushBufferUseCase
    config     AutoFlushConfig
}

type AutoFlushConfig struct {
    CheckInterval time.Duration  // default: 5 minutes
    IdleTimeout   time.Duration  // default: 1 hour
}

func (s *AutoFlushScheduler) Run(ctx context.Context) {
    ticker := time.NewTicker(s.config.CheckInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            s.checkAndFlush(ctx)
        case <-ctx.Done():
            slog.Info("auto-flush scheduler stopped")
            return
        }
    }
}

func (s *AutoFlushScheduler) checkAndFlush(ctx context.Context) {
    users, err := s.bufferRepo.GetUsersWithStaleIdleBuffers(ctx, s.config.IdleTimeout)
    if err != nil {
        slog.Warn("auto-flush: query failed", "error", err)
        return
    }

    for _, user := range users {
        user := user
        go func() {
            flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            result, err := s.flushUC.Execute(flushCtx, FlushBufferRequest{
                UserID: user.UserID, ProjectID: user.ProjectID,
                BlobType: "chat", Sync: false,
            })
            if err != nil {
                slog.Warn("auto-flush: flush failed", "user_id", user.UserID, "error", err)
            } else if result.Skipped {
                slog.Debug("auto-flush: skipped (another flush in progress)", "user_id", user.UserID)
            }
        }()
    }
}
```

### 4. `internal/adapter/event/subscriber.go` — NATS Consumer

```go
type Subscriber struct {
    bufferRepo port.BufferRepository
    blobRepo   port.BlobRepository
    config     IngestionConfig
    js         nats.JetStreamContext
}

func (s *Subscriber) Start(ctx context.Context) error {
    // Subscribe to engine.completed
    s.js.Subscribe("memobase.engine.completed",
        s.HandleEngineCompleted,
        nats.Durable("memobase-ingestion-completed"),
        nats.AckExplicit(),
    )
    // Subscribe to engine.failed
    s.js.Subscribe("memobase.engine.failed",
        s.HandleEngineFailed,
        nats.Durable("memobase-ingestion-failed"),
        nats.AckExplicit(),
    )
    // Subscribe to user.deleted (cleanup in-flight buffers)
    s.js.Subscribe("memobase.admin.user.deleted",
        s.HandleUserDeleted,
        nats.Durable("memobase-ingestion-user-deleted"),
        nats.AckExplicit(),
    )
    return nil
}

func (s *Subscriber) HandleEngineCompleted(msg *nats.Msg) {
    var payload struct {
        BufferIDs []string `json:"buffer_ids"`
        ProjectID string   `json:"project_id"`
    }
    if err := json.Unmarshal(msg.Data, &payload); err != nil {
        slog.Error("invalid engine.completed payload", "error", err)
        msg.Ack()  // Don't retry bad messages
        return
    }

    bufferIDs := stringsToUUIDs(payload.BufferIDs)
    if err := s.bufferRepo.MarkDone(context.Background(), bufferIDs, payload.ProjectID); err != nil {
        slog.Warn("mark buffer done failed", "error", err)
        msg.Nak()
        return
    }

    // If persistent_chat_blobs=false, delete raw blobs after successful processing
    if !s.config.PersistentChatBlobs {
        // Get blob IDs from buffer entries then delete
        // (buffer entries have blob_id reference)
        s.blobRepo.DeleteBatch(context.Background(), bufferIDs, payload.ProjectID)
    }

    msg.Ack()
}

func (s *Subscriber) HandleEngineFailed(msg *nats.Msg) {
    var payload struct {
        BufferIDs []string `json:"buffer_ids"`
        ProjectID string   `json:"project_id"`
        Error     string   `json:"error"`
    }
    json.Unmarshal(msg.Data, &payload)
    bufferIDs := stringsToUUIDs(payload.BufferIDs)
    s.bufferRepo.MarkFailed(context.Background(), bufferIDs, payload.ProjectID, payload.Error)
    msg.Ack()
}

func (s *Subscriber) HandleUserDeleted(msg *nats.Msg) {
    // PostgreSQL CASCADE already handles DB cleanup
    // Just ack immediately
    msg.Ack()
}
```

### 5. `internal/adapter/grpc/handler.go` — gRPC Handler

```go
type IngestionHandler struct {
    ingestionv1.UnimplementedIngestionServiceServer
    insertBlobUC    *usecase.InsertBlobUseCase
    flushBufferUC   *usecase.FlushBufferUseCase
    blobRepo        port.BlobRepository
    bufferRepo      port.BufferRepository
}

func (h *IngestionHandler) InsertBlob(ctx context.Context, req *ingestionv1.InsertBlobRequest) (*ingestionv1.InsertBlobResponse, error) {
    // 1. Deserialize blob data
    blobData, err := domain.DeserializeBlobData(req.BlobData, domain.BlobType(req.BlobType))
    if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid blob data: %v", err) }

    userID, err := uuid.Parse(req.UserId)
    if err != nil { return nil, status.Error(codes.InvalidArgument, "invalid user_id") }

    result, err := h.insertBlobUC.Execute(ctx, usecase.InsertBlobRequest{
        UserID: userID, ProjectID: req.ProjectId,
        BlobData: blobData,
        AdditionalFields: mapProtoToGo(req.AdditionalFields),
    })
    if err != nil { return nil, mapDomainError(err) }

    return &ingestionv1.InsertBlobResponse{
        BlobId:         result.BlobID,
        FlushTriggered: result.FlushTriggered,
    }, nil
}

func (h *IngestionHandler) FlushBuffer(ctx context.Context, req *ingestionv1.FlushBufferRequest) (*ingestionv1.FlushBufferResponse, error)
func (h *IngestionHandler) GetBufferCapacity(ctx context.Context, req *ingestionv1.GetBufferCapacityRequest) (*ingestionv1.GetBufferCapacityResponse, error)
func (h *IngestionHandler) GetBlobsForProcessing(ctx context.Context, req *ingestionv1.GetBlobsForProcessingRequest) (*ingestionv1.GetBlobsForProcessingResponse, error)
func (h *IngestionHandler) MarkBufferDone(ctx context.Context, req *ingestionv1.MarkBufferDoneRequest) (*ingestionv1.MarkBufferDoneResponse, error)
func (h *IngestionHandler) MarkBufferFailed(ctx context.Context, req *ingestionv1.MarkBufferFailedRequest) (*ingestionv1.MarkBufferFailedResponse, error)
```

### 6. `internal/adapter/event/publisher.go`

```go
type NATSPublisher struct {
    js nats.JetStreamContext
}

func (p *NATSPublisher) PublishBufferReady(ctx context.Context, payload port.BufferReadyPayload) error {
    data, err := json.Marshal(payload)
    if err != nil { return err }
    _, err = p.js.Publish("memobase.buffer.ready", data)
    return err
}
```

### 7. Config

```yaml
ingestion:
  server:
    grpc_port: 9041
    health_port: 9091

  buffer:
    max_chat_blob_token_size: 1024      # MEMOBASE_MAX_CHAT_BLOB_BUFFER_TOKEN_SIZE
    flush_interval: 3600s               # idle timeout → auto-flush
    check_interval: 300s                # auto-flush scheduler tick
    max_concurrent_flush: 5
    persistent_chat_blobs: false        # MEMOBASE_PERSISTENT_CHAT_BLOBS

  tokenizer:
    model: "gpt-4o"

  database:
    url: "${DATABASE_URL}"
    pool_size: 25
    max_overflow: 10

  nats:
    url: "${NATS_URL}"
    stream: "memobase"
    subjects:
      publish_buffer_ready: "memobase.buffer.ready"
      subscribe_engine_completed: "memobase.engine.completed"
      subscribe_engine_failed: "memobase.engine.failed"
      subscribe_user_deleted: "memobase.admin.user.deleted"
```

---

## Unit Tests

```
TestInsertBlobUseCase_ValidChatBlob      → blob saved, buffer entry created
TestInsertBlobUseCase_InvalidRole        → "model" role → ErrInvalidBlobRole returned
TestInsertBlobUseCase_TokensCounted      → chat blob → tokenizer called → token_size set
TestInsertBlobUseCase_BelowThreshold     → 500 tokens < 1024 → FlushTriggered=false
TestInsertBlobUseCase_AboveThreshold     → 1025 tokens → FlushTriggered=true, goroutine started
TestFlushBufferUseCase_AcquiresLock      → idle entries → MarkProcessing called
TestFlushBufferUseCase_NothingToFlush    → no idle → Skipped=true
TestFlushBufferUseCase_ConcurrentFlush   → 2 goroutines → only 1 publishes
TestFlushBufferUseCase_AsyncPublishNATS  → publisher.PublishBufferReady called
TestFlushBufferUseCase_NATSFail_Rollback → publisher fails → MarkFailed called
TestAutoFlushScheduler_StaleUsers        → 3 stale users → 3 flush calls
TestAutoFlushScheduler_EmptyBuffers      → no stale → no flush calls
TestAutoFlushScheduler_Stop             → ctx cancel → scheduler exits
TestNATSHandleEngineCompleted_MarksDone  → payload → MarkDone called
TestNATSHandleEngineCompleted_DeleteBlobs → persistent=false → DeleteBatch called
TestNATSHandleEngineCompleted_Persistent  → persistent=true → DeleteBatch NOT called
TestNATSHandleEngineFailed_MarksFailed   → payload → MarkFailed with error
TestGRPCInsertBlob_ValidRequest          → proto req → usecase called → proto response
TestGRPCInsertBlob_InvalidBlobType       → bad type → codes.InvalidArgument
TestGRPCInsertBlob_InvalidUserID         → bad UUID → codes.InvalidArgument
TestGRPCFlushBuffer_Skipped              → Skipped=true → response skipped=true
```

---

## Monolith Bootstrap Integration

**File: `apps/memory/internal/bootstrap/memobase_ingestion.go`**

```go
func bootstrapMemobaseIngestion(ctx context.Context, cfg *config.Config, registry *bus.InProcessRegistry) error {
    db := infra.NewPostgresPool(cfg.Ingestion.Database)
    js := infra.GetNATSJetStream(ctx)
    tok, _ := tokenizer.New(cfg.Ingestion.Tokenizer.Model)

    blobRepo   := postgres.NewBlobRepository(db)
    bufferRepo := postgres.NewBufferRepository(db)
    publisher  := event.NewNATSPublisher(js)

    flushUC    := usecase.NewFlushBufferUseCase(bufferRepo, blobRepo, publisher, cfg.Ingestion.Buffer)
    insertUC   := usecase.NewInsertBlobUseCase(blobRepo, bufferRepo, tok, publisher, flushUC, cfg.Ingestion.Buffer)
    autoFlush  := usecase.NewAutoFlushScheduler(bufferRepo, flushUC, cfg.Ingestion.Buffer)

    subscriber := event.NewSubscriber(bufferRepo, blobRepo, js, cfg.Ingestion)
    subscriber.Start(ctx)

    handler := grpchandler.New(insertUC, flushUC, blobRepo, bufferRepo)

    server := grpc.NewServer()
    ingestionv1.RegisterIngestionServiceServer(server, handler)
    registry.Register("memobase-ingestion", server, bufconn.Listen(1024*1024))

    go autoFlush.Run(ctx)
    return nil
}
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

buf generate services/memobase-ingestion/
go build ./services/memobase-ingestion/...
go test ./services/memobase-ingestion/... -v -count=1 -race

# Integration (requires PostgreSQL + NATS)
go test ./services/memobase-ingestion/... -v -tags integration -count=1
```

---

## Ghi chú triển khai

- `FlushBufferUseCase` phải được inject vào `InsertBlobUseCase` — tránh circular dependency bằng lazy init hoặc interface
- Auto-flush goroutines: dùng `context.WithTimeout(context.Background(), 30s)` — không dùng request context (đã expired)
- `DeleteBatch`: một SQL `DELETE WHERE id = ANY($ids) AND project_id=$p`
- NATS stream `memobase` phải được tạo trước (script setup hoặc trong bootstrap)
