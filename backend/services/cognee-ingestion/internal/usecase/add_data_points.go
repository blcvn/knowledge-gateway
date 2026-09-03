// Package usecase implements AddDataPoints — structured ingestion with zero LLM calls.
// TASK-CE-007: DataPoint Schema (SOL-003 §2.1–§2.8)
//
// Design: User provides explicit schema → system maps directly to Neo4j + Qdrant.
// Key constraint: NO LLM call in this flow (pure embedding only for index_fields).
package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"vnp-memory/services/cognee-ingestion/internal/domain"
)

// ─── Request / Response ───────────────────────────────────────────────────────

// AddDataPointsRequest is the input for AddDataPointsUseCase.Execute.
type AddDataPointsRequest struct {
	DatasetID  uuid.UUID
	TenantID   string
	DataPoints []domain.DataPoint
	NodeSets   []string // Optional NodeSet tags (CR-002 integration)
}

// AddDataPointsResult is the output of AddDataPointsUseCase.Execute.
type AddDataPointsResult struct {
	Upserted int
	Created  int
	Updated  int
}

// ─── Ports ────────────────────────────────────────────────────────────────────

// DataPointRepository persists DataPoint metadata in Postgres (cognee_datapoints table).
type DataPointRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.DataPoint, error)
	Upsert(ctx context.Context, dp domain.DataPoint) error
}

// DataPointGraphRepository upserts DataPoint nodes and edges into Neo4j.
type DataPointGraphRepository interface {
	UpsertNode(ctx context.Context, node domain.GraphNode) error
	UpsertEdge(ctx context.Context, edge domain.GraphEdge) error
}

// DataPointVectorRepository upserts vector embeddings for indexed fields.
type DataPointVectorRepository interface {
	UpsertPoint(ctx context.Context, collection, pointID string, vec []float32, payload map[string]any) error
}

// DataPointEmbedder generates embeddings from text (NO LLM — embedding only).
type DataPointEmbedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// DataPointEventPublisher publishes domain events to NATS.
type DataPointEventPublisher interface {
	Publish(ctx context.Context, subject string, payload map[string]any) error
}

// ─── Use Case ─────────────────────────────────────────────────────────────────

// AddDataPointsUseCase handles structured ingestion of DataPoints without LLM calls.
// Flow: validate → version check → Neo4j node → Neo4j edges → embed index_fields → Postgres upsert → NATS event
type AddDataPointsUseCase struct {
	dataPointRepo DataPointRepository
	graphRepo     DataPointGraphRepository
	vectorRepo    DataPointVectorRepository
	embedder      DataPointEmbedder
	eventPub      DataPointEventPublisher
}

// NewAddDataPointsUseCase constructs the use case with dependencies injected.
func NewAddDataPointsUseCase(
	dataPointRepo DataPointRepository,
	graphRepo DataPointGraphRepository,
	vectorRepo DataPointVectorRepository,
	embedder DataPointEmbedder,
	eventPub DataPointEventPublisher,
) *AddDataPointsUseCase {
	return &AddDataPointsUseCase{
		dataPointRepo: dataPointRepo,
		graphRepo:     graphRepo,
		vectorRepo:    vectorRepo,
		embedder:      embedder,
		eventPub:      eventPub,
	}
}

// Execute processes all DataPoints in the request, running 5 steps per point.
func (uc *AddDataPointsUseCase) Execute(ctx context.Context, req AddDataPointsRequest) (*AddDataPointsResult, error) {
	result := &AddDataPointsResult{}

	for i := range req.DataPoints {
		dp := &req.DataPoints[i]

		// Attach dataset context + propagate NodeSets if not set per-point
		dp.DatasetID = req.DatasetID
		dp.TenantID = req.TenantID
		if len(req.NodeSets) > 0 && len(dp.NodeSets) == 0 {
			dp.NodeSets = req.NodeSets
		}

		// Step 1: Validate
		if err := validateDataPoint(*dp); err != nil {
			return nil, fmt.Errorf("datapoint %s: %w", dp.ID, err)
		}

		// Step 2: Version check (upsert semantics — same ID → increment version)
		existing, _ := uc.dataPointRepo.GetByID(ctx, dp.ID)
		if existing != nil {
			dp.Version = existing.Version + 1
			result.Updated++
		} else {
			dp.Version = 1
			result.Created++
		}

		// Step 3: Upsert Neo4j node (labels = [DataPoint, Type, ...NodeSets])
		node := buildDataPointGraphNode(*dp)
		if err := uc.graphRepo.UpsertNode(ctx, node); err != nil {
			return nil, fmt.Errorf("upsert neo4j node %s: %w", dp.ID, err)
		}

		// Step 4: Create Neo4j edges from Relations (non-fatal — target may not exist yet)
		for _, rel := range dp.Relations {
			edge := domain.GraphEdge{
				ID:        buildEdgeID(dp.ID, rel.TargetID, rel.Label),
				Subject:   dp.ID.String(),
				Object:    rel.TargetID.String(),
				Predicate: rel.Label,
				Properties: map[string]any{"weight": rel.Weight},
			}
			_ = uc.graphRepo.UpsertEdge(ctx, edge) // non-fatal
		}

		// Step 5: Embed only IndexFields → Qdrant (NO LLM — pure embedding)
		_ = uc.embedIndexFields(ctx, *dp) // non-fatal

		// Step 6: Persist metadata in Postgres
		_ = uc.dataPointRepo.Upsert(ctx, *dp)
		result.Upserted++
	}

	// Emit NATS event for downstream consumers (e.g., cognee-cognify community detection)
	_ = uc.eventPub.Publish(ctx, "cognee.ingestion.datapoints.added", map[string]any{
		"dataset_id": req.DatasetID.String(),
		"tenant_id":  req.TenantID,
		"count":      len(req.DataPoints),
	})

	return result, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildDataPointGraphNode maps a DataPoint → GraphNode for Neo4j persistence.
// Labels: [DataPoint, Type, ...NodeSets] — enables multi-label filtering.
func buildDataPointGraphNode(dp domain.DataPoint) domain.GraphNode {
	labels := []string{"DataPoint", dp.Type}
	labels = append(labels, dp.NodeSets...)

	props := map[string]any{
		"id":         dp.ID.String(),
		"dataset_id": dp.DatasetID.String(),
		"tenant_id":  dp.TenantID,
		"version":    dp.Version,
		"type":       dp.Type,
	}
	for k, v := range dp.Fields {
		props[k] = v
	}

	return domain.GraphNode{
		ID:         dp.ID.String(),
		Name:       extractFieldName(dp.Fields),
		Type:       dp.Type,
		Labels:     labels,
		Properties: props,
	}
}

// embedIndexFields embeds only the specified index_fields text and upserts into Qdrant.
func (uc *AddDataPointsUseCase) embedIndexFields(ctx context.Context, dp domain.DataPoint) error {
	if len(dp.IndexFields) == 0 {
		return nil
	}

	var parts []string
	for _, field := range dp.IndexFields {
		if val, ok := dp.Fields[field]; ok {
			parts = append(parts, fmt.Sprint(val))
		}
	}
	if len(parts) == 0 {
		return nil
	}

	text := strings.Join(parts, " ")
	vec, err := uc.embedder.Embed(ctx, text)
	if err != nil {
		return err
	}

	collectionName := fmt.Sprintf("cognee_%s", dp.TenantID)
	return uc.vectorRepo.UpsertPoint(ctx, collectionName, dp.ID.String(), vec, map[string]any{
		"dataset_id":   dp.DatasetID.String(),
		"datapoint_id": dp.ID.String(),
		"type":         dp.Type,
		"node_sets":    dp.NodeSets,
	})
}

// validateDataPoint checks required fields and index_fields consistency.
func validateDataPoint(dp domain.DataPoint) error {
	if dp.Type == "" {
		return fmt.Errorf("type is required")
	}
	if len(dp.Fields) == 0 {
		return fmt.Errorf("fields cannot be empty")
	}
	for _, field := range dp.IndexFields {
		if _, ok := dp.Fields[field]; !ok {
			return fmt.Errorf("index_field %q not found in fields", field)
		}
	}
	return nil
}

// buildEdgeID generates a deterministic edge ID from source, target, and label.
func buildEdgeID(src, dst uuid.UUID, label string) string {
	return domain.DeterministicUUID(
		fmt.Sprintf("%s_%s", label, dst.String()),
		src.String(),
	).String()
}

// extractFieldName picks the most descriptive field for use as graph node name.
func extractFieldName(fields map[string]any) string {
	for _, key := range []string{"name", "title", "label", "id"} {
		if v, ok := fields[key]; ok {
			return fmt.Sprint(v)
		}
	}
	return "unnamed"
}
