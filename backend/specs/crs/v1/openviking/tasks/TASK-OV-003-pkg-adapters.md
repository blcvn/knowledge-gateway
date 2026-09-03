# TASK-OV-003 — `pkg/adapters/` Infrastructure Interfaces

**Wave:** 1 (Foundation)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-001 (pkg/viking)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-007 §5](../solutions/SOL-OV-007-Shared-Infrastructure.md)

**Trạng thái:** ✅ Implemented  
**Ghi chú:** shared/pkg/adapters covers LLM + storage adapters  
---

## Mục tiêu

Tạo package `pkg/adapters/` gồm 5 infrastructure interfaces và các implementation cần thiết. Tất cả interfaces đều có **no-op / disabled implementation** để dùng trong testing mà không cần external services.

---

## Các Sub-packages cần tạo

### 1. `pkg/adapters/vectordb/` — Vector Store

**File: `interface.go`**
```go
package vectordb

type CollectionSchema struct {
    Name          string
    DenseDim      int
    SparseEnabled bool
}

type ScoredContext struct {
    URI          string
    ParentURI    string
    ContextType  viking.ContextType
    Level        int
    Abstract     string
    Score        float64
    ActiveCount  int64
    IsDirectory  bool
}

type ContextVector struct {
    URI            string
    ParentURI      string
    ContextType    viking.ContextType
    Level          int
    OwnerAccountID string
    OwnerUserID    string
    Abstract       string
    ActiveCount    int64
    DenseVector    []float32
    SparseVector   map[string]float32
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type VectorStore interface {
    CreateCollection(ctx context.Context, schema CollectionSchema) error
    CollectionExists(ctx context.Context, name string) (bool, error)
    DropCollection(ctx context.Context, name string) error
    SearchGlobalRoots(ctx context.Context, dense []float32, sparse map[string]float32, accountID string, topK int) ([]ScoredContext, error)
    SearchChildren(ctx context.Context, parentURI string, dense []float32, sparse map[string]float32, accountID string) ([]ScoredContext, error)
    UpsertContext(ctx context.Context, vec ContextVector) error
    DeleteContext(ctx context.Context, uri string) error
    UpdateActiveCount(ctx context.Context, uri string, delta int64) error
}
```

**File: `qdrant/client.go`** — Qdrant implementation
- Import: `github.com/qdrant/go-client`
- Collection naming: `"openviking_{accountID}"`
- HNSW params: `m=16, ef_construction=128`
- Hybrid search: dense + sparse (nếu SparseEnabled)
- Upsert: payload includes ParentURI, Level, Abstract, ActiveCount để filter

**File: `memory/client.go`** — In-memory implementation (cho testing)
- Thread-safe map-based storage
- Simple cosine similarity cho search (không cần external service)

### 2. `pkg/adapters/embedder/` — Embedding Client

**File: `interface.go`**
```go
package embedder

type EmbedResult struct {
    DenseVector  []float32
    SparseVector map[string]float32
}

type EmbedderClient interface {
    Embed(ctx context.Context, text string, isQuery bool) (*EmbedResult, error)
    EmbedBatch(ctx context.Context, texts []string) ([]*EmbedResult, error)
    Dimension() int
    SupportsSparse() bool
    ProviderName() string
}
```

**File: `bifrost/client.go`** — Bifrost HTTP client
- POST `{bifrost_url}/v1/embeddings`
- Request: `{"provider":"openai","model":"text-embedding-3-small","input":text,"is_query":bool}`
- Response: `{"dense_vector":[...],"sparse_vector":{}}`
- Timeout: 30s
- Retry: 2 lần với exponential backoff khi timeout

**File: `disabled/client.go`** — No-op (trả về zero vector)
```go
type DisabledEmbedder struct{ dim int }
func NewDisabledEmbedder(dim int) *DisabledEmbedder
func (e *DisabledEmbedder) Embed(...) (*EmbedResult, error) {
    return &EmbedResult{DenseVector: make([]float32, e.dim)}, nil
}
```

### 3. `pkg/adapters/vlm/` — Vision-Language Model Client

**File: `interface.go`**
```go
package vlm

type VLMOption func(*VLMConfig)
type VLMConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    JSONSchema  any
}

func WithVLMModel(m string) VLMOption
func WithVLMMaxTokens(n int) VLMOption
func WithVLMTemperature(t float64) VLMOption

type VLMClient interface {
    Generate(ctx context.Context, prompt string, opts ...VLMOption) (string, error)
    GenerateWithImage(ctx context.Context, prompt string, image []byte, opts ...VLMOption) (string, error)
    GenerateStructured(ctx context.Context, prompt string, schema any, opts ...VLMOption) (json.RawMessage, error)
    ProviderName() string
}
```

**File: `bifrost/client.go`** — Routes all VLM via Bifrost
- POST `{bifrost_url}/v1/chat/completions`
- Support structured output via `response_format: {type: "json_object"}`
- Timeout: 120s (VLM calls can be slow)

**File: `disabled/client.go`** — No-op (returns empty string)

### 4. `pkg/adapters/reranker/` — Reranker Client

**File: `interface.go`**
```go
package reranker

type RerankResult struct {
    Index    int
    Score    float64
    Document string
}

type RerankerClient interface {
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
    ProviderName() string
}
```

**File: `jina/client.go`** — Jina reranker
- POST `https://api.jina.ai/v1/rerank`
- Model: `jina-reranker-v2-base-multilingual`
- Request: `{"model":...,"query":...,"documents":[...],"top_n":n}`

**File: `disabled/client.go`** — No-op: returns original order with score=1.0
```go
// Returns documents unchanged (pass-through, no actual reranking)
func (r *DisabledReranker) Rerank(ctx, query, docs, topN) ([]RerankResult, error) {
    results := make([]RerankResult, min(topN, len(docs)))
    for i := range results {
        results[i] = RerankResult{Index: i, Score: 1.0, Document: docs[i]}
    }
    return results, nil
}
```

### 5. `pkg/adapters/kms/` — Key Management Service

**File: `interface.go`**
```go
package kms

type KMSProvider interface {
    GetRootKey(ctx context.Context) ([]byte, error)
    DeriveAccountKey(ctx context.Context, accountID string) ([]byte, error)
    RotateRootKey(ctx context.Context) error
    ProviderType() byte
}

const (
    ProviderTypeLocal byte = 0x01
    ProviderTypeVault byte = 0x02
    ProviderTypeCloud byte = 0x03
)
```

**File: `local/provider.go`** — Key từ local file
```go
type LocalProvider struct {
    keyPath string  // path to root.key file (32 bytes)
}

func NewLocalProvider(keyPath string) (*LocalProvider, error)

func (p *LocalProvider) GetRootKey(ctx context.Context) ([]byte, error)
// Read 32 bytes từ keyPath; nếu không tồn tại → generate ngẫu nhiên + save

func (p *LocalProvider) DeriveAccountKey(ctx context.Context, accountID string) ([]byte, error)
// HKDF(root_key, accountID, "openviking-account-key") → 32 bytes
// golang.org/x/crypto/hkdf

func (p *LocalProvider) RotateRootKey(ctx context.Context) error
// Generate new 32 bytes → write to keyPath

func (p *LocalProvider) ProviderType() byte { return ProviderTypeLocal }
```

**File: `disabled/provider.go`** — No-op (passthrough, không encrypt)
```go
// GetRootKey → returns static 32-byte zeros (NOT for production)
// DeriveAccountKey → HKDF with zeros root key
```

---

## Unit Tests

```
// vectordb/memory
TestMemoryVectorStore_UpsertAndSearch → upsert 10 vectors → search finds top-K
TestMemoryVectorStore_DeleteRemovesFromResults → delete → no longer in search
TestMemoryVectorStore_UpdateActiveCount → increment → higher in next query

// embedder/disabled  
TestDisabledEmbedder_ReturnZeroVector → len(DenseVector) == dim
TestDisabledEmbedder_Batch → returns N results for N inputs

// reranker/disabled
TestDisabledReranker_PreservesOrder → original order maintained
TestDisabledReranker_TopNRespected → topN=3, 10 docs → 3 results

// kms/local
TestLocalProvider_GeneratesKeyIfMissing → no file → generates + saves 32 bytes
TestLocalProvider_LoadsExistingKey → create file → loads correctly
TestLocalProvider_DeriveAccountKey_Deterministic → same inputs → same key
TestLocalProvider_DeriveAccountKey_Different → different accountIDs → different keys
TestLocalProvider_RotateKey → rotate → new key, different from old
```

---

## Cấu trúc thư mục kết quả

```
pkg/adapters/
├── vectordb/
│   ├── interface.go
│   ├── qdrant/
│   │   └── client.go
│   └── memory/
│       ├── client.go
│       └── client_test.go
├── embedder/
│   ├── interface.go
│   ├── bifrost/
│   │   └── client.go
│   └── disabled/
│       ├── client.go
│       └── client_test.go
├── vlm/
│   ├── interface.go
│   ├── bifrost/
│   │   └── client.go
│   └── disabled/
│       └── client.go
├── reranker/
│   ├── interface.go
│   ├── jina/
│   │   └── client.go
│   └── disabled/
│       ├── client.go
│       └── client_test.go
└── kms/
    ├── interface.go
    ├── local/
    │   ├── provider.go
    │   └── provider_test.go
    └── disabled/
        └── provider.go
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./pkg/adapters/...
go test ./pkg/adapters/... -v -count=1
```

---

## Ghi chú triển khai

- Qdrant client cần `github.com/qdrant/go-client` trong `go.mod`
- KMS `local/` cần `golang.org/x/crypto` cho HKDF
- Bifrost clients dùng `net/http` standard library (không dùng SDK)
- Tất cả clients có `timeout` field có thể config
- `disabled/` implementations: dùng cho unit testing tất cả services mà không cần external services
