# TASK-GR-016 — Domain Ontology Presets (HR, CRM, Software)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-016 |
| **Wave** | 3 |
| **Component** | `pkg/graph/presets/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-005 §3 |
| **Priority** | Medium |
| **Depends On** | TASK-GR-001 |
| **Estimated** | 2h |

---

## Context

Tạo 3 domain preset ontologies: HR (Human Resources), CRM (Customer Relationship Management), Software Project. Các preset này cho phép users nhanh chóng apply ontology đã được định nghĩa sẵn.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `pkg/graph/presets/hr.go` |
| CREATE | `pkg/graph/presets/crm.go` |
| CREATE | `pkg/graph/presets/software.go` |
| CREATE | `pkg/graph/presets/registry.go` |

---

## Implementation

### File 1: `pkg/graph/presets/hr.go`

```go
package presets

import "github.com/vnp-memory/pkg/graph"

var HRPreset = graph.OntologyRegistry{
    EntityTypes: map[string]graph.EntityTypeSchema{
        "Employee": {
            Name:        "Employee",
            Description: "A person employed at the organization",
            Properties: []graph.OntologyProperty{
                {Name: "department", Type: "string"},
                {Name: "role", Type: "string"},
                {Name: "start_date", Type: "datetime"},
                {Name: "employee_id", Type: "string"},
            },
            Examples: []string{"Alice the Software Engineer", "Bob in HR department"},
        },
        "Department": {
            Name:        "Department",
            Description: "An organizational unit or division within the company",
            Examples:    []string{"Engineering", "Human Resources", "Sales", "Finance"},
        },
        "Role": {
            Name:        "Role",
            Description: "A job title, position, or function",
            Examples:    []string{"Senior Software Engineer", "VP of Sales", "Product Manager"},
        },
        "Project": {
            Name:        "Project",
            Description: "A work project, initiative, or task",
            Properties:  []graph.OntologyProperty{{Name: "deadline", Type: "datetime"}, {Name: "status", Type: "string"}},
            Examples:    []string{"Q4 Product Launch", "Infrastructure Migration"},
        },
        "Team": {
            Name:        "Team",
            Description: "A group of employees working together",
            Examples:    []string{"Platform Team", "Growth Team"},
        },
    },
    EdgeTypes: map[string]graph.EdgeTypeSchema{
        "REPORTS_TO": {
            Name:        "REPORTS_TO",
            Description: "Employee reports to another employee in the management hierarchy",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Employee"},
        },
        "WORKS_IN": {
            Name:        "WORKS_IN",
            Description: "Employee works in a department",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Department"},
        },
        "HAS_ROLE": {
            Name:        "HAS_ROLE",
            Description: "Employee holds a job role or title",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Role"},
        },
        "WORKS_ON": {
            Name:        "WORKS_ON",
            Description: "Employee contributes to a project",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Project"},
        },
        "MEMBER_OF": {
            Name:        "MEMBER_OF",
            Description: "Employee is a member of a team",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Team"},
        },
        "LEADS": {
            Name:        "LEADS",
            Description: "Employee leads a team or project",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Team", "Project"},
        },
        "TRANSFERRED_TO": {
            Name:        "TRANSFERRED_TO",
            Description: "Employee transferred to a different department",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Department"},
        },
    },
}
```

### File 2: `pkg/graph/presets/crm.go`

```go
package presets

import "github.com/vnp-memory/pkg/graph"

var CRMPreset = graph.OntologyRegistry{
    EntityTypes: map[string]graph.EntityTypeSchema{
        "Customer": {
            Name:        "Customer",
            Description: "A person, company, or organization that is or could be a customer",
            Properties: []graph.OntologyProperty{
                {Name: "company", Type: "string"},
                {Name: "email", Type: "string"},
                {Name: "tier", Type: "string"},
                {Name: "industry", Type: "string"},
            },
            Examples: []string{"Acme Corp", "John Smith at TechCo"},
        },
        "Deal": {
            Name:        "Deal",
            Description: "A sales opportunity, proposal, or contract",
            Properties: []graph.OntologyProperty{
                {Name: "value", Type: "number"},
                {Name: "stage", Type: "string"},
                {Name: "close_date", Type: "datetime"},
                {Name: "probability", Type: "number"},
            },
            Examples: []string{"Enterprise License Q4 2024", "Professional Services Renewal"},
        },
        "Product": {
            Name:        "Product",
            Description: "A product, service, or solution being sold",
            Examples:    []string{"Enterprise Plan", "Analytics Module", "Support Package"},
        },
        "SalesRep": {
            Name:        "SalesRep",
            Description: "A sales representative or account executive",
            Examples:    []string{"Alice Johnson (AE)", "Bob Smith (SDR)"},
        },
        "Company": {
            Name:        "Company",
            Description: "A business entity or organization",
            Examples:    []string{"Acme Corp", "TechStartup Inc"},
        },
    },
    EdgeTypes: map[string]graph.EdgeTypeSchema{
        "BOUGHT": {
            Name:        "BOUGHT",
            Description: "Customer purchased a product",
            SourceTypes: []string{"Customer"},
            TargetTypes: []string{"Product"},
        },
        "INTERESTED_IN": {
            Name:        "INTERESTED_IN",
            Description: "Customer expressed interest in a product or deal",
            SourceTypes: []string{"Customer"},
            TargetTypes: []string{"Product", "Deal"},
        },
        "ASSIGNED_TO": {
            Name:        "ASSIGNED_TO",
            Description: "Deal is assigned to a sales representative",
            SourceTypes: []string{"Deal"},
            TargetTypes: []string{"SalesRep"},
        },
        "WORKS_AT": {
            Name:        "WORKS_AT",
            Description: "Customer works at a company",
            SourceTypes: []string{"Customer"},
            TargetTypes: []string{"Company"},
        },
        "RENEWED": {
            Name:        "RENEWED",
            Description: "Customer renewed a product or service",
            SourceTypes: []string{"Customer"},
            TargetTypes: []string{"Product", "Deal"},
        },
        "CHURNED": {
            Name:        "CHURNED",
            Description: "Customer stopped using a product or cancelled",
            SourceTypes: []string{"Customer"},
            TargetTypes: []string{"Product"},
        },
    },
}
```

### File 3: `pkg/graph/presets/software.go`

```go
package presets

import "github.com/vnp-memory/pkg/graph"

var SoftwareProjectPreset = graph.OntologyRegistry{
    EntityTypes: map[string]graph.EntityTypeSchema{
        "Developer": {
            Name:        "Developer",
            Description: "A software developer, engineer, or contributor",
            Examples:    []string{"Alice (Backend Engineer)", "Bob (Frontend Dev)"},
        },
        "Service": {
            Name:        "Service",
            Description: "A microservice, application, or backend component",
            Examples:    []string{"auth-service", "payment-gateway", "notification-worker"},
        },
        "Repository": {
            Name:        "Repository",
            Description: "A code repository or codebase",
            Examples:    []string{"github.com/company/backend", "monorepo-frontend"},
        },
        "Feature": {
            Name:        "Feature",
            Description: "A software feature, user story, or capability",
            Examples:    []string{"OAuth 2.0 login", "Real-time notifications", "CSV export"},
        },
        "Bug": {
            Name:        "Bug",
            Description: "A software bug, defect, or issue",
            Properties:  []graph.OntologyProperty{{Name: "severity", Type: "string"}, {Name: "ticket_id", Type: "string"}},
            Examples:    []string{"Memory leak in auth-service", "Login button unresponsive"},
        },
        "Deployment": {
            Name:        "Deployment",
            Description: "A release or deployment event",
            Properties:  []graph.OntologyProperty{{Name: "version", Type: "string"}, {Name: "environment", Type: "string"}},
            Examples:    []string{"v2.3.1 to production", "Hotfix deploy to staging"},
        },
    },
    EdgeTypes: map[string]graph.EdgeTypeSchema{
        "OWNS": {
            Name:        "OWNS",
            Description: "Developer owns or is responsible for a service or repository",
            SourceTypes: []string{"Developer"},
            TargetTypes: []string{"Service", "Repository"},
        },
        "DEPENDS_ON": {
            Name:        "DEPENDS_ON",
            Description: "Service depends on another service",
            SourceTypes: []string{"Service"},
            TargetTypes: []string{"Service"},
        },
        "IMPLEMENTS": {
            Name:        "IMPLEMENTS",
            Description: "Developer implements a feature",
            SourceTypes: []string{"Developer"},
            TargetTypes: []string{"Feature"},
        },
        "FIXES": {
            Name:        "FIXES",
            Description: "Developer fixes a bug",
            SourceTypes: []string{"Developer"},
            TargetTypes: []string{"Bug"},
        },
        "REPORTED_IN": {
            Name:        "REPORTED_IN",
            Description: "Bug was found in a service",
            SourceTypes: []string{"Bug"},
            TargetTypes: []string{"Service"},
        },
        "DEPLOYED_BY": {
            Name:        "DEPLOYED_BY",
            Description: "Service was deployed by a developer",
            SourceTypes: []string{"Service"},
            TargetTypes: []string{"Developer"},
        },
    },
}
```

### File 4: `pkg/graph/presets/registry.go`

```go
package presets

import "github.com/vnp-memory/pkg/graph"

// PresetByName maps preset names to their ontology registries.
// Available presets: "hr", "crm", "software_project"
var PresetByName = map[string]*graph.OntologyRegistry{
    "hr":               &HRPreset,
    "crm":              &CRMPreset,
    "software_project": &SoftwareProjectPreset,
}

// ListPresets returns all available preset names
func ListPresets() []string {
    names := make([]string, 0, len(PresetByName))
    for k := range PresetByName { names = append(names, k) }
    return names
}
```

---

## Verification

```bash
cd /path/to/vnp-memory
go build ./pkg/graph/presets/...
go test ./pkg/graph/presets/... -v
```

**Test:**
```go
func TestPresets_AllPresetsLoad(t *testing.T) {
    for name, preset := range presets.PresetByName {
        if len(preset.EntityTypes) == 0 {
            t.Errorf("preset %s has no entity types", name)
        }
        if len(preset.EdgeTypes) == 0 {
            t.Errorf("preset %s has no edge types", name)
        }
        t.Logf("preset %s: %d entity types, %d edge types",
            name, len(preset.EntityTypes), len(preset.EdgeTypes))
    }
}
```
