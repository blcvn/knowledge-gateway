# SOL-GR-005 — Solution: Custom Ontology

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-005 |
| **CR** | CR-GR-005 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/graphiti-cognify` |

---

## 1. Phân tích

Allow tenants to define entity types and edge types that guide LLM extraction.

```go
// services/graphiti-knowledge/internal/domain/ontology.go [NEW]
type Ontology struct {
    TenantID    string
    EntityTypes []string  // ["Person", "Project", "Decision", "Task"]
    EdgeTypes   []string  // ["ASSIGNED_TO", "DEPENDS_ON", "DECIDED_BY"]
    Properties  map[string][]string  // entity_type → allowed properties
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-knowledge/internal/domain/ontology.go` | NEW |
| `deployment/dev/migrations/0XX_ontologies.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] Tenants can CRUD ontology via API
- [ ] Ontology applied during entity extraction
- [ ] Default ontology for tenants without custom ontology
