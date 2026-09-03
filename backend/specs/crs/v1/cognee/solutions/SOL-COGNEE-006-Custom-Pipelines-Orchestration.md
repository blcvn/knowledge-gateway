# SOL-COGNEE-006 — Solution: Custom Pipelines Orchestration

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-006 |
| **CR** | [CR-COGNEE-006](../../../../docs/crs/v1/cognee/CR-COGNEE-006*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

---
## 1. Giải pháp

Custom pipelines = tenants define which pipeline steps to run and in what order.

### 1.1 `services/cognee-cognify/internal/domain/pipeline.go` [NEW]

```go
type PipelineDefinition struct {
    TenantID  string
    Name      string
    Steps     []PipelineStep
}

type PipelineStep struct {
    Name    string  // "classify", "chunk", "extract", "ontology", "embed", "community"
    Config  map[string]any
    Enabled bool
}

// Default pipeline
var DefaultPipeline = PipelineDefinition{
    Steps: []PipelineStep{
        {Name: "classify", Enabled: true},
        {Name: "chunk", Enabled: true},
        {Name: "extract", Enabled: true},
        {Name: "ontology", Enabled: true},
        {Name: "embed", Enabled: true},
        {Name: "community", Enabled: true},
    },
}
```

## 2. Acceptance Criteria

- [ ] Tenants can enable/disable pipeline steps
- [ ] Custom step ordering supported
- [ ] Pipeline definition stored per tenant
