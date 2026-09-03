# Solution: SOL-MB-007 — Shared Infrastructure (LLM Adapters, Tokenizer, OTel)

**CR:** [CR-MB-007](../CR-MB-007-Shared-Infrastructure.md)  
**Wave:** 4 (Access Layer — tuy nhiên tokenizer cần sớm cho Wave 1)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `pkg/` shared packages — các building blocks được dùng bởi TẤT CẢ memobase services. Đây là **cross-cutting concern layer**, cần được xây dựng song song với hoặc trước các services.

### Phân loại ưu tiên

| Package | Needed by Wave | Priority |
|---|---|---|
| `pkg/tokenizer/` | Wave 1 (ingestion: đếm tokens) | **Xây dựng đầu tiên** |
| `pkg/adapters/llm/` | Wave 2 (engine: LLM calls) | Wave 2 |
| `pkg/adapters/embedder/` | Wave 2 (engine: embeddings) | Wave 2 |
| `pkg/prompt/` | Wave 2 (engine: prompt templates) | Wave 2 |
| `pkg/resilience/` | Wave 2 (circuit breaker + bulkhead) | Wave 2 |
| `pkg/config/` | Wave 1 (all services: config loading) | Wave 1 |
| `pkg/observability/` | Wave 4 (metrics + tracing) | Wave 4 |
| `pkg/middleware/` | Wave 4 (gateway middleware) | Wave 4 |

---

## 2. Package Structure

```
vnp-memory/
└── pkg/
    ├── tokenizer/
    │   ├── tokenizer.go      # Tokenizer interface
    │   └── tiktoken.go       # tiktoken-go implementation
    ├── adapters/
    │   ├── llm/
    │   │   ├── client.go     # LLMClient interface + Options
    │   │   ├── bifrost.go    # Bifrost gateway adapter
    │   │   ├── openai.go     # OpenAI compatible (incl. Ollama, vLLM)
    │   │   └── doubao.go     # ByteDance Doubao adapter
    │   └── embedder/
    │       ├── client.go     # EmbedderClient interface
    │       ├── openai.go     # OpenAI embeddings
    │       ├── jina.go       # Jina AI embeddings
    │       ├── ollama.go     # Ollama self-hosted
    │       └── disabled.go   # No-op embedder
    ├── prompt/
    │   ├── provider.go       # PromptProvider interface
    │   ├── registry.go       # map[language]PromptProvider
    │   ├── en/
    │   │   ├── provider.go
    │   │   └── *.go          # Template constants/functions
    │   └── zh/
    │       ├── provider.go
    │       └── *.go
    ├── resilience/
    │   ├── circuit_breaker.go # sony/gobreaker wrapper
    │   ├── retry.go           # Exponential backoff
    │   └── bulkhead.go        # Semaphore concurrency control
    ├── config/
    │   ├── loader.go          # Viper YAML + MEMOBASE_* ENV override
    │   └── validator.go       # Config validation + startup checks
    ├── observability/
    │   ├── tracer.go          # OTel tracer setup
    │   ├── metrics.go         # Prometheus metrics definitions
    │   ├── health.go          # gRPC health + HTTP /healthz endpoints
    │   └── pool_monitor.go    # DB pool utilization monitor
    └── middleware/
        ├── auth/              # JWT + API key extraction
        ├── logging/           # slog structured access log
        ├── tracing/           # OTel trace propagation
        ├── recovery/          # Panic recovery → 500
        ├── ratelimit/         # Redis sliding window
        └── validation/        # Request validation
```

---

## 3. pkg/tokenizer — tiktoken-go Wrapper

```go
// pkg/tokenizer/tokenizer.go

package tokenizer

type Tokenizer interface {
    Count(text string) int
    CountMessages(messages []ChatMessage) int
    TruncateToTokens(text string, maxTokens int) string
    CountBlob(blobData any, blobType string) int
}

type ChatMessage struct {
    Role    string
    Content string
}

// pkg/tokenizer/tiktoken.go

import "github.com/pkoukk/tiktoken-go"

type TiktokenTokenizer struct {
    enc *tiktoken.Tiktoken
}

func New(model string) (*TiktokenTokenizer, error) {
    enc, err := tiktoken.EncodingForModel(model)  // "gpt-4o" → cl100k_base
    if err != nil {
        return nil, fmt.Errorf("tiktoken init: %w", err)
    }
    return &TiktokenTokenizer{enc: enc}, nil
}

func (t *TiktokenTokenizer) Count(text string) int {
    return len(t.enc.Encode(text, nil, nil))
}

func (t *TiktokenTokenizer) CountMessages(messages []ChatMessage) int {
    total := 3  // base overhead per request
    for _, m := range messages {
        total += 4  // message overhead
        total += t.Count(m.Role)
        total += t.Count(m.Content)
    }
    return total
}

func (t *TiktokenTokenizer) TruncateToTokens(text string, maxTokens int) string {
    tokens := t.enc.Encode(text, nil, nil)
    if len(tokens) <= maxTokens {
        return text
    }
    truncated := t.enc.Decode(tokens[:maxTokens])
    return string(truncated)
}

func (t *TiktokenTokenizer) CountBlob(blobData any, blobType string) int {
    switch blobType {
    case "chat":
        if data, ok := blobData.(ChatBlobData); ok {
            return t.CountMessages(data.Messages)
        }
    case "doc", "summary":
        if text, ok := blobData.(string); ok {
            return t.Count(text)
        }
    }
    return 0
}
```

---

## 4. pkg/adapters/llm — LLM Client

### 4.1 Interface

```go
// pkg/adapters/llm/client.go

package llm

type LLMClient interface {
    Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
    CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error)
}

type RequestConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    JSONMode    bool
}

type Option func(*RequestConfig)

func WithJSONMode() Option      { return func(c *RequestConfig) { c.JSONMode = true } }
func WithModel(m string) Option  { return func(c *RequestConfig) { c.Model = m } }
func WithMaxTokens(n int) Option { return func(c *RequestConfig) { c.MaxTokens = n } }
```

### 4.2 Bifrost Adapter

```go
// pkg/adapters/llm/bifrost.go

// Bifrost là production LLM gateway với auto-failover giữa OpenAI, Anthropic, Gemini
// API: OpenAI-compatible endpoint tại custom BaseURL

type BifrostClient struct {
    httpClient  *http.Client
    baseURL     string
    apiKey      string
    defaultConf RequestConfig
}

func NewBifrostClient(baseURL, apiKey, model string) *BifrostClient {
    return &BifrostClient{
        httpClient:  &http.Client{Timeout: 60 * time.Second},
        baseURL:     baseURL,
        apiKey:      apiKey,
        defaultConf: RequestConfig{Model: model, MaxTokens: 1024},
    }
}

func (c *BifrostClient) CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error) {
    cfg := c.defaultConf
    for _, opt := range opts {
        opt(&cfg)
    }

    reqBody := map[string]any{
        "model": cfg.Model,
        "messages": []map[string]string{{"role": "user", "content": prompt}},
        "max_tokens": cfg.MaxTokens,
    }
    if cfg.JSONMode {
        reqBody["response_format"] = map[string]string{"type": "json_object"}
    }

    // POST to Bifrost endpoint (OpenAI-compatible)
    resp, err := c.postJSON(ctx, "/v1/chat/completions", reqBody)
    // Parse response.choices[0].message.content as JSON
    return extractJSONContent(resp)
}
```

### 4.3 OpenAI Compatible Adapter

```go
// pkg/adapters/llm/openai.go

// Dùng cho: OpenAI direct, vLLM, LM Studio, Ollama (với -compat endpoint)
type OpenAICompatClient struct {
    client      *openai.Client
    model       string
    defaultConf RequestConfig
}

// Ollama integration:
// openai.NewClientWithOptions(
//     openai.WithBaseURL("http://localhost:11434/v1"),
//     openai.WithAPIKey("ollama"),
// )
```

### 4.4 Doubao Adapter

```go
// pkg/adapters/llm/doubao.go

// ByteDance Doubao — cung cấp LLM response caching (giảm cost ~60% với repeated prompts)
type DoubaoClient struct {
    httpClient *http.Client
    apiKey     string
    region     string
    model      string
}

// API endpoint: https://ark.cn-beijing.volces.com/api/v3/chat/completions
// Auth: Authorization: Bearer {apiKey}
// Cache: Doubao tự động cache responses → cost reduction
```

---

## 5. pkg/adapters/embedder — Embedding Client

```go
// pkg/adapters/embedder/client.go

type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
    Dimension() int
    IsEnabled() bool
}

// pkg/adapters/embedder/jina.go
type JinaEmbedder struct {
    httpClient *http.Client
    apiKey     string
    model      string   // "jina-embeddings-v3"
    dimension  int      // 1024
}

func (j *JinaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    reqBody := map[string]any{
        "model": j.model,
        "input": texts,
    }
    // POST https://api.jina.ai/v1/embeddings
    // Authorization: Bearer {apiKey}
    resp, err := j.postJSON(ctx, "https://api.jina.ai/v1/embeddings", reqBody)
    return extractEmbeddings(resp)
}

// pkg/adapters/embedder/ollama.go
type OllamaEmbedder struct {
    httpClient *http.Client
    baseURL    string   // "http://localhost:11434"
    model      string   // "nomic-embed-text"
}

func (o *OllamaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
    // POST {baseURL}/api/embeddings
    reqBody := map[string]string{"model": o.model, "prompt": query}
    resp, _ := o.postJSON(ctx, o.baseURL+"/api/embeddings", reqBody)
    return extractEmbedding(resp)
}

// pkg/adapters/embedder/disabled.go
type DisabledEmbedder struct{}
func (d *DisabledEmbedder) IsEnabled() bool { return false }
func (d *DisabledEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    return nil, ErrEmbeddingDisabled
}
```

---

## 6. pkg/resilience — Circuit Breaker + Retry + Bulkhead

### 6.1 Circuit Breaker

```go
// pkg/resilience/circuit_breaker.go

import "github.com/sony/gobreaker"

type CircuitBreaker struct {
    cb *gobreaker.CircuitBreaker
}

func NewCircuitBreaker(name string) *CircuitBreaker {
    settings := gobreaker.Settings{
        Name:        name,
        MaxRequests: 3,                // requests in half-open state
        Interval:    1 * time.Minute,  // reset interval
        Timeout:     30 * time.Second, // open → half-open timeout
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            // Trip when >50% failure in last 10 requests
            if counts.Requests < 10 {
                return false
            }
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return failureRatio >= 0.5
        },
    }
    return &CircuitBreaker{cb: gobreaker.NewCircuitBreaker(settings)}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    _, err := cb.cb.Execute(func() (interface{}, error) {
        return nil, fn()
    })
    if errors.Is(err, gobreaker.ErrOpenState) {
        return ErrCircuitOpen
    }
    return err
}
```

### 6.2 Exponential Backoff Retry

```go
// pkg/resilience/retry.go

type RetryConfig struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    IsRetryable  func(error) bool
}

var DefaultRetryConfig = RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    IsRetryable: func(err error) bool {
        // Retry on: unavailable, deadline exceeded, internal (transient)
        code := status.Code(err)
        return code == codes.Unavailable ||
               code == codes.DeadlineExceeded ||
               code == codes.ResourceExhausted
    },
}

func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    delay := cfg.InitialDelay
    var lastErr error
    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        if attempt > 0 {
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return ctx.Err()
            }
            delay = min(time.Duration(float64(delay)*cfg.Multiplier), cfg.MaxDelay)
        }
        lastErr = fn()
        if lastErr == nil {
            return nil
        }
        if !cfg.IsRetryable(lastErr) {
            return lastErr
        }
    }
    return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
```

### 6.3 Bulkhead (Semaphore)

```go
// pkg/resilience/bulkhead.go

type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead {
    return &Bulkhead{sem: make(chan struct{}, maxConcurrent)}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
    select {
    case b.sem <- struct{}{}:   // acquire slot
        defer func() { <-b.sem }()  // release slot
        return fn()
    case <-ctx.Done():
        return ctx.Err()
    }
}
// Default: max 10 concurrent LLM calls per engine instance
```

---

## 7. pkg/config — Viper + ENV Override

```go
// pkg/config/loader.go

import "github.com/spf13/viper"

func Load(configPath string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(configPath)
    v.SetEnvPrefix("MEMOBASE")  // MEMOBASE_* prefix
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

    if err := v.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    return &cfg, nil
}

// ENV override examples:
// MEMOBASE_LLM_API_KEY=sk-xxx         → cfg.LLM.APIKey
// MEMOBASE_LANGUAGE=zh                → cfg.Language
// MEMOBASE_ENABLE_EVENT_EMBEDDING=false → cfg.EnableEventEmbedding
// MEMOBASE_MAX_CHAT_BLOB_BUFFER_TOKEN_SIZE=2048 → cfg.MaxChatBlobBufferTokenSize
```

---

## 8. pkg/observability — OTel + Prometheus

### 8.1 OTel Tracer Setup

```go
// pkg/observability/tracer.go

func InitTracer(serviceName, endpoint string) (*trace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint(endpoint),  // OTel Collector
        otlptracehttp.WithInsecure(),
    )
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
        trace.WithSampler(trace.AlwaysSample()),
    )
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    return tp, nil
}

// Span names used across memobase:
// process_blobs          (engine: orchestrator)
// llm_call.entry_summary (engine: LLM call #1)
// llm_call.extract_profile (engine: LLM call #2)
// llm_call.merge_yolo    (engine: LLM call #3)
// embed.event            (engine/event: embedding call)
// db.query.profiles      (context: PG query)
// redis.get.profiles     (context: Redis cache)
// grpc.{service}.{method}
```

### 8.2 Prometheus Metrics

```go
// pkg/observability/metrics.go

var (
    HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_http_requests_total",
        Help: "Total HTTP requests",
    }, []string{"path", "method", "status", "project_id"})

    HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "memobase_http_request_duration_ms",
        Help:    "HTTP request latency",
        Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000},
    }, []string{"path", "method"})

    LLMCalls = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_llm_calls_total",
        Help: "Total LLM API calls",
    }, []string{"provider", "model", "prompt_type", "status"})

    LLMDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "memobase_llm_duration_ms",
        Help:    "LLM call latency",
        Buckets: []float64{100, 500, 1000, 2000, 5000, 10000},
    }, []string{"provider", "prompt_type"})

    BufferFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_buffer_flush_total",
        Help: "Total buffer flushes",
    }, []string{"project_id", "status"})

    ProfileMergeOps = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_profile_merge_operations_total",
        Help: "Profile merge operations",
    }, []string{"action"})  // add|update|delete

    EmbeddingCalls = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_embedding_calls_total",
    }, []string{"provider", "model", "status"})
)
```

### 8.3 DB Pool Monitor

```go
// pkg/observability/pool_monitor.go

func MonitorDBPool(ctx context.Context, db *sql.DB, serviceName string) {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        for {
            select {
            case <-ticker.C:
                stats := db.Stats()
                if stats.MaxOpenConnections == 0 {
                    continue
                }
                utilization := float64(stats.InUse) / float64(stats.MaxOpenConnections)
                
                // Prometheus metric
                DBPoolUtilization.WithLabelValues(serviceName).Set(utilization)

                if utilization > 0.8 {
                    slog.Warn("DB pool utilization high",
                        "service", serviceName,
                        "in_use", stats.InUse,
                        "max_open", stats.MaxOpenConnections,
                        "utilization_pct", fmt.Sprintf("%.1f%%", utilization*100),
                    )
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

---

## 9. LLM Client với Resilience Wrapper

```go
// Composition pattern: LLM + Circuit Breaker + Retry + Bulkhead + OTel

type ResilientLLMClient struct {
    inner    llm.LLMClient
    cb       *resilience.CircuitBreaker
    bulkhead *resilience.Bulkhead
    tracer   trace.Tracer
}

func (c *ResilientLLMClient) CompleteJSON(ctx context.Context, prompt string, opts ...llm.Option) (json.RawMessage, error) {
    ctx, span := c.tracer.Start(ctx, "llm_call.complete_json")
    defer span.End()

    var result json.RawMessage

    err := c.bulkhead.Execute(ctx, func() error {
        return c.cb.Execute(func() error {
            return resilience.Retry(ctx, resilience.DefaultRetryConfig, func() error {
                var err error
                result, err = c.inner.CompleteJSON(ctx, prompt, opts...)
                return err
            })
        })
    })

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    return result, nil
}
```

---

## 10. Testing Strategy

### Unit Tests
- `TestTiktokenTokenizer_Count` — verify token count matches tiktoken expectations
- `TestTiktokenTokenizer_Truncate` — truncated text has correct token count
- `TestCircuitBreaker_TripsAfterFailures` — 5 failures → circuit opens
- `TestCircuitBreaker_HalfOpenRecovery` — after timeout → allows through
- `TestBulkhead_MaxConcurrent` — 15 goroutines → max 10 active simultaneously
- `TestRetry_ExponentialBackoff` — delays follow exponential pattern
- `TestRetry_NonRetryableError` — fails immediately without retry
- `TestConfigLoader_ENVOverride` — `MEMOBASE_LANGUAGE=zh` overrides config.yaml

### Integration Tests
- `TestBifrostClient_CompleteJSON` — real API call (mark `//go:build integration`)
- `TestJinaEmbedder_Embed` — real API call
- `TestOllamaEmbedder_Embed` — requires local Ollama

---

## 11. Dependency Graph

```
pkg/tokenizer        ← depended by: memobase-ingestion, memobase-engine, memobase-context
pkg/config           ← depended by: ALL services
pkg/adapters/llm     ← depended by: memobase-engine
pkg/adapters/embedder ← depended by: memobase-engine, memobase-event
pkg/prompt           ← depended by: memobase-engine
pkg/resilience       ← depended by: memobase-engine, memobase-context, gateway
pkg/observability    ← depended by: ALL services
pkg/middleware       ← depended by: gateway
```

---

## 12. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| tiktoken-go encoding file download (network) | Thấp | Bundle encoding files vào binary (embed tiktoken data files) |
| Bifrost endpoint không available | Trung bình | Circuit breaker trips → fallback error; configure OpenAI backup |
| Jina API rate limits | Thấp | Bulkhead + retry; Jina có generous free tier |
| Prometheus cardinality explosion (project_id label) | Trung bình | Hạn chế `project_id` label chỉ cho critical metrics; dùng histogram thay counter |
| OTel exporter blocking on slow collector | Thấp | Async batcher với buffer; drop spans nếu collector chậm |
| Config struct mismatch sau schema change | Thấp | `mapstructure:"..."` tags + validation function tại startup |
