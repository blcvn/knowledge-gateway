# SOL-COGNEE-003 — Solution: DataPoint Custom Schema

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-003 |
| **CR** | [CR-COGNEE-003](../../../../docs/crs/v1/cognee/CR-COGNEE-003*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |

---
## 1. Giải pháp

Custom schema = allow tenants to define domain-specific entity types and properties for their knowledge graph.

### 1.1 `services/cognee-ingestion/internal/domain/schema.go` [NEW]

```go
type CustomSchema struct {
    TenantID    string
    EntityTypes []EntityTypeDef   // custom entity types
    EdgeTypes   []EdgeTypeDef     // custom relationship types
    Validators  []ValidationRule  // per-field validation
}

type EntityTypeDef struct {
    Name       string            // "Invoice", "Contract", "Regulation"
    Properties map[string]string // field_name → type (string, number, date)
    Required   []string
}
```

### 1.2 Schema-aware entity extraction

```go
// LLM prompt includes schema to guide extraction
func (u *CognifyUseCase) ExtractWithSchema(ctx context.Context, text string, schema *CustomSchema) ([]Entity, error) {
    prompt := buildSchemaAwarePrompt(text, schema)
    return u.llm.ExtractEntities(ctx, prompt)
}
```

## 2. File Changes

| File | Action |
|---|---|
| `services/cognee-ingestion/internal/domain/schema.go` | NEW |
| `services/cognee-cognify/internal/usecase/cognify.go` | MODIFY — schema-aware extraction |
| `deployment/dev/migrations/0XX_custom_schemas.sql` | NEW |

## 3. Acceptance Criteria

- [ ] Tenants can define custom entity/edge types via API
- [ ] Schema applied during Cognify pipeline
- [ ] Schema validation before ingestion
