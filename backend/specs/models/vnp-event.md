# vnp-event — Data Models

> **Service**: `services/vnp-event`
> **Role**: Cross-engine bi-temporal event store. Aggregates events from all memory engines (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory) into a unified timeline with vector search.

---

## UserEvent

```go
type UserEvent struct {
    ID        uuid.UUID   `json:"id"`
    UserID    uuid.UUID   `json:"user_id"`
    TenantID  uuid.UUID   `json:"tenant_id"`
    Source    EventSource `json:"source"`
    Content   string      `json:"content"`
    Tags      []string    `json:"tags"`
    Embedding []float32   `json:"-"`              // vector(1536), not exposed in JSON
    CreatedAt time.Time   `json:"created_at"`
    ValidAt   time.Time   `json:"valid_at"`       // bi-temporal: when this event was true
    InvalidAt *time.Time  `json:"invalid_at,omitempty"` // bi-temporal: when this event ceased being true
}

type EventSource string
// COGNEE | GRAPHITI | MEMOBASE | OPENVIKING | ZEP | SUPERMEMORY
```

---

## EventGist

```go
// EventGist summarizes a batch of events (LLM-generated condensed summary).
type EventGist struct {
    ID        uuid.UUID   `json:"id"`
    EventIDs  []uuid.UUID `json:"event_ids"`
    Summary   string      `json:"summary"`
    Embedding []float32   `json:"-"`
    CreatedAt time.Time   `json:"created_at"`
}
```

---

## TimelineEntry

```go
// TimelineEntry is a single item in a user's timeline view.
type TimelineEntry struct {
    Event *UserEvent `json:"event"`
    Score float64    `json:"score,omitempty"` // Relevance score for search results
}
```

---

## Notes

- Supports **bi-temporal** queries: `ValidAt` / `InvalidAt` allow point-in-time lookups.
- `Embedding` field uses `pgvector` extension in PostgreSQL (1536 dims, not JSON-exposed).
- `EventGist` provides LLM-condensed summaries for large event batches.

---

## Sources
- [`services/vnp-event/domain/model/event.go`](../../services/vnp-event/domain/model/event.go)
