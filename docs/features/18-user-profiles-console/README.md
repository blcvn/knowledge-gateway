# Feature 18 — User Profiles Console

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

User Profiles Console cho phép Product Manager và AI Engineer xem, phân tích, và configure structured user profiles được Memobase YOLO Engine tạo ra. Bao gồm buffer monitoring, profile explorer, event timeline, và context preview.

---

## Business Logic

### Profile Explorer

Hiển thị tất cả users và profiles của họ:
- List users với profile count, last updated, buffer status
- Per-user: structured profile (key/value/category/score)
- Filter theo category: preference/fact/goal/habit
- Sort theo score (highest confidence first)

### Buffer Monitoring

Real-time monitoring của Memobase buffer:
- Buffer size per user (current blobs count)
- Threshold: 20 blobs → trigger flush
- Last flush timestamp
- Flush history

### Event Timeline per User

Xem lịch sử tương tác của user qua AI:
- Timeline sorted by timestamp
- EventType: ingestion/search/memory/profile/admin
- GistText: AI-generated summary cho mỗi event

### Context Preview

Xem context string mà AI sẽ nhận được:
- Render prompt-ready context string
- Token count estimate
- Configurable token budget để xem context ở các budget khác nhau

### Profile Configuration

Admin có thể configure:
- `FlushThreshold`: Số blobs trước khi auto-flush (default: 20)
- `ContextTokenBudget`: Token budget cho context assembly
- `ProfileCategories`: Enable/disable specific categories

---

## Dataflow

```
Console UI (User Profiles)
        │
        ├── GET /v1/console/profiles
        │         └── List all users với profile summaries
        │
        ├── GET /v1/console/profiles/{user_id}
        │         └── Full profile: [{key, value, category, score}]
        │
        ├── GET /v1/console/profiles/{user_id}/events
        │         └── Event timeline: [{event_type, gist_text, timestamp}]
        │
        ├── GET /v1/console/profiles/{user_id}/context
        │         └── Prompt-ready context string + token_count
        │
        ├── GET /v1/console/profiles/{user_id}/buffers
        │         └── Buffer status: {count, threshold, last_flush}
        │
        ├── GET /v1/console/profiles/config
        │         └── Current Memobase configuration
        │
        └── PUT /v1/console/profiles/config
                  └── Update configuration
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/profiles` | List all user profiles |
| `GET` | `/v1/console/profiles/{user_id}` | User profile detail |
| `GET` | `/v1/console/profiles/{user_id}/events` | Event timeline |
| `GET` | `/v1/console/profiles/{user_id}/context` | Context preview |
| `GET` | `/v1/console/profiles/{user_id}/buffers` | Buffer status |
| `GET` | `/v1/console/profiles/config` | Get config |
| `PUT` | `/v1/console/profiles/config` | Update config |
