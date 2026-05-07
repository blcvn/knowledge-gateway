---
skill_id: SKILL-009
version: 1.0.0
status: active
priority: P0
group: AI & Data Engineering
created_at: 2026-04-24
---

# SKILL-009 · Natural Language Processing (NLP)

## Mô tả

Xử lý và phân tích văn bản phi cấu trúc — tokenization, entity recognition, semantic similarity, text classification — để chuẩn bị input cho LLM và post-process output.

## Agents sử dụng

- `requirement-parser-agent`
- `semantic-extractor-agent`

---

## Năng lực cốt lõi

### 1. Text Preprocessing

```go
// Pipeline chuẩn preprocessing cho requirement text
func PreprocessText(raw string) ProcessedText {
    // 1. Language detection
    lang := detectLanguage(raw)  // "vi" | "en" | "mixed"
    
    // 2. Normalization
    text := strings.TrimSpace(raw)
    text = normalizeUnicode(text)      // NFC normalization
    text = normalizeWhitespace(text)   // collapse multiple spaces
    text = fixTypography(text)         // curly quotes → straight quotes
    
    // 3. Sentence segmentation
    sentences := segmentSentences(text, lang)
    
    // 4. Paragraph detection
    paragraphs := detectParagraphs(sentences)
    
    return ProcessedText{
        Language:   lang,
        Sentences:  sentences,
        Paragraphs: paragraphs,
        WordCount:  countWords(text),
        TokenCount: estimateTokens(text),
    }
}
```

### 2. Named Entity Recognition (NER)

#### Domain-specific entities cho Knowledge Gateway

| Entity Type | Examples | Detection Method |
|-------------|----------|-----------------|
| `ACTOR` | "Người dùng", "Admin", "Payment Gateway" | Pattern + LLM |
| `ACTION` | "đăng nhập", "tạo đơn hàng", "xác nhận" | Verb extraction |
| `BUSINESS_OBJECT` | "Đơn hàng", "Sản phẩm", "Khách hàng" | Noun phrase + domain dict |
| `CONSTRAINT` | "phải", "tối đa", "không được", "trong vòng" | Modal verb pattern |
| `QUANTITY` | "100ms", "5 lần", "24 giờ" | Regex pattern |
| `STATUS` | "thành công", "thất bại", "đang xử lý" | Status vocabulary |

```go
// Rule-based NER cho Vietnamese requirement text
var actorPatterns = []string{
    `(?i)(người dùng|khách hàng|admin|quản trị viên|hệ thống|system)`,
    `(?i)(operator|manager|agent|bot|service)`,
}

var constraintIndicators = []string{
    "phải", "cần", "bắt buộc", "tối đa", "tối thiểu",
    "must", "shall", "should", "maximum", "minimum", "within",
}

func ExtractEntities(text string) []Entity {
    entities := []Entity{}
    
    // Rule-based extraction (fast, deterministic)
    entities = append(entities, extractByPatterns(text, actorPatterns, "ACTOR")...)
    entities = append(entities, extractConstraints(text, constraintIndicators)...)
    
    // LLM-based extraction (for complex cases)
    if needsLLM(text) {
        llmEntities := extractWithLLM(text)
        entities = merge(entities, llmEntities)
    }
    
    return deduplicateEntities(entities)
}
```

### 3. Text Classification

```go
// Phân loại paragraph vào categories chuẩn
type ParagraphCategory string

const (
    CategoryOverview    ParagraphCategory = "OVERVIEW"
    CategoryFunctional  ParagraphCategory = "FUNCTIONAL"
    CategoryConstraint  ParagraphCategory = "CONSTRAINT"
    CategoryAPI         ParagraphCategory = "API"
    CategoryUI          ParagraphCategory = "UI"
    CategoryDataModel   ParagraphCategory = "DATA_MODEL"
)

// Rule-based classification (priority 1)
var classificationRules = map[ParagraphCategory][]string{
    CategoryConstraint: {"phải", "tối đa", "must", "shall not", "maximum", "SLA"},
    CategoryAPI:        {"endpoint", "POST", "GET", "request", "response", "HTTP"},
    CategoryUI:         {"màn hình", "screen", "button", "form", "display", "show"},
    CategoryDataModel:  {"field", "column", "table", "schema", "attribute"},
}

func ClassifyParagraph(text string) ParagraphCategory {
    // Rule-based first (deterministic)
    for category, keywords := range classificationRules {
        if matchesKeywords(text, keywords) {
            return category
        }
    }
    
    // LLM fallback for ambiguous cases
    return classifyWithLLM(text)
}
```

### 4. Semantic Similarity & Deduplication

```go
// Embedding-based entity deduplication
// Phát hiện "Order" và "Đơn hàng" là cùng concept

type EmbeddingClient interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

func DeduplicateEntities(entities []Entity, client EmbeddingClient) []Entity {
    embeddings := make([][]float32, len(entities))
    
    // Get embeddings for all entities
    for i, e := range entities {
        emb, _ := client.Embed(context.Background(), e.Name)
        embeddings[i] = emb
    }
    
    // Cluster by cosine similarity threshold
    threshold := float32(0.85)
    groups := clusterBySimilarity(embeddings, threshold)
    
    // Keep canonical name (most frequent or English version)
    return selectCanonical(entities, groups)
}

func cosineSimilarity(a, b []float32) float32 {
    var dot, normA, normB float32
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
```

### 5. Dependency Parsing

```go
// Trích xuất Subject-Verb-Object cho relationship inference
// "Người dùng tạo đơn hàng" → Actor=Người dùng, Action=tạo, Object=đơn hàng

type SVO struct {
    Subject string // Actor
    Verb    string // Action
    Object  string // Business Object
}

// Simplified SVO extraction (works for structured requirement text)
func ExtractSVO(sentence string) []SVO {
    // 1. POS tagging (via dictionary + rules for Vietnamese)
    tokens := tokenize(sentence)
    tagged := posTag(tokens)
    
    // 2. Find verb chunks
    verbChunks := findVerbChunks(tagged)
    
    // 3. For each verb, find subject (left) and object (right)
    svos := []SVO{}
    for _, verb := range verbChunks {
        subject := findSubject(tagged, verb.Position)
        object := findObject(tagged, verb.Position)
        
        if subject != "" && object != "" {
            svos = append(svos, SVO{
                Subject: subject,
                Verb:    verb.Text,
                Object:  object,
            })
        }
    }
    return svos
}
```

### 6. Entity Normalization

```go
// Chuẩn hóa entity names về canonical form
var normalizationRules = map[string]string{
    // Vietnamese → English canonical
    "người dùng": "User",
    "khách hàng": "Customer",
    "đơn hàng":   "Order",
    "sản phẩm":   "Product",
    // Abbreviation expansion
    "KH": "Customer",
    "DH": "Order",
    "SP": "Product",
}

func NormalizeEntityName(name string) string {
    lower := strings.ToLower(strings.TrimSpace(name))
    
    // Check exact match
    if canonical, ok := normalizationRules[lower]; ok {
        return canonical
    }
    
    // Check partial match
    for pattern, canonical := range normalizationRules {
        if strings.Contains(lower, pattern) {
            return canonical
        }
    }
    
    // Title case as default
    return cases.Title(language.English).String(lower)
}
```

---

## Vietnamese NLP Notes

```yaml
# Đặc thù tiếng Việt cần xử lý
challenges:
  - no_word_boundaries: "Tiếng Việt không có dấu cách rõ ràng giữa âm tiết"
  - tone_marks: "Dấu thanh ảnh hưởng nghĩa: ma/má/mà/mã/mạ/mả"
  - compound_words: "đăng ký = 'đăng' + 'ký' (không tách được)"

tools:
  tokenization: "underthesea (Python) hoặc ViTokenizer"
  ner: "PhoNER (Vietnamese NER model)"
  embedding: "PhoBERT cho Vietnamese text"
  
strategy:
  primary: "LLM-based extraction (GPT-4o / Claude handles Vietnamese well)"
  fallback: "Rule-based patterns for common structures"
  validation: "Post-processing normalization rules"
```

---

## Checklist

- [ ] Text preprocessing pipeline xử lý được cả Vietnamese và English
- [ ] NER patterns bao gồm domain-specific terms của dự án
- [ ] Semantic deduplication threshold được calibrate (mặc định 0.85)
- [ ] Entity normalization dictionary đã build cho domain cụ thể
- [ ] SVO extraction được test với requirement sentences thực tế
- [ ] Eval set có ≥ 50 annotated requirement paragraphs
