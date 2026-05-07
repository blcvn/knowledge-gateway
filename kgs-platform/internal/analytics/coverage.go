package analytics

import (
	"context"
	"errors"
	"sort"
	"time"
)

const coverageQuery = `
MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
WHERE $domain = '' OR coalesce(n.domain, '') = $domain OR $domain IN coalesce(n.domains, [])
WITH n, coalesce(head(labels(n)), 'Entity') AS entity_type
OPTIONAL MATCH (n)-[r]->()
WITH entity_type, n, count(r) AS outgoing_edges
WITH entity_type,
     count(n) AS total_entities,
     sum(CASE WHEN outgoing_edges > 0 THEN 1 ELSE 0 END) AS covered_entities
RETURN entity_type, total_entities, covered_entities
ORDER BY entity_type ASC
`

func (e *Engine) CoverageReport(ctx context.Context, namespace, domain string) (*CoverageReport, error) {
	if e == nil || e.query == nil {
		return nil, errors.New("analytics engine is not configured")
	}

	params := map[string]any{"domain": domain}
	var cached CoverageReport
	if e.cache != nil {
		if ok, err := e.cache.Get(ctx, "coverage", namespace, params, &cached); err == nil && ok {
			return &cached, nil
		}
	}

	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}

	result, err := e.query.ExecuteQuery(ctx, coverageQuery, map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"domain":    domain,
	})
	if err != nil {
		return nil, err
	}

	report := &CoverageReport{
		Domain:      domain,
		GeneratedAt: time.Now().UTC(),
		ByType:      make([]CoverageByType, 0),
	}

	for _, row := range toRows(result) {
		stat := CoverageByType{
			EntityType:      asString(row["entity_type"]),
			TotalEntities:   asInt(row["total_entities"]),
			CoveredEntities: asInt(row["covered_entities"]),
		}
		if stat.TotalEntities > 0 {
			stat.CoveragePercent = float64(stat.CoveredEntities) * 100 / float64(stat.TotalEntities)
		}
		report.TotalEntities += stat.TotalEntities
		report.CoveredEntities += stat.CoveredEntities
		if stat.CoveredEntities < stat.TotalEntities {
			report.UncoveredTypes = append(report.UncoveredTypes, stat.EntityType)
		}
		report.ByType = append(report.ByType, stat)
	}

	if report.TotalEntities > 0 {
		report.CoveragePercent = float64(report.CoveredEntities) * 100 / float64(report.TotalEntities)
	}
	sort.Strings(report.UncoveredTypes)

	if e.cache != nil {
		_ = e.cache.Set(ctx, "coverage", namespace, params, report)
	}
	return report, nil
}
