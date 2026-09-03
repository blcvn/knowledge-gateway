package domain

import "time"

// ──── T08: Audit Log Types ─────────────────────────────────────

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	UserID    string         `json:"user_id"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Details   map[string]any `json:"details,omitempty"`
	IP        string         `json:"ip,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// AuditFilter holds search filters for audit log queries.
type AuditFilter struct {
	TenantID string    `json:"tenant_id,omitempty"`
	UserID   string    `json:"user_id,omitempty"`
	Action   string    `json:"action,omitempty"`
	Resource string    `json:"resource,omitempty"`
	From     time.Time `json:"from,omitempty"`
	To       time.Time `json:"to,omitempty"`
	Limit    int       `json:"limit,omitempty"`
	Offset   int       `json:"offset,omitempty"`
}

// ──── T09: Policy Types ────────────────────────────────────────

// Policy represents a governance policy (OPA Rego).
type Policy struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type"` // "retention", "access", "gdpr", "rate_limit"
	RegoBody    string         `json:"rego_body"`
	Enabled     bool           `json:"enabled"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ──── T10: Pipeline Types ──────────────────────────────────────

// EngineStatus represents the health status of a single engine service.
type EngineStatus struct {
	Service string    `json:"service"`
	Status  string    `json:"status"` // "healthy", "unhealthy", "degraded"
	CheckAt time.Time `json:"check_at"`
}

// ──── T11: Infrastructure Types ────────────────────────────────

// InfraTopology represents the full service topology.
type InfraTopology struct {
	TotalServices   int           `json:"total_services"`
	HealthyServices int           `json:"healthy_services"`
	Nodes           []ServiceNode `json:"nodes"`
	Timestamp       time.Time     `json:"timestamp"`
}

// ServiceNode represents a single service in the topology graph.
type ServiceNode struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "healthy", "unhealthy"
}

// ──── T12: Search Types ────────────────────────────────────────

// UnifiedSearchResult holds results from a fan-out cross-engine search.
type UnifiedSearchResult struct {
	Results   []EngineSearchResult `json:"results"`
	LatencyMs int64                `json:"latency_ms"`
	Engines   int                  `json:"engines"`
}

// EngineSearchResult holds search results from a single engine.
type EngineSearchResult struct {
	Engine string `json:"engine"`
	Data   []byte `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ──── T13: GDPR Forget Types ───────────────────────────────────

// ForgetPreview describes the impact of a GDPR forget request.
type ForgetPreview struct {
	UserID  string         `json:"user_id"`
	Targets []ForgetTarget `json:"targets"`
	Total   int            `json:"total"`
	DryRun  bool           `json:"dry_run"`
}

// ForgetTarget represents a service that holds user data to be deleted.
type ForgetTarget struct {
	Service string `json:"service"`
	Status  string `json:"status"` // "reachable", "unreachable"
}

// ForgetResult holds the outcome of a GDPR cascading forget operation.
type ForgetResult struct {
	UserID    string   `json:"user_id"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
	LatencyMs int64    `json:"latency_ms"`
}
