// Package extractor implements TabularFKExtractor for structured FK data ingestion.
// TASK-CE-008: Advanced Loaders — Tabular FK (SOL-004 §3)
//
// Converts a JSON row array + FK schema → DataPoints + Relations without LLM.
// This is the "TABULAR_FK" content type handler.
package extractor

import (
	"context"
	"fmt"

	"vnp-memory/services/cognee-ingestion/internal/domain"
)

// TabularFKExtractor converts tabular data (JSON row arrays) with FK relations
// into DataPoint + DataPointRelation structures for zero-LLM ingestion.
type TabularFKExtractor struct{}

// TabularDataInput holds the rows and schema for tabular extraction.
type TabularDataInput struct {
	Rows   []map[string]any `json:"rows"`
	Schema TabularSchema    `json:"schema"`
}

// TabularSchema defines how to interpret table rows as DataPoints.
type TabularSchema struct {
	IDField     string       `json:"id_field"`    // field that uniquely identifies each row
	TypeName    string       `json:"type_name"`   // DataPoint type e.g. "Employee", "Paper"
	FKRelations []FKRelation `json:"fk_relations"` // FK edges to other DataPoints
}

// FKRelation describes a foreign-key edge between DataPoints.
type FKRelation struct {
	FromField string `json:"from_field"` // e.g. "dept_id" (field in this table)
	ToDataset string `json:"to_dataset"` // target namespace e.g. "departments"
	EdgeLabel string `json:"edge_label"` // Neo4j relationship type e.g. "works_in"
}

// ExtractDataPoints converts tabular rows to DataPoints with FK relations.
// Uses DeterministicUUID for stable, idempotent IDs — same row re-ingested → same ID.
func (e *TabularFKExtractor) ExtractDataPoints(ctx context.Context, input TabularDataInput) ([]domain.DataPoint, error) {
	if input.Schema.TypeName == "" {
		return nil, fmt.Errorf("tabular_fk: schema.type_name is required")
	}
	if input.Schema.IDField == "" {
		return nil, fmt.Errorf("tabular_fk: schema.id_field is required")
	}

	dps := make([]domain.DataPoint, 0, len(input.Rows))

	for _, row := range input.Rows {
		idVal, ok := row[input.Schema.IDField]
		if !ok {
			continue // skip rows with missing ID field
		}
		idStr := fmt.Sprint(idVal)

		// Deterministic UUID: same type+id → same UUID across re-ingestions
		dpID := domain.DeterministicUUID(input.Schema.TypeName, idStr)

		// Copy all fields
		fields := make(map[string]any, len(row))
		for k, v := range row {
			fields[k] = v
		}

		// Build FK relations: each FK field → DataPointRelation edge
		var relations []domain.DataPointRelation
		for _, fk := range input.Schema.FKRelations {
			if fkVal, ok := row[fk.FromField]; ok {
				targetID := domain.DeterministicUUID(fk.ToDataset, fmt.Sprint(fkVal))
				relations = append(relations, domain.DataPointRelation{
					TargetID: targetID,
					Label:    fk.EdgeLabel,
					Weight:   1.0,
				})
			}
		}

		dp := domain.DataPoint{
			ID:          dpID,
			Type:        input.Schema.TypeName,
			Fields:      fields,
			Relations:   relations,
			IndexFields: detectStringFields(fields),
		}
		dps = append(dps, dp)
	}
	return dps, nil
}

// detectStringFields returns field names whose values are non-trivial strings.
// These are good candidates for semantic embedding.
func detectStringFields(fields map[string]any) []string {
	var result []string
	for k, v := range fields {
		if s, ok := v.(string); ok && len(s) > 10 {
			result = append(result, k)
		}
	}
	return result
}
