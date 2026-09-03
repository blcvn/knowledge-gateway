# TASK-GR-006 — LLM Client Adapters (Bifrost + OpenAI)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-006 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/graphiti-knowledge/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §2, §9 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-001 |
| **Estimated** | 4h |

---

## Context

Xây dựng `LLMClient` interface và các adapter (Bifrost primary, OpenAI fallback). Bao gồm: Redis LLM response caching, retry với exponential backoff, `EmbedderClient` interface, `TokenTracker` cho per-prompt usage tracking.

---

## Goal

- `LLMClient` interface với `GenerateResponse` (structured JSON output)
- `BifrostLLMClient` adapter — routing qua Bifrost LLM gateway
- `OpenAILLMClient` adapter — direct OpenAI API fallback
- `EmbedderClient` interface + OpenAI embedder
- `RedisLLMCache` — cache LLM responses by message hash
- `TokenTracker` — per-prompt token accumulation (thread-safe)
- `CrossEncoderClient` — reranker interface for neural reranking

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-knowledge/internal/adapter/client/llm/client.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/client/llm/bifrost.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/client/llm/openai.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/client/embedder/client.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/client/reranker/client.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/cache/redis_llm_cache.go` |
| CREATE | `services/graphiti-knowledge/internal/infra/telemetry/token_tracker.go` |

---

## Implementation

### File 1: `services/graphiti-knowledge/internal/adapter/client/llm/client.go`

```go
package llm

import "context"

// ModelSize hints at which model tier to use (affects cost/quality tradeoff)
type ModelSize int

const (
    // ModelSizeMedium — best quality (gpt-4o, claude-3-5-sonnet, gemini-2.0-flash)
    ModelSizeMedium ModelSize = iota
    // ModelSizeSmall — cheaper/faster (gpt-4o-mini, haiku, flash-lite)
    // Used for resolution decisions where cost matters more than quality
    ModelSizeSmall
)

// Message represents a single chat turn
type Message struct {
    Role    string `json:"role"`    // "system" | "user" | "assistant"
    Content string `json:"content"`
}

// GenerateOpts configures a single LLM call
type GenerateOpts struct {
    ResponseSchema interface{} // JSON schema for structured output (nil = free text)
    PromptName     string      // for token tracking (e.g. "extract_nodes")
    ModelSize      ModelSize
    MaxTokens      int         // 0 = use default (4096)
    Temperature    float64     // 0.0 = deterministic
}

// TokenUsage captures token consumption for a single LLM call
type TokenUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

func (u *TokenUsage) Add(other TokenUsage) {
    u.PromptTokens     += other.PromptTokens
    u.CompletionTokens += other.CompletionTokens
    u.TotalTokens      += other.TotalTokens
}

// LLMResponse contains the parsed LLM output
type LLMResponse struct {
    Content    []byte     // raw JSON bytes (parsed into target struct by caller)
    TokenUsage TokenUsage
    Cached     bool       // true if served from Redis cache
    Provider   string     // "bifrost" | "openai" | "anthropic"
    Model      string     // actual model used
}

// LLMClient — unified interface for all LLM providers.
// All implementations must be safe for concurrent use.
// Content is always returned as raw JSON bytes (structured output).
type LLMClient interface {
    GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error)
    Provider() string
}
```

### File 2: `services/graphiti-knowledge/internal/adapter/client/llm/bifrost.go`

```go
package llm

import (
    "context"
    "crypto/md5"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"

    bifrost "github.com/maximhq/bifrost/transports/bifrost-http/client"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/cache"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/infra/telemetry"
)

type RetryConfig struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    CacheTTL     time.Duration
}

var DefaultRetryConfig = RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 500 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    CacheTTL:     1 * time.Hour,
}

type BifrostLLMClient struct {
    client       *bifrost.Client
    mediumModel  string  // e.g., "openai/gpt-4o"
    smallModel   string  // e.g., "openai/gpt-4o-mini"
    llmCache     cache.LLMCache
    tokenTracker *telemetry.TokenTracker
    retry        RetryConfig
}

type BifrostConfig struct {
    BaseURL     string
    APIKey      string
    MediumModel string
    SmallModel  string
}

func NewBifrostLLMClient(cfg BifrostConfig, llmCache cache.LLMCache, tracker *telemetry.TokenTracker) *BifrostLLMClient {
    client := bifrost.NewClient(bifrost.Config{
        BaseURL: cfg.BaseURL,
        APIKey:  cfg.APIKey,
    })
    return &BifrostLLMClient{
        client:       client,
        mediumModel:  cfg.MediumModel,
        smallModel:   cfg.SmallModel,
        llmCache:     llmCache,
        tokenTracker: tracker,
        retry:        DefaultRetryConfig,
    }
}

func (c *BifrostLLMClient) Provider() string { return "bifrost" }

func (c *BifrostLLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
    // 1. Check cache
    cacheKey := computeCacheKey(messages, opts)
    if cached, ok := c.llmCache.Get(ctx, cacheKey); ok {
        return &LLMResponse{Content: cached.Content, Cached: true, Provider: "bifrost"}, nil
    }

    // 2. Select model
    model := c.mediumModel
    if opts.ModelSize == ModelSizeSmall { model = c.smallModel }

    // 3. Build request
    maxTokens := opts.MaxTokens
    if maxTokens == 0 { maxTokens = 4096 }

    req := bifrost.ChatRequest{
        Model:       model,
        Messages:    mapMessages(messages),
        MaxTokens:   maxTokens,
        Temperature: opts.Temperature,
    }
    if opts.ResponseSchema != nil {
        req.ResponseFormat = bifrost.ResponseFormat{
            Type:       "json_schema",
            JSONSchema: opts.ResponseSchema,
        }
    }

    // 4. Execute with retry + exponential backoff
    var resp *bifrost.ChatResponse
    var lastErr error
    for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
        resp, lastErr = c.client.Chat(ctx, req)
        if lastErr == nil { break }
        if !isRetryable(lastErr) { break }
        delay := c.retry.InitialDelay * time.Duration(1<<(attempt-1))
        if delay > c.retry.MaxDelay { delay = c.retry.MaxDelay }
        select {
        case <-ctx.Done(): return nil, ctx.Err()
        case <-time.After(delay):
        }
    }
    if lastErr != nil {
        return nil, fmt.Errorf("bifrost after %d attempts: %w", c.retry.MaxAttempts, lastErr)
    }

    result := &LLMResponse{
        Content:  []byte(resp.Choices[0].Message.Content),
        Provider: "bifrost",
        Model:    model,
        TokenUsage: TokenUsage{
            PromptTokens:     resp.Usage.PromptTokens,
            CompletionTokens: resp.Usage.CompletionTokens,
            TotalTokens:      resp.Usage.TotalTokens,
        },
    }

    // 5. Track usage + cache
    if opts.PromptName != "" {
        c.tokenTracker.Track(opts.PromptName, result.TokenUsage)
    }
    c.llmCache.Set(ctx, cacheKey, result, c.retry.CacheTTL)

    return result, nil
}

// computeCacheKey generates a stable cache key from messages + prompt config
func computeCacheKey(messages []Message, opts GenerateOpts) string {
    h := md5.New()
    json.NewEncoder(h).Encode(messages)
    fmt.Fprintf(h, "|%s|%d", opts.PromptName, opts.ModelSize)
    return hex.EncodeToString(h.Sum(nil))
}

func isRetryable(err error) bool {
    if err == nil { return false }
    msg := err.Error()
    return contains(msg, "429") || contains(msg, "503") ||
        contains(msg, "timeout") || contains(msg, "connection")
}

func mapMessages(msgs []Message) []bifrost.Message {
    result := make([]bifrost.Message, len(msgs))
    for i, m := range msgs {
        result[i] = bifrost.Message{Role: m.Role, Content: m.Content}
    }
    return result
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr { return true }
    }
    return false
}
```

### File 3: `services/graphiti-knowledge/internal/adapter/client/llm/openai.go`

```go
package llm

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    openai "github.com/sashabaranov/go-openai"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/cache"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/infra/telemetry"
)

type OpenAIConfig struct {
    APIKey      string
    MediumModel string // default: gpt-4o
    SmallModel  string // default: gpt-4o-mini
    BaseURL     string // optional custom endpoint
}

type OpenAILLMClient struct {
    client       *openai.Client
    mediumModel  string
    smallModel   string
    llmCache     cache.LLMCache
    tokenTracker *telemetry.TokenTracker
    retry        RetryConfig
}

func NewOpenAILLMClient(cfg OpenAIConfig, llmCache cache.LLMCache, tracker *telemetry.TokenTracker) *OpenAILLMClient {
    config := openai.DefaultConfig(cfg.APIKey)
    if cfg.BaseURL != "" { config.BaseURL = cfg.BaseURL }
    mediumModel := cfg.MediumModel
    if mediumModel == "" { mediumModel = "gpt-4o" }
    smallModel := cfg.SmallModel
    if smallModel == "" { smallModel = "gpt-4o-mini" }

    return &OpenAILLMClient{
        client:       openai.NewClientWithConfig(config),
        mediumModel:  mediumModel,
        smallModel:   smallModel,
        llmCache:     llmCache,
        tokenTracker: tracker,
        retry:        DefaultRetryConfig,
    }
}

func (c *OpenAILLMClient) Provider() string { return "openai" }

func (c *OpenAILLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
    // Check cache
    cacheKey := computeCacheKey(messages, opts)
    if cached, ok := c.llmCache.Get(ctx, cacheKey); ok {
        return &LLMResponse{Content: cached.Content, Cached: true, Provider: "openai"}, nil
    }

    model := c.mediumModel
    if opts.ModelSize == ModelSizeSmall { model = c.smallModel }

    maxTokens := opts.MaxTokens
    if maxTokens == 0 { maxTokens = 4096 }

    openaiMsgs := make([]openai.ChatCompletionMessage, len(messages))
    for i, m := range messages {
        openaiMsgs[i] = openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
    }

    req := openai.ChatCompletionRequest{
        Model:       model,
        Messages:    openaiMsgs,
        MaxTokens:   maxTokens,
        Temperature: float32(opts.Temperature),
    }

    // Use JSON schema response format if schema provided
    if opts.ResponseSchema != nil {
        schemaBytes, _ := json.Marshal(opts.ResponseSchema)
        req.ResponseFormat = &openai.ChatCompletionResponseFormat{
            Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
            JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
                Name:   opts.PromptName,
                Schema: json.RawMessage(schemaBytes),
                Strict: true,
            },
        }
    }

    var resp openai.ChatCompletionResponse
    var lastErr error
    for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
        resp, lastErr = c.client.CreateChatCompletion(ctx, req)
        if lastErr == nil { break }
        if !isRetryable(lastErr) { break }
        delay := c.retry.InitialDelay * time.Duration(1<<(attempt-1))
        if delay > c.retry.MaxDelay { delay = c.retry.MaxDelay }
        select {
        case <-ctx.Done(): return nil, ctx.Err()
        case <-time.After(delay):
        }
    }
    if lastErr != nil { return nil, fmt.Errorf("openai: %w", lastErr) }

    if len(resp.Choices) == 0 { return nil, fmt.Errorf("openai: no choices returned") }

    result := &LLMResponse{
        Content:  []byte(resp.Choices[0].Message.Content),
        Provider: "openai",
        Model:    model,
        TokenUsage: TokenUsage{
            PromptTokens:     resp.Usage.PromptTokens,
            CompletionTokens: resp.Usage.CompletionTokens,
            TotalTokens:      resp.Usage.TotalTokens,
        },
    }

    if opts.PromptName != "" { c.tokenTracker.Track(opts.PromptName, result.TokenUsage) }
    c.llmCache.Set(ctx, cacheKey, result, c.retry.CacheTTL)
    return result, nil
}
```

### File 4: `services/graphiti-knowledge/internal/adapter/client/embedder/client.go`

```go
package embedder

import (
    "context"
    "fmt"

    openai "github.com/sashabaranov/go-openai"
)

// EmbedderClient — generates vector embeddings for text
type EmbedderClient interface {
    Create(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
}

// OpenAIEmbedder uses OpenAI's text-embedding-3-small (1536 dims)
type OpenAIEmbedder struct {
    client *openai.Client
    model  string
    dims   int
}

func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
    if model == "" { model = "text-embedding-3-small" }
    return &OpenAIEmbedder{
        client: openai.NewClient(apiKey),
        model:  model,
        dims:   1536,
    }
}

func (e *OpenAIEmbedder) Dimensions() int { return e.dims }

func (e *OpenAIEmbedder) Create(ctx context.Context, text string) ([]float32, error) {
    if text == "" { return make([]float32, e.dims), nil }

    // Truncate to 8192 tokens (API limit)
    if len(text) > 32000 { text = text[:32000] }

    resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Input: []string{text},
        Model: openai.EmbeddingModel(e.model),
    })
    if err != nil { return nil, fmt.Errorf("create embedding: %w", err) }
    if len(resp.Data) == 0 { return nil, fmt.Errorf("no embedding returned") }
    return resp.Data[0].Embedding, nil
}
```

### File 5: `services/graphiti-knowledge/internal/adapter/cache/redis_llm_cache.go`

```go
package cache

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

// LLMCache — interface for caching LLM responses
type LLMCache interface {
    Get(ctx context.Context, key string) (*CachedResponse, bool)
    Set(ctx context.Context, key string, resp *LLMResponse, ttl time.Duration)
}

type CachedResponse struct {
    Content []byte `json:"content"`
}

// LLMResponse (embedded reference from llm package via interface)
type LLMResponse interface {
    GetContent() []byte
}

// RedisLLMCache caches LLM responses by message hash (1h TTL default)
type RedisLLMCache struct {
    client redis.UniversalClient
    prefix string
}

func NewRedisLLMCache(client redis.UniversalClient) *RedisLLMCache {
    return &RedisLLMCache{client: client, prefix: "graphiti:llm:"}
}

func (c *RedisLLMCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
    val, err := c.client.Get(ctx, c.prefix+key).Bytes()
    if err != nil { return nil, false }
    var cached CachedResponse
    if err := json.Unmarshal(val, &cached); err != nil { return nil, false }
    return &cached, true
}

func (c *RedisLLMCache) Set(ctx context.Context, key string, content []byte, ttl time.Duration) {
    data, err := json.Marshal(CachedResponse{Content: content})
    if err != nil { return }
    c.client.Set(ctx, c.prefix+key, data, ttl)
}
```

### File 6: `services/graphiti-knowledge/internal/infra/telemetry/token_tracker.go`

```go
package telemetry

import (
    "sync"
)

// PromptUsageAgg accumulates token usage per prompt type
type PromptUsageAgg struct {
    PromptTokens     int64
    CompletionTokens int64
    TotalTokens      int64
    CallCount        int64
}

// TokenUsage for interface compatibility
type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// TokenTracker tracks LLM token usage per prompt type. Thread-safe.
type TokenTracker struct {
    mu    sync.RWMutex
    usage map[string]*PromptUsageAgg
}

func NewTokenTracker() *TokenTracker {
    return &TokenTracker{usage: make(map[string]*PromptUsageAgg)}
}

// Track adds token usage for a given prompt name
func (t *TokenTracker) Track(promptName string, usage TokenUsage) {
    t.mu.Lock()
    defer t.mu.Unlock()

    agg, ok := t.usage[promptName]
    if !ok {
        agg = &PromptUsageAgg{}
        t.usage[promptName] = agg
    }
    agg.PromptTokens     += int64(usage.PromptTokens)
    agg.CompletionTokens += int64(usage.CompletionTokens)
    agg.TotalTokens      += int64(usage.TotalTokens)
    agg.CallCount++
}

// GetAll returns a snapshot of all prompt usage (copy, safe to read without lock)
func (t *TokenTracker) GetAll() map[string]PromptUsageAgg {
    t.mu.RLock()
    defer t.mu.RUnlock()
    result := make(map[string]PromptUsageAgg, len(t.usage))
    for k, v := range t.usage { result[k] = *v }
    return result
}

// Reset clears all accumulated token usage
func (t *TokenTracker) Reset() {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.usage = make(map[string]*PromptUsageAgg)
}
```

---

## Dependencies to Add

```bash
cd services/graphiti-knowledge
go get github.com/sashabaranov/go-openai@latest
go get github.com/redis/go-redis/v9@latest
# Bifrost client (check internal registry for exact package path)
go get github.com/maximhq/bifrost/transports/bifrost-http/client@latest
```

---

## Verification

```bash
cd services/graphiti-knowledge
go build ./internal/adapter/client/...
go build ./internal/adapter/cache/...
go build ./internal/infra/telemetry/...
go vet ./...
```

**Expected:** No compilation errors.
