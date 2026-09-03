# sm-engine — Data Models

> **Service**: `services/sm-engine`
> **Role**: Supermemory adaptive memory engine — Ebbinghaus forgetting-curve memory, user profiling, and document management.

---

## memory — Ebbinghaus Memory

```go
// Memory with Ebbinghaus forgetting-curve decay.
type Memory struct {
    ID         uuid.UUID `json:"id"`
    TenantID   uuid.UUID `json:"tenant_id"`
    UserID     uuid.UUID `json:"user_id"`
    Content    string    `json:"content"`
    Category   string    `json:"category"`
    Strength   float64   `json:"strength"`    // 0.0 - 1.0, decays over time
    HalfLife   float64   `json:"half_life"`   // Hours until 50% strength
    LastAccess time.Time `json:"last_access"`
    CreatedAt  time.Time `json:"created_at"`
}

// CurrentStrength = initial_strength × e^(-t / half_life × ln(2))
```

### Relation

```go
type Relation struct {
    ID        uuid.UUID `json:"id"`
    SourceID  uuid.UUID `json:"source_id"`
    TargetID  uuid.UUID `json:"target_id"`
    Type      string    `json:"type"`   // related | contradicts | supersedes
    Weight    float64   `json:"weight"`
    CreatedAt time.Time `json:"created_at"`
}
```

---

## document — Document Storage

```go
type Document struct {
    ID          uuid.UUID `json:"id"`
    TenantID    uuid.UUID `json:"tenant_id"`
    UserID      uuid.UUID `json:"user_id"`
    Title       string    `json:"title"`
    URL         string    `json:"url,omitempty"`
    ContentType string    `json:"content_type"` // article | page | note
    RawContent  string    `json:"raw_content"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Chunk struct {
    ID         uuid.UUID `json:"id"`
    DocumentID uuid.UUID `json:"document_id"`
    TenantID   uuid.UUID `json:"tenant_id"`
    Content    string    `json:"content"`
    Index      int       `json:"index"`   // Position within document
    Tokens     int       `json:"tokens"`  // Token count
}
```

---

## profile — Adaptive User Profile

```go
type Profile struct {
    ID                uuid.UUID          `json:"id"`
    TenantID          uuid.UUID          `json:"tenant_id"`
    UserID            uuid.UUID          `json:"user_id"`
    StaticPreferences []StaticPreference `json:"static_preferences"` // User-defined
    DynamicTraits     []DynamicTrait     `json:"dynamic_traits"`     // System-inferred
    UpdatedAt         time.Time          `json:"updated_at"`
}

type StaticPreference struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type DynamicTrait struct {
    Category   TraitCategory `json:"category"`
    Name       string        `json:"name"`
    Confidence float64       `json:"confidence"` // 0.0 - 1.0
    InferredAt time.Time     `json:"inferred_at"`
}

type TraitCategory string
// interest | behavior | preference | expertise
```

---

## Sources
- [`services/sm-engine/domain/memory/entity.go`](../../services/sm-engine/domain/memory/entity.go)
- [`services/sm-engine/domain/document/entity.go`](../../services/sm-engine/domain/document/entity.go)
- [`services/sm-engine/domain/profile/entity.go`](../../services/sm-engine/domain/profile/entity.go)
