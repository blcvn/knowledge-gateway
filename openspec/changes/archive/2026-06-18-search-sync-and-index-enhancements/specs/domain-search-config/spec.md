# domain-search-config

## Requirements

### Requirement: SearchProfileResolver interface decouples resolution from service code
The system SHALL define a `SearchProfileResolver` interface with a single method `Resolve(domainID, tenantID, appID string) (ResolvedSearchProfile, error)`. `workers.Runtime` and `search.Service` SHALL call this interface — never inline the resolution logic themselves.

The default implementation applies overrides in this precedence: app > tenant > domain > hardcoded defaults. A future implementation may consult a runtime config store, feature flags, or an external policy engine without changing any service code.

#### Scenario: Resolution precedence is enforced by the resolver, not by service code
- WHEN `workers.Runtime` needs the active search profile for a node
- THEN it SHALL call `SearchProfileResolver.Resolve(domainID, ownerTenantID, ownerAppID)`
- AND SHALL use the returned `ResolvedSearchProfile` without inspecting override maps directly

#### Scenario: Swap resolution algorithm at runtime
- WHEN the resolution algorithm is changed (e.g. to consult a feature flag store)
- THEN only the `SearchProfileResolver` implementation changes
- AND `workers.Runtime` and `search.Service` require no changes

### Requirement: SearchProfile per domain
The system SHALL store a `SearchProfile` on each ontology `Domain`. It SHALL be persisted as a nullable JSONB column `search_profile` on `kg_domains`.

`SearchProfile` references a `QueryStrategy` by key (`QueryStrategyRef`) rather than embedding strategy parameters directly. This separates lifecycle: strategies can be versioned and shared across domains independently of per-domain profiles.

#### Scenario: Domain created without a search profile
- WHEN a domain is created without a `search_profile`
- THEN `SearchProfileResolver.Resolve` SHALL return the system defaults:
  - `SemanticFields` = `[{id,1.0}, {node_type,1.0}, {domain_id,1.0}, {external_ref,1.0}, {status_value,1.0}, {*properties,1.0}]` — all built-in fields + all node.Properties at equal weight
  - `FTSLanguage` = `"simple"`
  - `QueryStrategyRef` = `"default"`

#### Scenario: SearchProfile present but SemanticFields is nil
- WHEN a domain has a `SearchProfile` whose `SemanticFields` field is nil (omitted from JSON)
- THEN `SearchProfileResolver.Resolve` SHALL treat it as unset and use the system default field list
- AND the behavior SHALL be identical to the no-profile case

#### Scenario: SearchProfile with empty SemanticFields is rejected on save
- WHEN an admin submits a `SearchProfile` with `SemanticFields = []` (explicitly empty array)
- THEN the ontology API SHALL return a 422 validation error: "semantic_fields must be nil (use defaults) or a non-empty list"
- AND the profile SHALL NOT be persisted

### Requirement: QueryStrategy as a versioned ontology object
The system SHALL store `QueryStrategy` records in the ontology layer (persisted in Postgres), keyed by a unique string. Built-in strategies (`"default"`, `"deep_traversal"`) SHALL be pre-seeded and immutable. Custom strategies are operator-defined.

```go
type QueryStrategy struct {
    Key      string         `json:"key"`
    Version  int            `json:"version"`
    MaxDepth int            `json:"max_depth"`
    Params   map[string]any `json:"params,omitempty"`
}
```

`Params` is an open map. New parameters never break existing strategies or callers; the compiler ignores keys it does not recognize.

**Pre-seeded strategies** (seeded at bootstrap, immutable, cannot be deleted or overwritten via API):

| Key | MaxDepth | Params |
|---|---|---|
| `"default"` | 5 | `direction="out"`, `depth_mode="fixed"`, `acl_predicate="any_hop"` |
| `"deep_traversal"` | 10 | `direction="out"`, `depth_mode="variable"`, `acl_predicate="start_only"` |

- `depth_mode="fixed"`: each hop is a distinct step; no variable-length path operator
- `depth_mode="variable"`: compiler emits a variable-length path up to `MaxDepth`
- `acl_predicate="any_hop"`: ACL WHERE clause applied at every hop (safe, higher cost)
- `acl_predicate="start_only"`: ACL WHERE clause applied only on the start node (faster, suitable for homogeneous ACL graphs)

#### Scenario: Unknown QueryStrategyRef at index/query time
- WHEN `SearchProfileResolver.Resolve` is called and `QueryStrategyRef` references a key not found in the ontology
- THEN the system SHALL log a WARNING with the missing key and domain ID
- AND SHALL fall back to the `"default"` strategy
- AND SHALL NOT return an error to the caller (index/query continues with degraded but safe behavior)

#### Scenario: Register a new custom query strategy
- WHEN an admin creates a `QueryStrategy` record with key `"finance_deep"` and `MaxDepth=8`
- THEN any domain that sets `QueryStrategyRef="finance_deep"` in its profile SHALL use that strategy on the next `Resolve` call
- AND no code changes SHALL be required in `read.QueryTemplateCompiler`

#### Scenario: Update a shared strategy
- WHEN the `"finance_deep"` strategy's `MaxDepth` is updated from 8 to 10
- THEN all domains referencing `"finance_deep"` SHALL use the new depth on subsequent queries
- AND their individual `SearchProfile` records SHALL not need to change

### Requirement: Semantic index field configuration
The system SHALL allow per-domain (and per-tenant/app override) configuration of which node properties are included in the embedding text, in what order and weight.

#### Scenario: Custom field weights applied at index time
- WHEN `workers.Runtime` builds the embedding text for a node
- THEN it SHALL call `SearchProfileResolver.Resolve` to get the effective `SemanticFields`
- AND SHALL build the text by including each field value, repeated proportionally to its weight, with its optional prefix label

#### Scenario: New index configuration applied going forward
- WHEN a domain's `SearchProfile.SemanticFields` is updated
- THEN existing vector documents are NOT immediately re-indexed
- AND the next reconciliation run or a triggered re-index job SHALL rebuild affected vectors using the new field configuration

### Requirement: FTS language per domain
The system SHALL allow per-domain and per-override configuration of the FTS language string (`FTSLanguage`). Each `FTSAdapter` maps this string to its own analyzer config.

#### Scenario: Vietnamese domain uses "simple" analyzer
- WHEN a domain's `FTSLanguage` is set to `"simple"`
- THEN `PgFTSAdapter` SHALL use `to_tsvector('simple', ...)` for that domain's nodes

### Requirement: Per-tenant and per-app overrides
`SearchProfile` SHALL support `TenantOverrides` (keyed by `tenant_id`) and `AppOverrides` (keyed by `"tenant_id:app_id"`). Each override carries a `SearchProfileOverride` with the same shape as the domain baseline but all fields optional.

#### Scenario: App-level override takes precedence
- WHEN an actor with a matching `tenant_id:app_id` calls `SearchProfileResolver.Resolve`
- THEN fields present in the app-level override SHALL replace the corresponding domain-baseline and tenant-level fields
- AND fields absent from the app-level override SHALL fall through to the tenant override or domain baseline

#### Scenario: Locked fields cannot be overridden
- WHEN a `SearchProfile` marks `SemanticFields` as `locked: true` (future extension via `Params`)
- THEN `SearchProfileResolver.Resolve` SHALL ignore tenant/app override values for that field
- AND the domain baseline value SHALL be used regardless of the override

### Requirement: SearchProfile and QueryStrategy are managed via the ontology API
The system SHALL expose:
- `PUT /ontology/domains/{domain_id}/search-profile` — set or replace the domain search profile
- `GET /ontology/domains/{domain_id}/search-profile` — read the effective resolved profile for the requesting actor
- `POST /ontology/query-strategies` — register a new custom `QueryStrategy`
- `PUT /ontology/query-strategies/{key}` — update a strategy (version is incremented automatically)
- `GET /ontology/query-strategies` — list all strategies

All write endpoints SHALL validate: field names exist in the domain's node type schemas; weights are in [0.1, 10.0]; `QueryStrategyRef` references an existing strategy key; `FTSLanguage` is a non-empty string.
