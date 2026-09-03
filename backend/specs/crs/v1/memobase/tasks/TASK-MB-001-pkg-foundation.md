# TASK-MB-001 — `pkg/tokenizer/` & `pkg/config/` Foundation Packages

**Wave:** 1 (Data Layer — xây dựng trước mọi service)  
**Ưu tiên:** Critical  
**Phụ thuộc:** Không có  
**Ước tính:** 2 giờ  
**Solution tham chiếu:** [SOL-MB-007 §3, §7](../solutions/SOL-MB-007-Shared-Infrastructure.md)
**Trạng thái:** ✅ Implemented  
**Ghi chú:** shared/pkg/tokenizer + shared/pkg/config exists (3+3 .go)

---

## Mục tiêu

Tạo 2 foundation packages được dùng ngay từ Wave 1:
1. **`pkg/tokenizer/`** — tiktoken-go wrapper để đếm tokens cho blob ingestion
2. **`pkg/config/`** — Viper config loader với `MEMOBASE_*` ENV override

---

## 1. `pkg/tokenizer/` — tiktoken-go Wrapper

### File: `pkg/tokenizer/tokenizer.go`

```go
package tokenizer

type ChatMessage struct {
    Role    string
    Content string
}

type Tokenizer interface {
    Count(text string) int
    CountMessages(messages []ChatMessage) int
    TruncateToTokens(text string, maxTokens int) string
    CountBlob(blobData any, blobType string) int
}
```

### File: `pkg/tokenizer/tiktoken.go`

```go
package tokenizer

import "github.com/pkoukk/tiktoken-go"

type TiktokenTokenizer struct {
    enc *tiktoken.Tiktoken
}

func New(model string) (*TiktokenTokenizer, error)
// tiktoken.EncodingForModel(model) — e.g., "gpt-4o" → cl100k_base

func (t *TiktokenTokenizer) Count(text string) int
// len(t.enc.Encode(text, nil, nil))

func (t *TiktokenTokenizer) CountMessages(messages []ChatMessage) int
// total := 3  // base overhead
// for each msg: total += 4 (overhead) + Count(role) + Count(content)

func (t *TiktokenTokenizer) TruncateToTokens(text string, maxTokens int) string
// tokens := Encode(text)
// if len(tokens) <= maxTokens: return text
// return string(Decode(tokens[:maxTokens]))

func (t *TiktokenTokenizer) CountBlob(blobData any, blobType string) int
// switch blobType:
//   "chat":    blobData.(ChatBlobData) → CountMessages(data.Messages)
//   "doc","summary": blobData.(string) → Count(text)
```

### File: `pkg/tokenizer/tiktoken_test.go`

Tests:
```
TestTiktokenTokenizer_New_ValidModel        → "gpt-4o" → no error
TestTiktokenTokenizer_New_InvalidModel      → "invalid-model" → error
TestTiktokenTokenizer_Count_Empty           → "" → 0
TestTiktokenTokenizer_Count_ShortText       → "Hello" → non-zero count
TestTiktokenTokenizer_CountMessages_Overhead → [] messages → 3 (base overhead)
TestTiktokenTokenizer_CountMessages_Single  → 1 message → 4+role+content
TestTiktokenTokenizer_TruncateToTokens_Under → short text, large maxTokens → unchanged
TestTiktokenTokenizer_TruncateToTokens_Over  → long text, small maxTokens → truncated
TestTiktokenTokenizer_TruncateToTokens_Exact → result token count == maxTokens
TestTiktokenTokenizer_CountBlob_Chat        → ChatBlobData → sum of messages
TestTiktokenTokenizer_CountBlob_Doc         → string → Count result
TestTiktokenTokenizer_CountBlob_Unknown     → unknown type → 0
```

---

## 2. `pkg/config/` — Viper Config Loader

### File: `pkg/config/loader.go`

```go
package config

import "github.com/spf13/viper"

func Load[T any](configPath string) (*T, error) {
    v := viper.New()
    v.SetConfigFile(configPath)
    v.SetEnvPrefix("MEMOBASE")   // MEMOBASE_* vars override yaml
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

    if err := v.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config %s: %w", configPath, err)
    }

    var cfg T
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &cfg, nil
}

// Load với default values
func LoadWithDefaults[T any](configPath string, defaults map[string]any) (*T, error) {
    v := viper.New()
    for k, val := range defaults {
        v.SetDefault(k, val)
    }
    // ... same as above
}
```

### File: `pkg/config/validator.go`

```go
// StartupCheck validates required config fields
// Returns error with human-readable messages for missing/invalid fields

type ValidationRule struct {
    Field   string
    Check   func(v *viper.Viper) bool
    Message string
}

func Validate(v *viper.Viper, rules []ValidationRule) error {
    var errs []string
    for _, rule := range rules {
        if !rule.Check(v) {
            errs = append(errs, rule.Message)
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("config validation failed:\n%s", strings.Join(errs, "\n"))
    }
    return nil
}

// Common rules
func RequireNonEmpty(field string) ValidationRule
func RequirePositiveInt(field string) ValidationRule
func RequireValidURL(field string) ValidationRule
```

### File: `pkg/config/config_test.go`

```
TestLoad_ValidYAML              → parse valid YAML → struct populated
TestLoad_MissingFile            → non-existent path → error
TestLoad_ENVOverride            → MEMOBASE_LANGUAGE=zh → cfg.Language="zh"
TestLoad_ENVOverride_Nested     → MEMOBASE_LLM_API_KEY=xxx → cfg.LLM.APIKey="xxx"
TestLoad_ENVOverride_Integer    → MEMOBASE_MAX_CHAT_BLOB_BUFFER_TOKEN_SIZE=2048 → int field
TestValidate_AllPass            → all rules pass → nil error
TestValidate_OneFailure         → 1 rule fails → error contains field name
TestValidate_MultipleFailures   → 2 rules fail → error lists both
TestRequireNonEmpty_EmptyString → empty → fails validation
TestRequirePositiveInt_Zero     → 0 → fails validation
```

---

## Cấu trúc thư mục kết quả

```
pkg/
├── tokenizer/
│   ├── tokenizer.go        # Interface
│   ├── tiktoken.go         # Implementation
│   └── tiktoken_test.go
└── config/
    ├── loader.go
    ├── validator.go
    └── config_test.go
```

---

## Go Dependencies cần thêm vào `go.work`

```
github.com/pkoukk/tiktoken-go v0.1.7
github.com/spf13/viper v1.19.0
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go get github.com/pkoukk/tiktoken-go@v0.1.7
go get github.com/spf13/viper@v1.19.0
go mod tidy

go build ./pkg/tokenizer/...
go build ./pkg/config/...
go test ./pkg/tokenizer/... ./pkg/config/... -v -count=1
```

---

## Ghi chú triển khai

- tiktoken-go sẽ download encoding data file khi `New()` lần đầu — cache vào `~/.cache/tiktoken/`
- Để bundle encoding vào binary: `TIKTOKEN_CACHE_DIR` hoặc dùng embed
- Generics `Load[T any]` yêu cầu Go 1.21+
- `mapstructure` tags trên config struct cần khớp với yaml keys và ENV var names
