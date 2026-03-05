package analytics

import (
	"context"
	"errors"
	"sort"
	"time"
)

const traceabilityQuery = `
MATCH p=(s)-[*1..$max_hops]->(t)
WHERE s.app_id = $app_id
  AND s.tenant_id = $tenant_id
  AND t.app_id = $app_id
  AND t.tenant_id = $tenant_id
  AND any(lbl IN labels(s) WHERE lbl IN $source_types)
  AND any(lbl IN labels(t) WHERE lbl IN $target_types)
RETURN s.id AS source_id,
       coalesce(s.name, s.id) AS source_name,
       coalesce(head(labels(s)), 'Entity') AS source_type,
       t.id AS target_id,
       coalesce(t.name, t.id) AS target_name,
       coalesce(head(labels(t)), 'Entity') AS target_type,
       length(p) AS hops,
       [rel IN relationships(p) | type(rel)] AS path
ORDER BY source_id ASC, hops ASC
LIMIT $limit
`

func (e *Engine) TraceabilityMatrix(ctx context.Context, namespace string, sourceTypes, targetTypes []string, maxHops int) (*TraceabilityMatrix, error) {
	if e == nil || e.query == nil {
		return nil, errors.New("analytics engine is not configured")
	}

	if len(sourceTypes) == 0 || len(targetTypes) == 0 {
		return &TraceabilityMatrix{Matrix: []TraceabilityRow{}}, nil
	}
	maxHops = normalizeMaxHops(maxHops)

	params := map[string]any{
		"source_types": sourceTypes,
		"target_types": targetTypes,
		"max_hops":     maxHops,
	}
	var cached TraceabilityMatrix
	if e.cache != nil {
		if ok, err := e.cache.Get(ctx, "traceability", namespace, params, &cached); err == nil && ok {
			return &cached, nil
		}
	}

	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := e.query.ExecuteQuery(ctx, traceabilityQuery, map[string]any{
		"app_id":       appID,
		"tenant_id":    tenantID,
		"source_types": sourceTypes,
		"target_types": targetTypes,
		"max_hops":     maxHops,
		"limit":        2000,
	})
	if err != nil {
		return nil, err
	}

	matrixBySource := make(map[string]*TraceabilityRow)
	targetSeen := make(map[string]struct{})

	for _, row := range toRows(result) {
		sourceID := asString(row["source_id"])
		if sourceID == "" {
			continue
		}
		entry, ok := matrixBySource[sourceID]
		if !ok {
			entry = &TraceabilityRow{
				SourceID:   sourceID,
				SourceName: asString(row["source_name"]),
				SourceType: asString(row["source_type"]),
				Targets:    make([]TraceabilityTarget, 0),
			}
			matrixBySource[sourceID] = entry
		}

		targetID := asString(row["target_id"])
		path := asStringSlice(row["path"])
		entry.Targets = append(entry.Targets, TraceabilityTarget{
			EntityID: targetID,
			Name:     asString(row["target_name"]),
			Type:     asString(row["target_type"]),
			Hops:     asInt(row["hops"]),
			Path:     path,
		})
		if targetID != "" {
			targetSeen[targetID] = struct{}{}
		}
	}

	sourceIDs := make([]string, 0, len(matrixBySource))
	for sourceID := range matrixBySource {
		sourceIDs = append(sourceIDs, sourceID)
	}
	sort.Strings(sourceIDs)

	matrix := make([]TraceabilityRow, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		row := matrixBySource[sourceID]
		sort.Slice(row.Targets, func(i, j int) bool {
			if row.Targets[i].Hops == row.Targets[j].Hops {
				return row.Targets[i].EntityID < row.Targets[j].EntityID
			}
			return row.Targets[i].Hops < row.Targets[j].Hops
		})
		matrix = append(matrix, *row)
	}

	out := &TraceabilityMatrix{
		Matrix:            matrix,
		TotalSources:      len(matrix),
		TotalTargets:      len(targetSeen),
		ComputeDurationMs: time.Since(start).Milliseconds(),
	}

	if e.cache != nil {
		_ = e.cache.Set(ctx, "traceability", namespace, params, out)
	}
	return out, nil
}
