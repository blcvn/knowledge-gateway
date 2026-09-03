# Change Request: CR-GR-005 — Custom Ontology & Prescribed Entity/Edge Types

**CR ID:** CR-GR-005  
**Component:** `services/graphiti-knowledge` [EXTEND] | `services/graphiti-ingestion` [EXTEND]  
**Priority:** High  
**Status:** In Progress
**Reference:** graphiti PRD §5.1 (Custom Ontology), SRS §4, URD §SOP-005  
**Maps to Python:** `graphiti_core/nodes.py (BaseNode)`, `graphiti_core/edges.py (BaseEdge)`, Pydantic models

---

## 1. Mô tả

Triển khai **Custom Ontology** (Prescribed Ontology) — cho phép users định nghĩa trước entity types và edge types thông qua JSON Schema. LLM extraction sẽ constrain vào các types đã định nghĩa, thay vì tự do tạo bất kỳ label nào.

**Hai chế độ hoạt động:**
- **Learned Ontology** (default): LLM tự quyết định entity labels và relationship types.
- **Prescribed Ontology**: User định nghĩa schema → LLM extraction bị constrain.

---

## 2. Vấn đề hiện tại

Graphiti hiện tại (nếu đã có service) chỉ hỗ trợ **learned ontology** — LLM tự extract bất kỳ entity label nào. Không có:
- ❌ `EntityTypeSchema` validation (user-defined types).
- ❌ `EdgeTypeSchema` validation (user-defined relationships).
- ❌ Custom ontology constraints enforced trong extraction prompts.
- ❌ API để register/manage custom ontologies per tenant.
- ❌ Domain-specific ontology presets (e.g., CRM, HR, Software Project).

---

## 3. Thay đổi đề xuất

### 3.1. Entity Type Schema

```go
// pkg/graph/ontology.go

// EntityTypeSchema — prescribed entity type definition
type EntityTypeSchema struct {
    // Type name that LLM will use as label
    Name        string `json:"name"`
    // Description: guide LLM on when to classify as this type
    Description string `json:"description"`
    // Required properties for this entity type
    Properties  []OntologyProperty `json:"properties,omitempty"`
    // Example entities for few-shot in LLM prompt
    Examples    []string `json:"examples,omitempty"`
}

type OntologyProperty struct {
    Name        string `json:"name"`
    Type        string `json:"type"`  // string | number | boolean | datetime
    Description string `json:"description"`
    Required    bool   `json:"required"`
}

// Example: CRM ontology
var CRMEntityTypes = map[string]EntityTypeSchema{
    "Customer": {
        Name:        "Customer",
        Description: "A person or organization that is a customer or prospect",
        Properties: []OntologyProperty{
            {Name: "email", Type: "string"},
            {Name: "company", Type: "string"},
            {Name: "tier", Type: "string"},
        },
        Examples: []string{"Acme Corp", "John Smith at TechCo"},
    },
    "Deal": {
        Name:        "Deal",
        Description: "A sales opportunity or deal",
        Properties: []OntologyProperty{
            {Name: "value", Type: "number"},
            {Name: "stage", Type: "string"},
        },
    },
    "Product": {
        Name:        "Product",
        Description: "A product or service offered",
    },
}
```

### 3.2. Edge Type Schema

```go
// EdgeTypeSchema — prescribed relationship definition
type EdgeTypeSchema struct {
    Name         string `json:"name"`
    Description  string `json:"description"`
    // Which entity types can be source/target
    SourceTypes  []string `json:"source_types,omitempty"`
    TargetTypes  []string `json:"target_types,omitempty"`
    // Extra properties carried on the edge
    Properties   []OntologyProperty `json:"properties,omitempty"`
    Examples     []string `json:"examples,omitempty"`
}

// Example: CRM edge types
var CRMEdgeTypes = map[string]EdgeTypeSchema{
    "BOUGHT": {
        Name:        "BOUGHT",
        Description: "Customer purchased a product",
        SourceTypes: []string{"Customer"},
        TargetTypes: []string{"Product"},
        Properties: []OntologyProperty{
            {Name: "quantity", Type: "number"},
            {Name: "date", Type: "datetime"},
        },
    },
    "ASSIGNED_TO": {
        Name:        "ASSIGNED_TO",
        Description: "Deal is assigned to a sales rep",
        SourceTypes: []string{"Deal"},
        TargetTypes: []string{"SalesRep"},
    },
}
```

### 3.3. Ontology-Constrained Extraction Prompt

```go
// internal/adapter/prompt/extract_nodes.go

// When entity_types provided:
// System prompt includes:
// "Extract ONLY the following entity types:
//  - Customer: A person or organization that is a customer...
//  - Deal: A sales opportunity...
//  - Product: A product or service...
// Do NOT extract entities that don't match these types."

// LLM response schema changes from:
// {"name": "string", "label": "any string"}
// to:
// {"name": "string", "label": "Customer|Deal|Product", "properties": {...}}
```

### 3.4. Ontology Registry (per-tenant)

```go
// internal/domain/ontology_registry.go
// Stored in PostgreSQL or Redis, keyed by group_id/tenant_id

type OntologyRegistry struct {
    GroupID     string
    EntityTypes map[string]EntityTypeSchema
    EdgeTypes   map[string]EdgeTypeSchema
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type OntologyRepository interface {
    Save(ctx, registry OntologyRegistry) error
    Get(ctx, groupID string) (*OntologyRegistry, error)
    Delete(ctx, groupID string) error
}
```

### 3.5. Pre-defined Ontology Presets

```go
// pkg/graph/presets/

// HRPreset — Human Resources domain
var HRPreset = OntologyRegistry{
    EntityTypes: map[string]EntityTypeSchema{
        "Employee":   {Name: "Employee", Description: "A person employed at the company"},
        "Department": {Name: "Department", Description: "An organizational unit"},
        "Role":       {Name: "Role", Description: "A job title or position"},
        "Project":    {Name: "Project", Description: "A work project or initiative"},
    },
    EdgeTypes: map[string]EdgeTypeSchema{
        "REPORTS_TO": {SourceTypes: []string{"Employee"}, TargetTypes: []string{"Employee"}},
        "WORKS_IN":   {SourceTypes: []string{"Employee"}, TargetTypes: []string{"Department"}},
        "HAS_ROLE":   {SourceTypes: []string{"Employee"}, TargetTypes: []string{"Role"}},
        "WORKS_ON":   {SourceTypes: []string{"Employee"}, TargetTypes: []string{"Project"}},
    },
}

// SoftwareProjectPreset — Software development domain
var SoftwareProjectPreset = OntologyRegistry{
    EntityTypes: map[string]EntityTypeSchema{
        "Developer":   {Name: "Developer", Description: "A software developer"},
        "Service":     {Name: "Service", Description: "A microservice or application"},
        "Repository":  {Name: "Repository", Description: "A code repository"},
        "Feature":     {Name: "Feature", Description: "A software feature or user story"},
        "Bug":         {Name: "Bug", Description: "A software bug or issue"},
    },
    EdgeTypes: map[string]EdgeTypeSchema{
        "OWNS":      {SourceTypes: []string{"Developer"}, TargetTypes: []string{"Service", "Repository"}},
        "DEPENDS_ON": {SourceTypes: []string{"Service"}, TargetTypes: []string{"Service"}},
        "IMPLEMENTS": {SourceTypes: []string{"Developer"}, TargetTypes: []string{"Feature"}},
        "FIXES":     {SourceTypes: []string{"Developer"}, TargetTypes: []string{"Bug"}},
    },
}
```

### 3.6. API Endpoints (via Gateway)

```
# Ontology management
POST /v1/ontology/{group_id}         # Set custom ontology for group
GET  /v1/ontology/{group_id}         # Get current ontology
DELETE /v1/ontology/{group_id}       # Reset to learned ontology

# Preset application
POST /v1/ontology/{group_id}/apply-preset
  Body: { preset: "hr" | "software_project" | "crm" }

# Per-episode override (in IngestEpisodeRequest)
POST /v1/episodes
  Body: {
    ...
    entity_types: { "Customer": {...} },  # inline ontology (overrides registered)
    edge_types: { "BOUGHT": {...} }
  }
```

### 3.7. Validation Logic

```go
// After LLM extraction, validate against schema:
// 1. Entity label must be in entity_types keys
// 2. Required properties must be present
// 3. Edge source/target labels must match SourceTypes/TargetTypes
// 4. If validation fails: log warning, skip invalid entities/edges
// 5. If no valid entities extracted: continue with empty (no error)
```

---

## 4. Acceptance Criteria

- [ ] `POST /v1/ontology/project-alpha` với CRM entity_types → ingest episode "Acme Corp bought 10 licenses of ProductX" → LLM extracts `Customer(Acme Corp)` và `Product(ProductX)`, NOT generic `Organization`.
- [ ] Entity not matching prescribed types → NOT extracted (filtered out).
- [ ] Edge `BOUGHT` với `source_type=Company, target_type=Employee` → rejected (type mismatch).
- [ ] `GET /v1/ontology/project-alpha` → trả về registered ontology schema.
- [ ] `POST /v1/ontology/project-alpha/apply-preset` với `preset: "hr"` → HR ontology applied, subsequent ingestion extracts Employee/Department.
- [ ] Learned mode (no ontology): ingest "David is the CEO of Acme" → LLM freely assigns labels (Person, Company, etc.).
- [ ] Ontology cleared → back to learned mode.
- [ ] Per-episode inline ontology overrides registered ontology for that episode.
