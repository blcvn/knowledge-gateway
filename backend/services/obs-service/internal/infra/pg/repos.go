// Package pg implements PostgreSQL repos for obs-service.
//
// Covers: ErrorRepository, CostRepository, MetricsRepository
// (MERGE-P3-T2)
package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domobs "vnp-memory/services/obs-service/internal/domain/observability"
)

// ── ErrorRepository ────────────────────────────────────────────────────────

// ErrorRepo implements port.ErrorRepository.
type ErrorRepo struct{ pool *pgxpool.Pool }

// NewErrorRepo creates an ErrorRepo.
func NewErrorRepo(pool *pgxpool.Pool) *ErrorRepo { return &ErrorRepo{pool: pool} }

func (r *ErrorRepo) List(ctx context.Context, filter domobs.ErrorFilter) ([]*domobs.ErrorEntry, int, error) {
	query := `SELECT id, service, type, message, stack, count, first_seen, last_seen
		FROM obs_errors`
	args := []any{}
	argN := 1
	if filter.Service != "" {
		query += fmt.Sprintf(" WHERE service=$%d", argN)
		args = append(args, filter.Service)
		argN++
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" ORDER BY last_seen DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var entries []*domobs.ErrorEntry
	for rows.Next() {
		e := &domobs.ErrorEntry{}
		if err := rows.Scan(&e.ID, &e.Service, &e.Type, &e.Message, &e.Stack, &e.Count, &e.FirstSeen, &e.LastSeen); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, len(entries), nil
}

func (r *ErrorRepo) Record(ctx context.Context, entry *domobs.ErrorEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO obs_errors (id, service, type, message, stack, count, first_seen, last_seen)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (service, type, message) DO UPDATE
		 SET count = obs_errors.count + 1, last_seen = $8`,
		entry.ID, entry.Service, entry.Type, entry.Message, entry.Stack,
		entry.Count, entry.FirstSeen, entry.LastSeen,
	)
	return err
}

// ── CostRepository ─────────────────────────────────────────────────────────

// CostRepo implements port.CostRepository.
type CostRepo struct{ pool *pgxpool.Pool }

// NewCostRepo creates a CostRepo.
func NewCostRepo(pool *pgxpool.Pool) *CostRepo { return &CostRepo{pool: pool} }

func (r *CostRepo) GetByPeriod(ctx context.Context, period string) ([]*domobs.CostEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT service, period, llm_tokens, embed_tokens, storage_mb, est_cost_usd, recorded_at
		 FROM obs_costs WHERE period LIKE $1 ORDER BY recorded_at DESC LIMIT 100`,
		"%:"+period,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*domobs.CostEntry
	for rows.Next() {
		e := &domobs.CostEntry{}
		if err := rows.Scan(&e.Service, &e.Period, &e.LLMTokens, &e.EmbedTokens, &e.StorageMB, &e.EstCostUSD, &e.Timestamp); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *CostRepo) Record(ctx context.Context, entry *domobs.CostEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO obs_costs (id, service, period, llm_tokens, embed_tokens, storage_mb, est_cost_usd, recorded_at)
		 VALUES (gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (service, period) DO UPDATE
		 SET llm_tokens = obs_costs.llm_tokens + $3,
		     embed_tokens = obs_costs.embed_tokens + $4,
		     storage_mb = obs_costs.storage_mb + $5,
		     est_cost_usd = obs_costs.est_cost_usd + $6,
		     recorded_at = $7`,
		entry.Service, entry.Period, entry.LLMTokens, entry.EmbedTokens,
		entry.StorageMB, entry.EstCostUSD, time.Now(),
	)
	return err
}

// ── MetricsRepository ──────────────────────────────────────────────────────

// MetricsRepo implements port.MetricsRepository.
type MetricsRepo struct{ pool *pgxpool.Pool }

// NewMetricsRepo creates a MetricsRepo.
func NewMetricsRepo(pool *pgxpool.Pool) *MetricsRepo { return &MetricsRepo{pool: pool} }

func (r *MetricsRepo) Record(ctx context.Context, points []*domobs.MetricPoint) error {
	for _, p := range points {
		labelsJSON, _ := json.Marshal(p.Labels)
		_, err := r.pool.Exec(ctx,
			`INSERT INTO obs_metrics (id, name, value, labels, recorded_at)
			 VALUES (gen_random_uuid(),$1,$2,$3,$4)`,
			p.Name, p.Value, labelsJSON, p.Timestamp,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MetricsRepo) GetSummary(ctx context.Context) (*domobs.MetricSummary, error) {
	// Aggregate last hour of metrics
	summary := &domobs.MetricSummary{Timestamp: time.Now()}

	// Total requests
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(value), 0) FROM obs_metrics
		 WHERE name='http_requests_total' AND recorded_at > NOW() - INTERVAL '1 hour'`,
	).Scan(&summary.TotalRequests)

	// P50/P95/P99 latency from named metrics
	latencies := map[string]*float64{
		"http_request_duration_ms_p50": &summary.P50LatencyMs,
		"http_request_duration_ms_p95": &summary.P95LatencyMs,
		"http_request_duration_ms_p99": &summary.P99LatencyMs,
	}
	for metricName, target := range latencies {
		_ = r.pool.QueryRow(ctx,
			`SELECT COALESCE(AVG(value), 0) FROM obs_metrics
			 WHERE name=$1 AND recorded_at > NOW() - INTERVAL '1 hour'`,
			metricName,
		).Scan(target)
	}

	// Per-service metrics
	rows, err := r.pool.Query(ctx,
		`SELECT labels->>'service', SUM(value) FROM obs_metrics
		 WHERE name='http_requests_total' AND recorded_at > NOW() - INTERVAL '1 hour'
		 GROUP BY labels->>'service'`,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			svc := &domobs.ServiceMetric{Healthy: true}
			if err := rows.Scan(&svc.Name, &svc.RequestCount); err == nil {
				summary.Services = append(summary.Services, *svc)
			}
		}
	}

	if summary.TotalRequests > 0 {
		var totalErrors int64
		_ = r.pool.QueryRow(ctx,
			`SELECT COALESCE(SUM(value), 0) FROM obs_metrics
			 WHERE name='http_errors_total' AND recorded_at > NOW() - INTERVAL '1 hour'`,
		).Scan(&totalErrors)
		summary.ErrorRate = float64(totalErrors) / float64(summary.TotalRequests)
	}

	return summary, nil
}

// ── Noop impls for optional backends ──────────────────────────────────────

// NoopScraper is a no-op MetricsScraper (when Prometheus is not configured).
type NoopScraper struct{}

func (s *NoopScraper) ScrapeAll(_ context.Context) ([]*domobs.MetricPoint, error) {
	return nil, fmt.Errorf("prometheus: not configured")
}

// NoopTraceClient is a no-op TraceClient (when Jaeger is not configured).
type NoopTraceClient struct{}

func (c *NoopTraceClient) ListTraces(_ context.Context, _ domobs.TraceFilter) ([]*domobs.Trace, int, error) {
	return []*domobs.Trace{}, 0, nil
}
func (c *NoopTraceClient) GetTrace(_ context.Context, traceID string) (*domobs.Trace, error) {
	return nil, fmt.Errorf("trace %s: not found (jaeger not configured)", traceID)
}


