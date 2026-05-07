---
skill_id: SKILL-008
version: 1.0.0
status: active
priority: P0
group: AI & Data Engineering
created_at: 2026-04-24
---

# SKILL-008 · LLM Engineering & Prompt Design

## Mô tả

Thiết kế và tối ưu hóa prompts cho LLM để thực hiện các tác vụ phân tích văn bản phức tạp — trích xuất entities, phân loại đoạn văn, suy luận business rules — với độ chính xác và tính ổn định cao.

## Agents sử dụng

- `requirement-parser-agent`
- `semantic-extractor-agent`

---

## Năng lực cốt lõi

### 1. Prompt Engineering

- **Zero-shot prompting**: Hướng dẫn LLM thực hiện task mà không cần ví dụ
- **Few-shot prompting**: Cung cấp 3-5 examples điển hình để LLM học pattern
- **Chain-of-Thought (CoT)**: Yêu cầu LLM suy luận từng bước trước khi đưa ra kết quả
- **Structured Output (JSON mode)**: Ép LLM trả về JSON đúng schema, dùng function calling hoặc response_format
- **System / User / Assistant roles**: Tận dụng role separation cho context rõ ràng
- **Prompt Templates**: Tạo template có slot variables, dễ tái sử dụng và test

### 2. LLM Selection & Strategy

| Task | Model được khuyên dùng | Lý do |
|------|------------------------|-------|
| Complex reasoning / extraction | GPT-4o, Claude 3.5 Sonnet | Độ chính xác cao |
| Fast classification | GPT-3.5-turbo, Claude Haiku | Cost-efficient |
| Long document processing | Gemini 1.5 Pro (1M token) | Context window lớn |
| Local / offline | Llama 3.1 70B (Ollama) | Privacy, no API cost |
| Code generation | GPT-4o, Claude 3.5 Sonnet | Best code quality |

### 3. Output Parsing & Validation

```go
// Pattern chuẩn cho Golang LLM output parsing
type LLMExtractionResult struct {
    Entities    []Entity    `json:"entities"`
    Relations   []Relation  `json:"relations"`
    Confidence  float64     `json:"confidence"`
}

func ParseLLMResponse(raw string) (*LLMExtractionResult, error) {
    // 1. Extract JSON block nếu LLM wrap trong markdown
    jsonStr := extractJSONBlock(raw)
    
    // 2. Unmarshal với strict validation
    var result LLMExtractionResult
    if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
        return nil, fmt.Errorf("malformed LLM output: %w", err)
    }
    
    // 3. Semantic validation
    if result.Confidence < 0.6 {
        return nil, ErrLowConfidence
    }
    return &result, nil
}
```

### 4. Hallucination Mitigation

- **Grounding**: Luôn include source text trong prompt, yêu cầu LLM trích dẫn từ text gốc
- **Confidence scoring**: Yêu cầu LLM tự đánh giá độ tin cậy (0.0 - 1.0)
- **Fallback chain**: Nếu confidence < threshold → retry với different prompt → human review queue
- **Constraint prompts**: Rõ ràng yêu cầu LLM KHÔNG bịa thêm thông tin ngoài text gốc
- **Output verification**: Cross-validate entities được extract với text gốc bằng string search

### 5. Context Window Management

```
Document Chunking Strategy:
├── Max chunk size: 3000 tokens (giữ safety margin)
├── Overlap: 200 tokens giữa chunks (tránh mất context tại boundary)
├── Chunk boundary: Ưu tiên cắt tại paragraph/sentence boundary
└── RAG pattern:
    ├── Embed tất cả chunks → vector store (pgvector / Chroma)
    ├── Query time: embed question → similarity search → retrieve top-k chunks
    └── Combine retrieved chunks + question vào final prompt
```

### 6. LLM Cost Optimization

- **Token counting**: Luôn count tokens trước khi gửi request (`tiktoken` cho OpenAI)
- **Prompt caching**: OpenAI / Anthropic hỗ trợ prefix caching — đặt system prompt cố định ở đầu
- **Request batching**: Batch nhiều documents nhỏ vào 1 request thay vì N requests riêng lẻ
- **Model routing**: Dùng cheap model cho classification, expensive model chỉ cho complex extraction
- **Budget enforcement**: Hard limit token/day per project, alert khi reach 80%

### 7. Evaluation & Benchmarking

```yaml
# Evaluation framework cho LLM extraction
eval_config:
  test_set: docs/evals/extraction-test-set.jsonl  # 100+ annotated examples
  metrics:
    - precision: "extracted entities đúng / total extracted"
    - recall: "extracted entities đúng / total ground truth"
    - f1_score: "harmonic mean of precision & recall"
    - hallucination_rate: "entities không có trong source / total extracted"
  threshold:
    precision: 0.85
    recall: 0.80
    hallucination_rate: 0.05  # max 5%
```

---

## Patterns & Recipes

### Pattern 1: Structured Entity Extraction Prompt

```
SYSTEM: You are an expert business analyst. Extract entities from requirement documents.
Return ONLY valid JSON matching the schema. Do NOT invent information not in the text.

USER: Extract all actors and actions from this requirement text:

<text>
{requirement_text}
</text>

Return JSON:
{
  "actors": [{"name": string, "type": "human|system|external", "source_quote": string}],
  "actions": [{"verb": string, "actor": string, "object": string, "source_quote": string}],
  "confidence": float (0.0-1.0)
}
```

### Pattern 2: Classification với Few-shot

```
Classify this paragraph into ONE category:
- OVERVIEW: High-level description of the system
- FUNCTIONAL: Specific feature or user story
- CONSTRAINT: Business rule or limitation
- API: Technical API specification

Examples:
Text: "The system shall allow users to login with email and password" → FUNCTIONAL
Text: "The platform integrates with Stripe payment gateway" → API
Text: "Response time must be under 200ms" → CONSTRAINT

Now classify:
Text: "{paragraph}"
Category:
```

### Pattern 3: Retry với Exponential Backoff

```go
func CallLLMWithRetry(ctx context.Context, prompt string) (string, error) {
    backoff := []time.Duration{1*time.Second, 2*time.Second, 4*time.Second}
    
    for attempt, delay := range backoff {
        result, err := llmClient.Complete(ctx, prompt)
        if err == nil {
            return result, nil
        }
        
        if isRateLimitError(err) {
            time.Sleep(delay)
            continue
        }
        
        return "", fmt.Errorf("LLM call failed after %d attempts: %w", attempt+1, err)
    }
    return "", ErrMaxRetriesExceeded
}
```

---

## Checklist trước khi triển khai

- [ ] Prompt đã được test với ít nhất 20 diverse examples
- [ ] Output schema được validate bằng JSON Schema
- [ ] Confidence threshold được set (mặc định: 0.75)
- [ ] Fallback strategy đã được implement
- [ ] Token budget đã được tính toán cho worst case
- [ ] Cost per document đã được estimate
- [ ] Eval set đã có ít nhất 50 annotated examples
- [ ] Hallucination rate < 5% trên eval set

---

## Tài liệu liên kết

- `docs/standards/llm-guidelines.md`
- `services/*/docs/prompts/`
- `docs/evals/`
