---
skill_id: SKILL-004
version: 1.0.0
status: active
priority: existing
group: Backend Development
created_at: 2026-04-24
---

# SKILL-004 · Backend Development — Golang

## Mô tả

Phát triển backend hiệu năng cao với Golang — concurrency patterns, memory management, design patterns cho stability và performance.

## Agents sử dụng

- `requirement-parser-agent`
- `semantic-extractor-agent`
- `knowledge-graph-agent`
- `ui-schema-generator-agent`

---

## Năng lực cốt lõi

### 1. Goroutines & Channels

```go
// Worker pool pattern — xử lý N documents song song
func ProcessDocuments(docs []Document, workers int) []Result {
    jobs := make(chan Document, len(docs))
    results := make(chan Result, len(docs))
    
    // Launch workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for doc := range jobs {
                results <- processDoc(doc)
            }
        }()
    }
    
    // Send jobs
    for _, doc := range docs {
        jobs <- doc
    }
    close(jobs)
    
    // Collect results
    go func() {
        wg.Wait()
        close(results)
    }()
    
    var out []Result
    for r := range results {
        out = append(out, r)
    }
    return out
}
```

### 2. Context & Cancellation

```go
// Luôn propagate context — cho timeout và cancellation
func (s *Service) ProcessDocument(ctx context.Context, docID string) error {
    // Create child context với timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // All downstream calls must accept and use ctx
    doc, err := s.repo.GetDocument(ctx, docID)
    if err != nil {
        return fmt.Errorf("fetch document: %w", err)
    }
    
    result, err := s.nlp.Process(ctx, doc.Content)
    if err != nil {
        return fmt.Errorf("NLP processing: %w", err)
    }
    
    return s.repo.SaveResult(ctx, docID, result)
}
```

### 3. Error Handling

```go
// Sentinel errors cho domain errors
var (
    ErrDocumentNotFound  = errors.New("document not found")
    ErrJobAlreadyExists  = errors.New("job already exists")
    ErrInvalidProjectID  = errors.New("invalid project ID")
)

// Error wrapping với context
func (r *Repository) GetDocument(ctx context.Context, id string) (*Document, error) {
    doc, err := r.db.QueryRow(ctx, "SELECT * FROM documents WHERE id = $1", id)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, fmt.Errorf("GetDocument %s: %w", id, ErrDocumentNotFound)
    }
    if err != nil {
        return nil, fmt.Errorf("GetDocument %s: %w", id, err)
    }
    return doc, nil
}

// Check error type at handler level
func (h *Handler) GetDocument(c *gin.Context) {
    doc, err := h.service.GetDocument(c.Request.Context(), c.Param("id"))
    if errors.Is(err, ErrDocumentNotFound) {
        c.JSON(404, ErrorResponse{Code: "DOCUMENT_NOT_FOUND"})
        return
    }
    if err != nil {
        c.JSON(500, ErrorResponse{Code: "INTERNAL_ERROR"})
        return
    }
    c.JSON(200, doc)
}
```

### 4. sync.Pool — Memory Reuse

```go
// Reuse buffers để giảm GC pressure
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func processJSON(data []byte) (string, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // Use buf for processing
    json.NewEncoder(buf).Encode(data)
    return buf.String(), nil
}
```

### 5. Circuit Breaker

```go
// Circuit breaker cho LLM API calls
import "github.com/sony/gobreaker"

var llmBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "llm-api",
    MaxRequests: 5,    // half-open state: max 5 requests
    Interval:    60 * time.Second,
    Timeout:     30 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        failRate := float64(counts.TotalFailures) / float64(counts.Requests)
        return counts.Requests >= 5 && failRate >= 0.6  // open if >60% failure
    },
})

func CallLLM(ctx context.Context, prompt string) (string, error) {
    result, err := llmBreaker.Execute(func() (any, error) {
        return llmClient.Complete(ctx, prompt)
    })
    if err == gobreaker.ErrOpenState {
        return "", fmt.Errorf("LLM service unavailable: circuit breaker open")
    }
    return result.(string), err
}
```

### 6. Database Connection Pooling

```go
// pgxpool setup cho PostgreSQL
func NewDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }
    
    config.MaxConns = 50
    config.MinConns = 5
    config.MaxConnLifetime = 1 * time.Hour
    config.MaxConnIdleTime = 30 * time.Minute
    config.HealthCheckPeriod = 1 * time.Minute
    
    return pgxpool.NewWithConfig(ctx, config)
}
```

### 7. pprof Profiling

```go
// Enable pprof endpoint (development only)
import _ "net/http/pprof"

func init() {
    if os.Getenv("ENABLE_PPROF") == "true" {
        go func() {
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }
}

// Profile: go tool pprof http://localhost:6060/debug/pprof/heap
// CPU:     go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

---

## Project Conventions

```go
// Package structure
// cmd/server/main.go       — entry point
// internal/handler/        — HTTP/gRPC handlers
// internal/service/        — business logic
// internal/repository/     — data access
// internal/domain/         — entities, interfaces
// pkg/                     — reusable packages

// Naming conventions
// Interface: DocumentRepository (not IDocumentRepository)
// Constructor: NewDocumentService (not CreateDocumentService)
// Error vars: ErrDocumentNotFound (not DocumentNotFoundError)
// Context: always first parameter

// Linting (golangci-lint)
// .golangci.yml enabled: errcheck, govet, staticcheck, unused, gosec
```

## Checklist

- [ ] Context propagated đến mọi function call
- [ ] Errors wrapped với context: `fmt.Errorf("operation: %w", err)`
- [ ] Goroutines có proper cleanup (defer wg.Done(), cancel())
- [ ] DB queries parameterized
- [ ] Connection pool configured
- [ ] Circuit breaker cho external API calls
- [ ] pprof disabled trên production
- [ ] No global mutable state (dùng dependency injection)
