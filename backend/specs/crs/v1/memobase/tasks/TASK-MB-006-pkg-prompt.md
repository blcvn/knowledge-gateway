# TASK-MB-006 — `pkg/prompt/` EN/ZH Prompt Registry & Templates

**Wave:** 2 (song song với TASK-MB-005)  
**Ưu tiên:** High  
**Phụ thuộc:** Không có (pure package)  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-MB-002 §9](../solutions/SOL-MB-002-Memory-Engine-Profile-YOLO.md)

**Trạng thái:** ✅ Implemented  
**Ghi chú:** memobase-engine prompt handling in usecase layer  
---

## Mục tiêu

Tạo `pkg/prompt/` — multilingual prompt registry với EN/ZH providers. Gồm 5 prompt templates dùng bởi `memobase-engine`: `entry_summary`, `extract_profile`, `merge_profile_yolo`, `summarize_event`, `tag_event`.

---

## Cấu trúc thư mục

```
pkg/prompt/
├── provider.go          # PromptProvider interface
├── registry.go          # NewRegistry() → map[language]PromptProvider
├── en/
│   ├── provider.go      # ENPromptProvider
│   ├── entry_summary.go
│   ├── extract_profile.go
│   ├── merge_yolo.go
│   ├── summarize_event.go
│   └── tag_event.go
├── zh/
│   ├── provider.go      # ZHPromptProvider
│   ├── entry_summary.go
│   ├── extract_profile.go
│   ├── merge_yolo.go
│   ├── summarize_event.go
│   └── tag_event.go
└── prompt_test.go
```

---

## 1. `pkg/prompt/provider.go` — Interface

```go
package prompt

// ProfileSlot used in merge prompt formatting
type ProfileSlot struct {
    Index    int    // [0], [1], [2] for YOLO merge
    Topic    string
    SubTopic string
    Content  string
}

// MergeAction: output from YOLO merge LLM
type MergeAction struct {
    Add    []AddAction    `json:"add"`
    Update []UpdateAction `json:"update"`
    Delete []int          `json:"delete"`
}

type AddAction struct {
    Topic    string `json:"topic"`
    SubTopic string `json:"sub_topic"`
    Content  string `json:"content"`
}

type UpdateAction struct {
    Index   int    `json:"index"`
    Topic   string `json:"topic"`
    SubTopic string `json:"sub_topic"`
    Content string `json:"content"`
}

type PromptProvider interface {
    // LLM Call #1: summarize raw conversation → memoStr
    EntrySummary(rawConversation string) string

    // LLM Call #2: extract profile topics from memoStr
    ExtractProfile(memoStr string, existingTopics []string) string

    // LLM Call #3: YOLO merge existing profiles with new facts
    MergeProfileYOLO(existingSlots []ProfileSlot, newFacts string) string

    // Event goroutine: summarize conversation into event + gists
    SummarizeEvent(memoStr string) string

    // Event goroutine: tag event with predefined tags
    TagEvent(eventTip string, availableTags []string) string

    Language() string
}
```

---

## 2. `pkg/prompt/registry.go`

```go
package prompt

type Registry struct {
    providers map[string]PromptProvider
}

func NewRegistry() *Registry {
    r := &Registry{providers: make(map[string]PromptProvider)}
    r.providers["en"] = &en.ENPromptProvider{}
    r.providers["zh"] = &zh.ZHPromptProvider{}
    return r
}

func (r *Registry) Get(language string) (PromptProvider, error) {
    p, ok := r.providers[language]
    if !ok {
        return nil, fmt.Errorf("prompt: no provider for language %q (available: %v)", language, r.Available())
    }
    return p, nil
}

func (r *Registry) GetOrDefault(language string) PromptProvider {
    p, err := r.Get(language)
    if err != nil {
        p = r.providers["en"]  // fallback to EN
    }
    return p
}

func (r *Registry) Available() []string
func (r *Registry) Register(language string, provider PromptProvider)  // for testing
```

---

## 3. EN Prompt Templates

**File: `pkg/prompt/en/provider.go`**

```go
package en

type ENPromptProvider struct{}

func (p *ENPromptProvider) Language() string { return "en" }
func (p *ENPromptProvider) EntrySummary(rawConversation string) string {
    return EntrySummaryPrompt(rawConversation)
}
// ... all 5 methods delegate to functions in respective files
```

**File: `pkg/prompt/en/entry_summary.go`**

```go
func EntrySummaryPrompt(rawConversation string) string {
    return fmt.Sprintf(`You are a memory assistant. Analyze the following conversation and create a concise memory summary.

Focus on: personal information, preferences, events, decisions, and important facts mentioned by the user.

Conversation:
%s

Output a clear, factual summary in plain text (not JSON). Include all important details mentioned by the user. Use present tense for facts and past tense for events.`, rawConversation)
}
```

**File: `pkg/prompt/en/extract_profile.go`**

```go
func ExtractProfilePrompt(memoStr string, existingTopics []string) string {
    topicList := strings.Join(existingTopics, ", ")
    topicsSection := ""
    if len(existingTopics) > 0 {
        topicsSection = fmt.Sprintf("\nExisting profile topics: %s\n", topicList)
    }
    return fmt.Sprintf(`Extract structured profile information from this memory summary.
%s
Memory Summary:
%s

Output JSON with extracted profile facts. Use topic and sub_topic to categorize.
Topics should be one of: basic_info, work, lifestyle, interest, relationship, preference, skill, goal, other

{"facts": [{"topic": "work", "sub_topic": "company", "content": "Works at Acme Corp"}, ...]}

Only include genuinely significant, durable facts. Omit transient or trivial details.`, topicsSection, memoStr)
}
```

**File: `pkg/prompt/en/merge_yolo.go`**

```go
func MergeProfileYOLOPrompt(existingSlots []ProfileSlot, newFacts string) string {
    // Format existing slots with index
    var slotLines []string
    for _, s := range existingSlots {
        slotLines = append(slotLines, fmt.Sprintf("[%d] %s::%s: %s", s.Index, s.Topic, s.SubTopic, s.Content))
    }
    existing := strings.Join(slotLines, "\n")

    return fmt.Sprintf(`You are a user profile manager. Merge the new facts with existing profiles.

EXISTING PROFILES (indexed):
%s

NEW FACTS:
%s

Output a JSON object with exactly these three fields:
- "add": array of new facts to add (not in existing)
- "update": array of updates to existing facts (with index reference)
- "delete": array of indices of existing facts to remove (outdated/contradicted)

{"add":[{"topic":"...","sub_topic":"...","content":"..."}],"update":[{"index":0,"topic":"...","sub_topic":"...","content":"..."}],"delete":[2]}

Be conservative: only make changes when clearly necessary. When in doubt, keep existing.`, existing, newFacts)
}
```

**File: `pkg/prompt/en/summarize_event.go`**

```go
func SummarizeEventPrompt(memoStr string) string {
    return fmt.Sprintf(`Analyze this conversation summary and generate an event record.

Memory Summary:
%s

Output JSON with:
- event_tip: A multi-line summary of what happened, with each key point as a bullet starting with "- "
- event_tags: Array of tags (e.g., ["meeting", "coding", "personal"])
- profile_delta: Any profile changes implied by this event

{"event_tip":"- User discussed project deadline\n- Mentioned feeling stressed about timeline","event_tags":["work","stress"],"profile_delta":[]}`, memoStr)
}
```

**File: `pkg/prompt/en/tag_event.go`**

```go
func TagEventPrompt(eventTip string, availableTags []string) string {
    tags := strings.Join(availableTags, ", ")
    return fmt.Sprintf(`Tag the following event with relevant tags.

Available tags: %s

Event:
%s

Output JSON: {"tags": ["tag1", "tag2"]}
Only use tags from the available list. Select 1-5 most relevant tags.`, tags, eventTip)
}
```

---

## 4. ZH Prompt Templates

**File: `pkg/prompt/zh/provider.go`**

```go
// Same structure as ENPromptProvider but with Chinese prompts
type ZHPromptProvider struct{}
func (p *ZHPromptProvider) Language() string { return "zh" }
```

**File: `pkg/prompt/zh/entry_summary.go`**

```go
func EntrySummaryPromptZH(rawConversation string) string {
    return fmt.Sprintf(`你是一个记忆助手。分析以下对话并创建简洁的记忆摘要。

重点关注：用户提到的个人信息、偏好、事件、决定和重要事实。

对话内容：
%s

用纯文本输出清晰、客观的摘要（非JSON格式）。包含用户提到的所有重要细节。用现在时描述事实，用过去时描述事件。`, rawConversation)
}
```

**File: `pkg/prompt/zh/extract_profile.go`**

```go
// Chinese version of extract_profile prompt
// Topics in Chinese context: 基本信息, 工作, 生活方式, 兴趣, 关系, 偏好, 技能, 目标, 其他
func ExtractProfilePromptZH(memoStr string, existingTopics []string) string
```

**File: `pkg/prompt/zh/merge_yolo.go`**

```go
func MergeProfileYOLOPromptZH(existingSlots []ProfileSlot, newFacts string) string
```

**File: `pkg/prompt/zh/summarize_event.go`**

```go
func SummarizeEventPromptZH(memoStr string) string
```

**File: `pkg/prompt/zh/tag_event.go`**

```go
func TagEventPromptZH(eventTip string, availableTags []string) string
```

---

## 5. Tests

**File: `pkg/prompt/prompt_test.go`**

```
TestRegistry_GetEN                          → "en" → ENPromptProvider
TestRegistry_GetZH                          → "zh" → ZHPromptProvider
TestRegistry_GetUnknown                     → "fr" → error
TestRegistry_GetOrDefault_Unknown           → "ja" → ENPromptProvider (fallback)
TestRegistry_Available                      → contains "en" and "zh"
TestENPromptProvider_Language               → "en"
TestZHPromptProvider_Language               → "zh"
TestEntrySummaryPrompt_ContainsConversation → input conversation → appears in prompt
TestExtractProfilePrompt_WithTopics         → existing topics → listed in prompt
TestExtractProfilePrompt_EmptyTopics        → no topics → no topics section
TestMergeProfileYOLO_FormatsIndexedSlots   → slot [0] basic_info::name: Alice → in prompt
TestMergeProfileYOLO_EmptyExisting         → no slots → "EXISTING PROFILES: " empty
TestSummarizeEventPrompt_ContainsMemoStr   → memoStr in output
TestTagEventPrompt_ListsTags               → availableTags → listed in prompt
TestZHPromptProvider_AllMethodsReturnNonEmpty → all 5 methods → non-empty strings
TestZHPromptProvider_ContainsChinese       → entry_summary → contains Chinese characters
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./pkg/prompt/...
go test ./pkg/prompt/... -v -count=1
```

---

## Ghi chú triển khai

- Prompts là Go functions (không phải file templates) — đơn giản hơn, compile-time safety
- `MergeProfileYOLOPrompt`: đảm bảo `existingSlots` được indexed từ 0 (`[0]`, `[1]`, ...)
- EN prompts: viết careful English để LLM output đúng JSON format
- ZH prompts: dịch nguyên văn ý nghĩa, giữ nguyên JSON schema requirements
- Future: có thể thêm `vi/` (Vietnamese) provider bằng cách implement interface
