// Package usecase — Console-specific usecases for SOL-002 (T08-T13).
// These usecases implement gateway-level orchestration logic for the Console UI,
// aggregating data from multiple downstream services.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// ──── T08: Audit Log Service ───────────────────────────────────

// AuditUseCase provides audit log write and query operations.
type AuditUseCase struct {
	store     port.AuditStore
	publisher port.EventPublisher
	logger    *slog.Logger
}

// NewAuditUseCase creates a new AuditUseCase.
func NewAuditUseCase(store port.AuditStore, publisher port.EventPublisher, logger *slog.Logger) *AuditUseCase {
	return &AuditUseCase{store: store, publisher: publisher, logger: logger}
}

// Log records an audit event.
func (uc *AuditUseCase) Log(ctx context.Context, entry *domain.AuditEntry) error {
	entry.Timestamp = time.Now()
	if err := uc.store.Insert(ctx, entry); err != nil {
		uc.logger.Error("audit log insert failed", "error", err, "action", entry.Action)
		return err
	}

	// Publish audit event for downstream consumers
	if err := uc.publisher.Publish(ctx, "audit.created", entry); err != nil {
		uc.logger.Warn("audit event publish failed", "error", err)
	}
	return nil
}

// Search queries audit logs with filters.
func (uc *AuditUseCase) Search(ctx context.Context, filter *domain.AuditFilter) ([]*domain.AuditEntry, int, error) {
	return uc.store.Search(ctx, filter)
}

// ──── T09: OPA Policy CRUD ─────────────────────────────────────

// PolicyUseCase manages governance policies (OPA Rego).
type PolicyUseCase struct {
	store     port.PolicyStore
	publisher port.EventPublisher
	logger    *slog.Logger
}

// NewPolicyUseCase creates a new PolicyUseCase.
func NewPolicyUseCase(store port.PolicyStore, publisher port.EventPublisher, logger *slog.Logger) *PolicyUseCase {
	return &PolicyUseCase{store: store, publisher: publisher, logger: logger}
}

// List returns all policies with optional filters.
func (uc *PolicyUseCase) List(ctx context.Context, tenantID string) ([]*domain.Policy, error) {
	return uc.store.List(ctx, tenantID)
}

// Get returns a policy by ID.
func (uc *PolicyUseCase) Get(ctx context.Context, id string) (*domain.Policy, error) {
	return uc.store.Get(ctx, id)
}

// Create creates a new policy.
func (uc *PolicyUseCase) Create(ctx context.Context, p *domain.Policy) error {
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if err := uc.store.Create(ctx, p); err != nil {
		return err
	}
	uc.publisher.Publish(ctx, "policy.created", p)
	uc.logger.Info("policy created", "id", p.ID, "name", p.Name)
	return nil
}

// Update updates a policy.
func (uc *PolicyUseCase) Update(ctx context.Context, p *domain.Policy) error {
	p.UpdatedAt = time.Now()
	if err := uc.store.Update(ctx, p); err != nil {
		return err
	}
	uc.publisher.Publish(ctx, "policy.updated", p)
	return nil
}

// ──── T10: Pipeline Status Aggregation ─────────────────────────

// PipelineUseCase aggregates pipeline status from multiple engine services.
type PipelineUseCase struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewPipelineUseCase creates a new PipelineUseCase.
func NewPipelineUseCase(registry port.ServiceRegistry, logger *slog.Logger) *PipelineUseCase {
	return &PipelineUseCase{registry: registry, logger: logger}
}

// engineServices lists the engine groups for pipeline status.
var engineServices = []string{
	"cognee-ingestion", "cognee-cognify", "cognee-search",
	"graphiti-ingestion", "graphiti-search", "graphiti-store",
	"memobase-ingestion", "memobase-engine", "memobase-context",
	"ov-fs", "ov-search", "ov-session",
	"zep-memory", "zep-search", "zep-graph",
	"sm-document", "sm-memory", "sm-search",
}

// AggregateStatus returns pipeline status for all engines via fan-out health checks.
func (uc *PipelineUseCase) AggregateStatus(ctx context.Context) ([]domain.EngineStatus, error) {
	type result struct {
		service string
		healthy bool
		err     error
	}

	ch := make(chan result, len(engineServices))
	for _, svc := range engineServices {
		go func(s string) {
			healthy, err := uc.registry.HealthCheck(s)
			ch <- result{service: s, healthy: healthy, err: err}
		}(svc)
	}

	statuses := make([]domain.EngineStatus, 0, len(engineServices))
	for range engineServices {
		r := <-ch
		status := "healthy"
		if !r.healthy || r.err != nil {
			status = "unhealthy"
		}
		statuses = append(statuses, domain.EngineStatus{
			Service: r.service,
			Status:  status,
			CheckAt: time.Now(),
		})
	}

	return statuses, nil
}

// ──── T11: Infrastructure Health Probes ────────────────────────

// InfraUseCase probes infrastructure dependencies.
type InfraUseCase struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewInfraUseCase creates a new InfraUseCase.
func NewInfraUseCase(registry port.ServiceRegistry, logger *slog.Logger) *InfraUseCase {
	return &InfraUseCase{registry: registry, logger: logger}
}

// allServices is the comprehensive list for topology.
var allServices = []string{
	"cognee-ingestion", "cognee-cognify", "cognee-search",
	"graphiti-ingestion", "graphiti-search", "graphiti-knowledge", "graphiti-store",
	"memobase-ingestion", "memobase-engine", "memobase-context",
	"vnp-event", "vnp-search-hub", "vnp-admin",
	"vnp-dashboard", "vnp-pipelines", "vnp-infra", "vnp-observability",
	"ov-fs", "ov-search", "ov-session", "ov-resource", "ov-crypto", "ov-admin",
	"zep-user", "zep-thread", "zep-memory", "zep-graph", "zep-search", "zep-admin", "zep-core",
	"sm-document", "sm-memory", "sm-search", "sm-profile", "sm-connector",
	"sm-mcp", "sm-auth", "sm-analytics", "sm-project", "sm-engine",
}

// Topology returns the service topology with connection state.
func (uc *InfraUseCase) Topology(ctx context.Context) (*domain.InfraTopology, error) {
	type probeResult struct {
		service string
		healthy bool
	}

	ch := make(chan probeResult, len(allServices))
	for _, svc := range allServices {
		go func(s string) {
			healthy, _ := uc.registry.HealthCheck(s)
			ch <- probeResult{service: s, healthy: healthy}
		}(svc)
	}

	nodes := make([]domain.ServiceNode, 0, len(allServices))
	healthyCount := 0
	for range allServices {
		r := <-ch
		status := "healthy"
		if !r.healthy {
			status = "unhealthy"
		} else {
			healthyCount++
		}
		nodes = append(nodes, domain.ServiceNode{
			Name:   r.service,
			Status: status,
		})
	}

	return &domain.InfraTopology{
		TotalServices:   len(allServices),
		HealthyServices: healthyCount,
		Nodes:           nodes,
		Timestamp:       time.Now(),
	}, nil
}

// ──── T12: Unified Search Orchestration ────────────────────────

// SearchUseCase orchestrates cross-engine search with fan-out/fan-in.
type SearchUseCase struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewSearchUseCase creates a new SearchUseCase.
func NewSearchUseCase(registry port.ServiceRegistry, logger *slog.Logger) *SearchUseCase {
	return &SearchUseCase{registry: registry, logger: logger}
}

// searchEngines are the services capable of search.
var searchEngines = []string{
	"cognee-search", "graphiti-search", "vnp-search-hub",
	"ov-search", "zep-search", "sm-search",
}

// FanOutSearch executes a search across all engines concurrently.
func (uc *SearchUseCase) FanOutSearch(ctx context.Context, query []byte) (*domain.UnifiedSearchResult, error) {
	start := time.Now()

	type engineResult struct {
		engine string
		data   []byte
		err    error
	}

	var wg sync.WaitGroup
	ch := make(chan engineResult, len(searchEngines))

	for _, engine := range searchEngines {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			target, err := uc.registry.Resolve(svc)
			if err != nil {
				ch <- engineResult{engine: svc, err: err}
				return
			}
			resp, err := uc.registry.Forward(ctx, target, query)
			ch <- engineResult{engine: svc, data: resp, err: err}
		}(engine)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(ch)
	}()

	results := make([]domain.EngineSearchResult, 0, len(searchEngines))
	for r := range ch {
		esr := domain.EngineSearchResult{
			Engine: r.engine,
		}
		if r.err != nil {
			esr.Error = r.err.Error()
			uc.logger.Warn("search engine error", "engine", r.engine, "error", r.err)
		} else {
			esr.Data = r.data
		}
		results = append(results, esr)
	}

	return &domain.UnifiedSearchResult{
		Results:   results,
		LatencyMs: time.Since(start).Milliseconds(),
		Engines:   len(searchEngines),
	}, nil
}

// ──── T13: GDPR Cascading Forget ───────────────────────────────

// ForgetUseCase orchestrates GDPR right-to-erasure across all engines.
type ForgetUseCase struct {
	registry  port.ServiceRegistry
	publisher port.EventPublisher
	audit     *AuditUseCase
	logger    *slog.Logger
}

// NewForgetUseCase creates a new ForgetUseCase.
func NewForgetUseCase(registry port.ServiceRegistry, publisher port.EventPublisher, audit *AuditUseCase, logger *slog.Logger) *ForgetUseCase {
	return &ForgetUseCase{
		registry:  registry,
		publisher: publisher,
		audit:     audit,
		logger:    logger,
	}
}

// forgetTargets are all services that may hold user data.
var forgetTargets = []string{
	"cognee-ingestion", "graphiti-store",
	"memobase-ingestion", "memobase-context",
	"ov-session", "ov-resource",
	"zep-user", "zep-memory",
	"sm-memory", "sm-profile", "sm-document",
}

// Preview returns the set of services and estimated record counts affected by a forget.
func (uc *ForgetUseCase) Preview(ctx context.Context, userID string) (*domain.ForgetPreview, error) {
	affected := make([]domain.ForgetTarget, 0, len(forgetTargets))
	for _, svc := range forgetTargets {
		healthy, _ := uc.registry.HealthCheck(svc)
		status := "reachable"
		if !healthy {
			status = "unreachable"
		}
		affected = append(affected, domain.ForgetTarget{
			Service: svc,
			Status:  status,
		})
	}

	return &domain.ForgetPreview{
		UserID:   userID,
		Targets:  affected,
		Total:    len(forgetTargets),
		DryRun:   true,
	}, nil
}

// Execute performs cascading forget across all engines.
func (uc *ForgetUseCase) Execute(ctx context.Context, userID, requestedBy string) (*domain.ForgetResult, error) {
	start := time.Now()

	req, _ := json.Marshal(map[string]string{
		"action":  "forget",
		"user_id": userID,
	})

	type deleteResult struct {
		service string
		success bool
		err     error
	}

	ch := make(chan deleteResult, len(forgetTargets))
	for _, svc := range forgetTargets {
		go func(s string) {
			target, err := uc.registry.Resolve(s)
			if err != nil {
				ch <- deleteResult{service: s, err: err}
				return
			}
			_, err = uc.registry.Forward(ctx, target, req)
			ch <- deleteResult{service: s, success: err == nil, err: err}
		}(svc)
	}

	succeeded := 0
	failed := 0
	errors := make([]string, 0)
	for range forgetTargets {
		r := <-ch
		if r.success {
			succeeded++
		} else {
			failed++
			errMsg := fmt.Sprintf("%s: %v", r.service, r.err)
			errors = append(errors, errMsg)
			uc.logger.Error("forget failed for service", "service", r.service, "error", r.err)
		}
	}

	result := &domain.ForgetResult{
		UserID:    userID,
		Succeeded: succeeded,
		Failed:    failed,
		Errors:    errors,
		LatencyMs: time.Since(start).Milliseconds(),
	}

	// Record audit trail
	if uc.audit != nil {
		uc.audit.Log(ctx, &domain.AuditEntry{
			TenantID:  "", // Will be extracted from context
			UserID:    requestedBy,
			Action:    "gdpr.forget",
			Resource:  "user:" + userID,
			Details:   map[string]any{"succeeded": succeeded, "failed": failed},
		})
	}

	// Publish event
	uc.publisher.Publish(ctx, "gdpr.forget.completed", result)

	return result, nil
}
