# TASK-MB-005 — `pkg/adapters/llm/`, `pkg/adapters/embedder/`, `pkg/resilience/`

**Wave:** 2 (LLM Processing Foundation)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-001 (pkg/config đã có)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-MB-007 §4, §5, §6, §9](../solutions/SOL-MB-007-Shared-Infrastructure.md)
**Trạng thái:** ✅ Implemented

---

## Mục tiêu

Tạo 3 shared packages cho Wave 2 LLM processing:
1. **`pkg/adapters/llm/`** — LLMClient interface + Bifrost, OpenAI, Doubao adapters
2. **`pkg/adapters/embedder/`** — EmbedderClient interface + Jina, Ollama, Disabled adapters
3. **`pkg/resilience/`** — Circuit breaker, exponential backoff retry, bulkhead semaphore

---

## 1. `pkg/adapters/llm/` — LLM Client

### File: `pkg/adapters/llm/client.go`

```go
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

func WithJSONMode() Option          { return func(c *RequestConfig) { c.JSONMode = true } }
func WithModel(m string) Option     { return func(c *RequestConfig) { c.Model = m } }
func WithMaxTokens(n int) Option    { return func(c *RequestConfig) { c.MaxTokens = n } }
func WithTemperature(t float64) Option { return func(c *RequestConfig) { c.Temperature = t } }

var ErrLLMRequestFailed  = errors.New("llm: request failed")
var ErrLLMInvalidJSON    = errors.New("llm: response is not valid JSON")
var ErrLLMContextTimeout = errors.New("llm: context deadline exceeded")
```

### File: `pkg/adapters/llm/bifrost.go`

```go
// Bifrost: production LLM gateway với auto-failover
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
        defaultConf: RequestConfig{Model: model, MaxTokens: 1024, Temperature: 0.7},
    }
}

func (c *BifrostClient) Complete(ctx context.Context, prompt string, opts ...Option) (string, error) {
    raw, err := c.CompleteJSON(ctx, prompt, opts...)
    if err != nil { return "", err }
    // Extract text from response.choices[0].message.content
    return extractTextContent(raw)
}

func (c *BifrostClient) CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error) {
    cfg := c.defaultConf
    for _, opt := range opts { opt(&cfg) }

    reqBody := map[string]any{
        "model":      cfg.Model,
        "messages":   []map[string]string{{"role": "user", "content": prompt}},
        "max_tokens": cfg.MaxTokens,
    }
    if cfg.JSONMode {
        reqBody["response_format"] = map[string]string{"type": "json_object"}
    }
    if cfg.Temperature != 0 {
        reqBody["temperature"] = cfg.Temperature
    }

    data, _ := json.Marshal(reqBody)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(data))
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil { return nil, fmt.Errorf("%w: %v", ErrLLMRequestFailed, err) }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("%w: status=%d body=%s", ErrLLMRequestFailed, resp.StatusCode, body)
    }

    return extractJSONContent(body)
}

func extractJSONContent(body []byte) (json.RawMessage, error) {
    var resp struct {
        Choices []struct {
            Message struct { Content string `json:"content"` } `json:"message"`
        } `json:"choices"`
    }
    json.Unmarshal(body, &resp)
    if len(resp.Choices) == 0 { return nil, ErrLLMInvalidJSON }
    return json.RawMessage(resp.Choices[0].Message.Content), nil
}
```

### File: `pkg/adapters/llm/openai.go`

```go
// Dùng go-openai library; compatible với vLLM, Ollama (với /v1 compat), LM Studio
type OpenAICompatClient struct {
    client      *openai.Client
    model       string
    defaultConf RequestConfig
}

func NewOpenAICompatClient(baseURL, apiKey, model string) *OpenAICompatClient {
    config := openai.DefaultConfig(apiKey)
    if baseURL != "" {
        config.BaseURL = baseURL
    }
    return &OpenAICompatClient{
        client: openai.NewClientWithConfig(config),
        model:  model,
    }
}

// Ollama usage:
// NewOpenAICompatClient("http://localhost:11434/v1", "ollama", "llama3.2")
```

### File: `pkg/adapters/llm/doubao.go`

```go
// ByteDance Doubao — provides LLM response caching (~60% cost reduction on repeated prompts)
// API endpoint: https://ark.cn-beijing.volces.com/api/v3/chat/completions
type DoubaoClient struct {
    httpClient *http.Client
    apiKey     string
    model      string
    endpoint   string  // default: "https://ark.cn-beijing.volces.com/api/v3"
}

func NewDoubaoClient(apiKey, model string) *DoubaoClient
// Same API format as OpenAI (compatible)
// Auth: Authorization: Bearer {apiKey}
```

---

## 2. `pkg/adapters/embedder/` — Embedding Client

### File: `pkg/adapters/embedder/client.go`

```go
package embedder

type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
    Dimension() int
    IsEnabled() bool
}

var ErrEmbeddingDisabled = errors.New("embedder: embedding is disabled")
var ErrEmbeddingFailed   = errors.New("embedder: embedding request failed")
```

### File: `pkg/adapters/embedder/jina.go`

```go
// Jina AI embeddings: model "jina-embeddings-v3", dimension=1024
// API: POST https://api.jina.ai/v1/embeddings
type JinaEmbedder struct {
    httpClient *http.Client
    apiKey     string
    model      string
    dimension  int
}

func NewJinaEmbedder(apiKey, model string, dimension int) *JinaEmbedder {
    return &JinaEmbedder{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        apiKey:     apiKey,
        model:      model,
        dimension:  dimension,
    }
}

func (j *JinaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    reqBody := map[string]any{"model": j.model, "input": texts}
    data, _ := json.Marshal(reqBody)

    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.jina.ai/v1/embeddings", bytes.NewReader(data))
    req.Header.Set("Authorization", "Bearer "+j.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := j.httpClient.Do(req)
    if err != nil { return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err) }
    defer resp.Body.Close()

    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    embeddings := make([][]float32, len(result.Data))
    for i, d := range result.Data { embeddings[i] = d.Embedding }
    return embeddings, nil
}

func (j *JinaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
    embs, err := j.Embed(ctx, []string{query})
    if err != nil || len(embs) == 0 { return nil, err }
    return embs[0], nil
}

func (j *JinaEmbedder) Dimension() int  { return j.dimension }
func (j *JinaEmbedder) IsEnabled() bool { return true }
```

### File: `pkg/adapters/embedder/ollama.go`

```go
// Ollama self-hosted embedding (e.g., nomic-embed-text, dimension=768)
type OllamaEmbedder struct {
    httpClient *http.Client
    baseURL    string
    model      string
    dimension  int
}

func NewOllamaEmbedder(baseURL, model string, dimension int) *OllamaEmbedder

func (o *OllamaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
    reqBody := map[string]string{"model": o.model, "prompt": query}
    // POST {baseURL}/api/embeddings
    // Response: {"embedding": [0.1, 0.2, ...]}
}

func (o *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // Call EmbedQuery for each text (Ollama no batch API)
    results := make([][]float32, len(texts))
    for i, t := range texts {
        emb, err := o.EmbedQuery(ctx, t)
        if err != nil { return nil, err }
        results[i] = emb
    }
    return results, nil
}
```

### File: `pkg/adapters/embedder/disabled.go`

```go
type DisabledEmbedder struct{}

func (d *DisabledEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
    return nil, ErrEmbeddingDisabled
}
func (d *DisabledEmbedder) EmbedQuery(_ context.Context, query string) ([]float32, error) {
    return nil, ErrEmbeddingDisabled
}
func (d *DisabledEmbedder) Dimension() int  { return 0 }
func (d *DisabledEmbedder) IsEnabled() bool { return false }
```

---

## 3. `pkg/resilience/` — Circuit Breaker, Retry, Bulkhead

### File: `pkg/resilience/circuit_breaker.go`

```go
package resilience

import "github.com/sony/gobreaker"

var ErrCircuitOpen = errors.New("resilience: circuit breaker is open")

type CircuitBreaker struct {
    cb *gobreaker.CircuitBreaker
}

func NewCircuitBreaker(name string) *CircuitBreaker {
    settings := gobreaker.Settings{
        Name:        name,
        MaxRequests: 3,
        Interval:    1 * time.Minute,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            if counts.Requests < 10 { return false }
            return float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
        },
    }
    return &CircuitBreaker{cb: gobreaker.NewCircuitBreaker(settings)}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    _, err := cb.cb.Execute(func() (interface{}, error) { return nil, fn() })
    if errors.Is(err, gobreaker.ErrOpenState) {
        return ErrCircuitOpen
    }
    return err
}

func (cb *CircuitBreaker) State() gobreaker.State { return cb.cb.State() }
```

### File: `pkg/resilience/retry.go`

```go
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
                delay = min(time.Duration(float64(delay)*cfg.Multiplier), cfg.MaxDelay)
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        lastErr = fn()
        if lastErr == nil { return nil }
        if cfg.IsRetryable != nil && !cfg.IsRetryable(lastErr) { return lastErr }
    }
    return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
```

### File: `pkg/resilience/bulkhead.go`

```go
type Bulkhead struct {
    sem chan struct{}
    max int
}

func NewBulkhead(maxConcurrent int) *Bulkhead {
    return &Bulkhead{sem: make(chan struct{}, maxConcurrent), max: maxConcurrent}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
    select {
    case b.sem <- struct{}{}:
        defer func() { <-b.sem }()
        return fn()
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (b *Bulkhead) Active() int { return len(b.sem) }
func (b *Bulkhead) Max() int    { return b.max }
```

### File: `pkg/resilience/resilient_llm.go`

```go
// Composition pattern: LLM + Circuit Breaker + Retry + Bulkhead + OTel
type ResilientLLMClient struct {
    inner    llm.LLMClient
    cb       *CircuitBreaker
    bulkhead *Bulkhead
    retry    RetryConfig
    tracer   trace.Tracer
}

func NewResilientLLMClient(inner llm.LLMClient, cb *CircuitBreaker, bh *Bulkhead) *ResilientLLMClient

func (c *ResilientLLMClient) CompleteJSON(ctx context.Context, prompt string, opts ...llm.Option) (json.RawMessage, error) {
    ctx, span := c.tracer.Start(ctx, "llm_call.complete_json")
    defer span.End()

    var result json.RawMessage
    err := c.bulkhead.Execute(ctx, func() error {
        return c.cb.Execute(func() error {
            return Retry(ctx, c.retry, func() error {
                var err error
                result, err = c.inner.CompleteJSON(ctx, prompt, opts...)
                return err
            })
        })
    })
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    return result, nil
}
```

---

## Unit Tests

```
TestBifrostClient_CompleteJSON_ValidResponse → mock HTTP → JSON extracted
TestBifrostClient_CompleteJSON_HTTP400       → 400 response → ErrLLMRequestFailed
TestBifrostClient_CompleteJSON_JSONMode      → response_format set in request body
TestBifrostClient_Complete_ExtractsText      → JSON choices[0].message.content → string
TestOpenAICompatClient_CreatesWithBaseURL    → Ollama URL → client configured
TestJinaEmbedder_Embed_SingleText           → 1 text → 1 embedding
TestJinaEmbedder_Embed_Batch                → 3 texts → 3 embeddings
TestJinaEmbedder_Embed_HTTP_Error           → HTTP 500 → ErrEmbeddingFailed
TestJinaEmbedder_IsEnabled                  → true
TestOllamaEmbedder_EmbedQuery               → single prompt → embedding vector
TestDisabledEmbedder_Embed                  → ErrEmbeddingDisabled
TestDisabledEmbedder_IsEnabled              → false
TestCircuitBreaker_AllowsInitially          → first call → executes fn
TestCircuitBreaker_TripsAfterFailures       → 10 failures (>50%) → ErrCircuitOpen
TestCircuitBreaker_HalfOpenAfterTimeout     → tripped → wait timeout → allows test call
TestRetry_SuccessFirstAttempt               → fn returns nil → called once
TestRetry_RetriesOnRetryable                → 2 retryable errors, then success → 3 calls
TestRetry_NoRetryOnNonRetryable             → codes.InvalidArgument → called once
TestRetry_ContextCancel                     → ctx cancelled during wait → returns ctx.Err()
TestRetry_ExhaustsAttempts                  → 3 failures → "after 3 attempts" error
TestRetry_ExponentialBackoff                → delays 100ms, 200ms, 400ms
TestBulkhead_AllowsBelowMax                 → max=5, 3 concurrent → all proceed
TestBulkhead_BlocksAtMax                    → max=2, 3 goroutines → 1 waits
TestBulkhead_ContextCancel                  → blocked goroutine → ctx cancel → unblocks
TestBulkhead_Active                         → 3 active → Active()==3
TestResilientLLM_ComposesCorrectly          → fn → bulkhead.Execute → cb.Execute → retry
TestResilientLLM_CircuitOpenReturnsError    → cb open → ErrCircuitOpen propagated
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go get github.com/sony/gobreaker@latest
go get github.com/sashabaranov/go-openai@latest  # for OpenAI compat
go mod tidy

go build ./pkg/adapters/...
go build ./pkg/resilience/...
go test ./pkg/adapters/... ./pkg/resilience/... -v -count=1 -race
```

---

## Ghi chú triển khai

- Bifrost là primary cho production; OpenAI compat cho dev/local với Ollama
- `ResilientLLMClient` là decorator — wrap bất kỳ LLMClient nào (Bifrost, OpenAI, Doubao)
- Bulkhead max=10 cho engine (10 concurrent LLM calls per instance)
- `extractJSONContent`: response `choices[0].message.content` có thể là plain string HOẶC đã là JSON object — try parse as JSON first
- `pkg/adapters/llm/` và `pkg/adapters/embedder/` là PURE packages — không import business domain
