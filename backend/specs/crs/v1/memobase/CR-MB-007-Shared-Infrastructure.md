# Change Request: CR-MB-007 — Shared Infrastructure: Observability, LLM Adapters & Tokenizer

**CR ID:** CR-MB-007  
**Component:** `pkg/` [NEW SHARED PACKAGES] | Cross-cutting concerns  
**Priority:** High  
**Status:** In Progress
**Reference:** memobase PRD §12 (NFRs), SRS §4, specs/services/07-shared-packages.md  
**Maps to Python:** `llms/`, `embedders/`, `telemetry.py`, `config.py`

---

## 1. Mô tả

Xây dựng **shared packages** (`pkg/`) và cross-cutting infrastructure cho tất cả memobase microservices:
1. **LLM Adapter** — Bifrost + OpenAI compatible (OpenAI, Ollama, Doubao/vLLM).
2. **Embedding Adapter** — OpenAI, Jina, Ollama, LMStudio.
3. **Tokenizer** — tiktoken-go gpt-4o encoder (token counting, truncation).
4. **Prompt Template Engine** — EN/ZH registry, mustache-like.
5. **Observability** — OpenTelemetry traces + Prometheus metrics.
6. **Circuit Breaker + Retry** — resilience per downstream service.
7. **Configuration** — Viper YAML + ENV override system.
8. **DB Pool Monitoring** — pool utilization logging (warning at >80%).

---

## 2. Vấn đề hiện tại

VNP Memory hiện tại:
- ✅ Có basic OpenAI LLM client.
- ❌ Không có **Bifrost gateway adapter** (production multi-provider).
- ❌ Không có **Doubao/vLLM/Ollama** LLM adapters.
- ❌ Không có **Jina/Ollama** embedding adapters.
- ❌ Không có **tiktoken-go** cho accurate token counting.
- ❌ Không có **prompt template registry** (EN/ZH, per prompt type).
- ❌ Không có **circuit breaker** (sony/gobreaker) per downstream service.
- ❌ Không có **bulkhead** (semaphore for LLM calls, default max=10).
- ❌ OpenTelemetry không đầy đủ — thiếu `X-Process-Time`, request latency histograms.
- ❌ Không có **DB pool monitoring** (warning at >80%).
- ❌ Không có **`MEMOBASE_*` env var override** prefix.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `pkg/adapters/llm/` — LLM Client Interface

```go
// pkg/adapters/llm/client.go

type LLMClient interface {
    Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
    CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error)
}

type Option func(*RequestConfig)

func WithJSONMode() Option { return func(c *RequestConfig) { c.JSONMode = true } }
func WithModel(model string) Option { return func(c *RequestConfig) { c.Model = model } }
func WithMaxTokens(n int) Option { return func(c *RequestConfig) { c.MaxTokens = n } }
func WithTemperature(t float64) Option { return func(c *RequestConfig) { c.Temperature = t } }

// Implementations:

// pkg/adapters/llm/bifrost.go — Bifrost gateway (recommended for production)
// Routes to: OpenAI, Anthropic, Gemini, Azure via single endpoint
// Auto-failover between providers
type BifrostClient struct {
    BaseURL string
    APIKey  string
    Model   string
}

// pkg/adapters/llm/openai.go — Direct OpenAI API
// Compatible with: OpenAI, vLLM, Ollama, LM Studio (OpenAI-compatible endpoints)
type OpenAICompatClient struct {
    BaseURL string   // default: "https://api.openai.com/v1"
    APIKey  string
    Model   string
}

// pkg/adapters/llm/doubao.go — Doubao (ByteDance)
// Cache-optimized: Doubao provides LLM response caching
// Useful for: high-traffic applications reducing LLM cost
type DoubaoClient struct {
    APIKey  string
    Region  string
    Model   string
}
```

### 3.2. [NEW] `pkg/adapters/embedder/` — Embedding Client Interface

```go
// pkg/adapters/embedder/client.go

type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
    Dimension() int
    IsEnabled() bool
}

// Implementations:

// pkg/adapters/embedder/openai.go
// Models: text-embedding-3-small (1536d), text-embedding-3-large (3072d)
type OpenAIEmbedder struct {
    Client     *openai.Client
    Model      string
    Dimensions int
}

// pkg/adapters/embedder/jina.go
// Models: jina-embeddings-v3 (1024d)
// API: https://api.jina.ai/v1/embeddings
type JinaEmbedder struct {
    APIKey string
    Model  string
}

// pkg/adapters/embedder/ollama.go
// Self-hosted via Ollama (nomic-embed-text, etc.)
// API: OpenAI-compatible /api/embeddings
type OllamaEmbedder struct {
    BaseURL string    // default: "http://localhost:11434"
    Model   string
}

// pkg/adapters/embedder/disabled.go
// No-op embedder when enable_event_embedding=false
type DisabledEmbedder struct{}
func (d *DisabledEmbedder) IsEnabled() bool { return false }
func (d *DisabledEmbedder) Embed(...) ([][]float32, error) { return nil, ErrEmbeddingDisabled }
```

### 3.3. [NEW] `pkg/tokenizer/` — tiktoken-go Wrapper

```go
// pkg/tokenizer/tokenizer.go
// Wraps tiktoken-go for accurate token counting (gpt-4o encoder)

type Tokenizer interface {
    Count(text string) int
    CountMessages(messages []ChatMessage) int
    TruncateToTokens(text string, maxTokens int) string
    CountBlob(blob Blob) int
}

// Implementation: tiktoken-go using "cl100k_base" encoding (gpt-4o compatible)
type TiktokenTokenizer struct {
    encoding *tiktoken.Tiktoken
}

// Usage:
// - insert_blob: count tokens → populate buffer_zones.token_size
// - process_blobs: truncate blobs to max_process_token_size (16384)
// - context assembly: enforce max_token_size budget
// - profile truncation: count profile lines
```

### 3.4. [NEW] `pkg/prompt/` — Prompt Template Engine

```go
// pkg/prompt/provider.go

type PromptProvider interface {
    EntrySummary(blobs []blob.Blob, config profile.Schema) string
    ExtractProfile(memoStr string, schema profile.Schema) string
    MergeProfileYOLO(existing []profile.Slot, facts []profile.Slot) string
    SummarizeEvent(memoStr string) string
    TagEvent(memoStr string, tagDefs []profile.EventTagDef) string
    Language() string
}

// Template engine features:
// - Simple string interpolation with {{.Variable}} placeholders
// - Language selection: "en" | "zh" at runtime
// - Per-prompt-type template registration
// - Strict mode: append "Only collect these topics: ..." instruction
// - Tag mode: append "Extract these tags: ..." instruction

// Registry:
var Registry = map[string]PromptProvider{
    "en": &ENPromptProvider{},
    "zh": &ZHPromptProvider{},
}

// Template files:
// pkg/prompt/en/entry_chat_summary.go
// pkg/prompt/en/extract_profile.go
// pkg/prompt/en/merge_profile_yolo.go
// pkg/prompt/en/summarize_event.go
// pkg/prompt/en/tag_event.go
// pkg/prompt/zh/*.go  (same structure, Chinese language)
```

### 3.5. [NEW] `pkg/resilience/` — Circuit Breaker + Retry + Bulkhead

```go
// pkg/resilience/circuit_breaker.go
// sony/gobreaker per downstream service

type CircuitBreakerConfig struct {
    Name          string
    MaxRequests   uint32        // requests allowed in half-open (default: 3)
    Interval      time.Duration // reset interval (default: 1min)
    Timeout       time.Duration // open → half-open timeout (default: 30s)
    ReadyToTrip   func(counts gobreaker.Counts) bool
}

// Default: trip when >50% failure in last 10 requests

// pkg/resilience/retry.go
// Exponential backoff retry

type RetryConfig struct {
    MaxAttempts    int           // default: 3
    InitialDelay   time.Duration // default: 100ms
    MaxDelay       time.Duration // default: 10s
    Multiplier     float64       // default: 2.0
    RetryableCodes []codes.Code  // gRPC: codes.Unavailable, codes.DeadlineExceeded
}

// pkg/resilience/bulkhead.go
// Semaphore for LLM call concurrency control

type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error
// Default: max 10 concurrent LLM calls per engine instance
```

### 3.6. [NEW] `pkg/observability/` — OTel + Prometheus

```go
// pkg/observability/tracer.go

// OpenTelemetry traces:
// - Span per gRPC call (method, status)
// - Span per HTTP request (path, method, status, project_id)
// - Span per DB query (table, operation, latency)
// - Span per LLM call (provider, model, prompt_type, tokens)
// - Span per embedding call (provider, model, text_count)

// Prometheus metrics:
// http_requests_total{path, method, project_id, status}
// http_request_duration_ms{path, method}  (histogram)
// healthcheck_requests_total
// llm_calls_total{provider, model, prompt_type, status}
// llm_duration_ms{provider, prompt_type}
// embedding_calls_total{provider, model, status}
// buffer_flush_total{project_id, status}
// profile_merge_operations_total{action}  // add|update|delete

// pkg/observability/health.go
// Endpoints: /healthz | /readyz | /livez
// gRPC Health v1 (grpc_health_v1.HealthServer)
```

### 3.7. [NEW] `pkg/config/` — Viper + ENV Override

```go
// pkg/config/loader.go
// Load config.yaml + ENV overrides with MEMOBASE_* prefix

type Config struct {
    LLMAPIKey              string  `mapstructure:"llm_api_key"`
    LLMBaseURL             string  `mapstructure:"llm_base_url"`
    BestLLMModel           string  `mapstructure:"best_llm_model"`
    Language               string  `mapstructure:"language"`
    EnableEventEmbedding   bool    `mapstructure:"enable_event_embedding"`
    EmbeddingProvider      string  `mapstructure:"embedding_provider"`
    EmbeddingModel         string  `mapstructure:"embedding_model"`
    EmbeddingDim           int     `mapstructure:"embedding_dim"`
    MaxChatBlobBufferTokenSize int `mapstructure:"max_chat_blob_buffer_token_size"`
    MaxProfileSubtopics    int     `mapstructure:"max_profile_subtopics"`
    ProfileStrictMode      bool    `mapstructure:"profile_strict_mode"`
    PersistentChatBlobs    bool    `mapstructure:"persistent_chat_blobs"`
    CacheUserProfilesTTL   int     `mapstructure:"cache_user_profiles_ttl"`
}

// ENV overrides: MEMOBASE_LLM_API_KEY, MEMOBASE_LANGUAGE, etc.
// Priority: ENV > config.yaml > defaults

// Embedding dimension validation at startup:
// IF current_db_dimension != config.embedding_dim → fail fast with error message
// "Embedding dimension mismatch: config=1536, DB=768. 
//  Run migration or update embedding_dim in config."
```

### 3.8. [NEW] `pkg/middleware/` — Shared gRPC + HTTP Interceptors

```go
// pkg/middleware/auth/     — JWT/APIKey extraction, project context
// pkg/middleware/logging/  — Structured access log (slog JSON)
// pkg/middleware/tracing/  — OTel trace propagation
// pkg/middleware/recovery/ — Panic recovery → 500
// pkg/middleware/ratelimit/ — Redis sliding window (per-project per-endpoint)
// pkg/middleware/validation/ — Request validation

// DB Pool monitoring (warning at >80%):
// pkg/observability/pool_monitor.go

func MonitorPool(ctx context.Context, db *sql.DB, cfg PoolConfig) {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        stats := db.Stats()
        utilization := float64(stats.InUse) / float64(stats.MaxOpenConnections)
        if utilization > 0.8 {
            slog.Warn("DB pool utilization high",
                "in_use", stats.InUse,
                "max_open", stats.MaxOpenConnections,
                "utilization_pct", utilization * 100,
            )
        }
    }
}
```

---

## 4. Configuration

```yaml
# config.yaml (base config, overrideable via MEMOBASE_* env vars)

# LLM Configuration
llm_api_key: "${MEMOBASE_LLM_API_KEY}"
llm_base_url: null                        # MEMOBASE_LLM_BASE_URL
best_llm_model: "gpt-4o-mini"            # MEMOBASE_BEST_LLM_MODEL
language: "en"                            # MEMOBASE_LANGUAGE (en|zh)

# Embedding
enable_event_embedding: true              # MEMOBASE_ENABLE_EVENT_EMBEDDING
embedding_provider: "openai"             # MEMOBASE_EMBEDDING_PROVIDER
embedding_model: "text-embedding-3-small"
embedding_dim: 1536

# Memory Processing
max_chat_blob_buffer_token_size: 1024    # trigger flush
max_profile_subtopics: 15
max_pre_profile_token_size: 128
profile_strict_mode: false
profile_validate_mode: true
persistent_chat_blobs: false

# Context
cache_user_profiles_ttl: 1200            # seconds

# Database
database_url: "${DATABASE_URL}"
database_pool_size: 75
database_max_overflow: 50
database_pool_timeout: 45                # seconds
database_pool_recycle: 300              # seconds
database_pool_pre_ping: true

# Redis
redis_url: "${REDIS_URL}"
```

---

## 5. Acceptance Criteria

- [ ] `MEMOBASE_LANGUAGE=zh` env var → engine uses ZH prompts (overrides config.yaml `language: en`).
- [ ] tiktoken token count: 10 messages ≈ X tokens → `buffer_zones.token_size` matches (±5% margin).
- [ ] `CompleteJSON()` returns valid JSON → engine parses MergeResult without error.
- [ ] Circuit breaker: LLM service returns 500 x 5 consecutive → circuit opens → next call returns ErrCircuitOpen without calling LLM.
- [ ] Bulkhead: 15 concurrent engine flushes → only 10 execute LLM simultaneously (5 queued).
- [ ] Prometheus `/metrics` exposed: `http_request_duration_ms` histogram contains p50/p95/p99.
- [ ] OTel trace: flush pipeline → parent span `process_blobs` with child spans: `llm_call.entry_summary`, `llm_call.extract_profile`, `llm_call.merge_yolo`.
- [ ] DB pool >80%: log warning `"DB pool utilization high"` với `in_use`, `max_open`.
- [ ] Embedding dimension mismatch at startup → process exits with error: `"Embedding dimension mismatch: config=1536, DB=768"`.
- [ ] Ollama embedder: `embedding_provider=ollama, embedding_model=nomic-embed-text` → embeddings generated locally.
- [ ] Jina embedder: `embedding_provider=jina` → `POST https://api.jina.ai/v1/embeddings` called correctly.
- [ ] Doubao LLM adapter: `llm_provider=doubao` → API calls routed to ByteDance endpoint.
