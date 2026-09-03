# TASK-CORE-003 — LLM Memory Type Classifier

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-003 |
| **Wave** | 1 |
| **Solution** | [SOL-CORE-001](../solutions/SOL-CORE-001-Unified-Memory-Router.md) §2.3 |
| **Component** | `gateway/internal/usecase/classifier.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-CORE-001 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** RouteUseCase.Classify() in route.go; LLM-based classification via registered adapter
---

## Mục tiêu

Implement Bifrost LLM classifier để phân loại nội dung khi `type=auto`.

---

## Công việc cụ thể

### 1. `gateway/internal/port/classifier.go` [NEW]

```go
package port

type MemoryClassifier interface {
    Classify(ctx context.Context, content string) (string, error)
}
```

### 2. `gateway/internal/usecase/classifier.go` [NEW]

```go
package usecase

const classifyPrompt = `You are a memory classification system.
Classify the following content into exactly ONE category:
- episodic: events, activities, what happened
- semantic: knowledge, facts, concepts
- conversational: conversation messages, chat history
- profile: personal info, preferences, user traits
- procedural: instructions, workflows, how-to steps
- adaptive: personal learning patterns, behavioral evolution

Content: %s

Respond with ONLY one word (the category name).`

type LLMClassifier struct {
    llm port.LLMClient
}

func NewLLMClassifier(llm port.LLMClient) *LLMClassifier {
    return &LLMClassifier{llm: llm}
}

func (c *LLMClassifier) Classify(ctx context.Context, content string) (string, error) {
    // Truncate content to 500 chars for classification
    if len(content) > 500 {
        content = content[:500]
    }

    resp, err := c.llm.Complete(ctx, &port.CompletionRequest{
        Prompt:      fmt.Sprintf(classifyPrompt, content),
        MaxTokens:   5,
        Temperature: 0.0,
        Task:        "memory_classify",
    })
    if err != nil {
        return "", fmt.Errorf("classifier LLM call failed: %w", err)
    }

    t := strings.TrimSpace(strings.ToLower(resp.Content))
    valid := map[string]bool{
        "episodic": true, "semantic": true, "conversational": true,
        "profile": true, "procedural": true, "adaptive": true,
    }
    if !valid[t] {
        return domain.MemoryTypeSemantic, nil // fallback
    }
    return t, nil
}
```

### 3. Unit test: `gateway/internal/usecase/classifier_test.go`

```go
func TestClassifier_ValidResponse(t *testing.T) {
    mockLLM := &mockLLMClient{response: "episodic"}
    c := NewLLMClassifier(mockLLM)
    result, err := c.Classify(context.Background(), "Yesterday I went to the store")
    assert.NoError(t, err)
    assert.Equal(t, "episodic", result)
}

func TestClassifier_InvalidResponse_FallsBackToSemantic(t *testing.T) {
    mockLLM := &mockLLMClient{response: "garbage"}
    c := NewLLMClassifier(mockLLM)
    result, _ := c.Classify(context.Background(), "some content")
    assert.Equal(t, "semantic", result)
}

func TestClassifier_LLMError_ReturnsError(t *testing.T) {
    mockLLM := &mockLLMClient{err: errors.New("llm unavailable")}
    c := NewLLMClassifier(mockLLM)
    _, err := c.Classify(context.Background(), "content")
    assert.Error(t, err)
}
```

---

## Acceptance Criteria

- [ ] Classifier returns valid MemoryType string
- [ ] Invalid LLM response → fallback to "semantic"
- [ ] LLM error → return error (caller handles fallback)
- [ ] Content truncated to 500 chars before sending
- [ ] `go test ./gateway/internal/usecase/...` passes

## Files

```
gateway/internal/port/classifier.go         [NEW]
gateway/internal/usecase/classifier.go      [NEW]
gateway/internal/usecase/classifier_test.go [NEW]
```
