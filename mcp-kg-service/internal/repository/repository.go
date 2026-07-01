package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository is the direct Postgres access layer for MCP KG queries.
type Repository struct {
	db *gorm.DB
}

type DocumentSummary struct {
	DocumentID   string    `json:"document_id"`
	NodeCount    int64     `json:"node_count"`
	EdgeCount    int64     `json:"edge_count"`
	LastUpdated  time.Time `json:"last_updated"`
	DocumentType string    `json:"document_type,omitempty"`
	FeatureID    string    `json:"feature_id,omitempty"`
	FeatureName  string    `json:"feature_name,omitempty"`
}

type RequirementNode struct {
	ID          string         `json:"id"`
	DocumentID  string         `json:"document_id"`
	ReferenceID string         `json:"reference_id"`
	Type        string         `json:"type"`
	Summary     string         `json:"summary"`
	Description string         `json:"description"`
	SourceID    string         `json:"source_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type DependencyEdge struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	SourceID   string `json:"source_id"`
	TargetID   string `json:"target_id"`
	Type       string `json:"type"`
	Reason     string `json:"reason,omitempty"`
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) ListDocuments(ctx context.Context) ([]DocumentSummary, error) {
	type nodeSummaryRow struct {
		DocumentID  string
		NodeCount   int64
		LastUpdated sql.NullString
	}
	type edgeCountRow struct {
		DocumentID string
		EdgeCount  int64
	}

	var nodeRows []nodeSummaryRow
	if err := r.db.WithContext(ctx).
		Model(&RequirementNodeModel{}).
		Select("document_id, COUNT(*) AS node_count, MAX(updated_at) AS last_updated").
		Where("document_id <> ''").
		Group("document_id").
		Order("MAX(updated_at) DESC").
		Scan(&nodeRows).Error; err != nil {
		return nil, fmt.Errorf("list KG documents from requirement_nodes: %w", err)
	}

	var edgeRows []edgeCountRow
	if err := r.db.WithContext(ctx).
		Model(&DependencyEdgeModel{}).
		Select("document_id, COUNT(*) AS edge_count").
		Where("document_id <> ''").
		Group("document_id").
		Scan(&edgeRows).Error; err != nil {
		return nil, fmt.Errorf("list KG document edge counts: %w", err)
	}

	edgeCountByDocument := make(map[string]int64, len(edgeRows))
	for _, row := range edgeRows {
		edgeCountByDocument[row.DocumentID] = row.EdgeCount
	}

	summaries := make([]DocumentSummary, 0, len(nodeRows))
	for _, row := range nodeRows {
		summaries = append(summaries, DocumentSummary{
			DocumentID:  row.DocumentID,
			NodeCount:   row.NodeCount,
			EdgeCount:   edgeCountByDocument[row.DocumentID],
			LastUpdated: parseNullableTime(row.LastUpdated),
		})
	}
	return summaries, nil
}

func (r *Repository) ListDocumentsByProject(ctx context.Context, projectID string) ([]DocumentSummary, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	type workspaceDocRow struct {
		DocumentID  string
		FeatureID   string
		FeatureName string
		DocType     string
	}
	var workspaceDocs []workspaceDocRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT fd.id AS document_id, fd.feature_id, f.name AS feature_name, fd.type AS doc_type
		FROM feature_documents fd
		JOIN features f ON f.id = fd.feature_id
		WHERE f.project_id = ? AND fd.deleted_at IS NULL
		ORDER BY f.name ASC, fd.type ASC
	`, projectID).Scan(&workspaceDocs).Error; err != nil {
		return nil, fmt.Errorf("list workspace documents for project %q: %w", projectID, err)
	}

	if len(workspaceDocs) == 0 {
		return []DocumentSummary{}, nil
	}

	docIDs := make([]string, 0, len(workspaceDocs))
	for _, row := range workspaceDocs {
		docIDs = append(docIDs, row.DocumentID)
	}

	type nodeSummaryRow struct {
		DocumentID  string
		NodeCount   int64
		LastUpdated sql.NullString
	}
	type edgeCountRow struct {
		DocumentID string
		EdgeCount  int64
	}

	var nodeRows []nodeSummaryRow
	if err := r.db.WithContext(ctx).
		Model(&RequirementNodeModel{}).
		Select("document_id, COUNT(*) AS node_count, MAX(updated_at) AS last_updated").
		Where("document_id IN ?", docIDs).
		Group("document_id").
		Scan(&nodeRows).Error; err != nil {
		return nil, fmt.Errorf("list KG node counts for project %q: %w", projectID, err)
	}

	var edgeRows []edgeCountRow
	if err := r.db.WithContext(ctx).
		Model(&DependencyEdgeModel{}).
		Select("document_id, COUNT(*) AS edge_count").
		Where("document_id IN ?", docIDs).
		Group("document_id").
		Scan(&edgeRows).Error; err != nil {
		return nil, fmt.Errorf("list KG edge counts for project %q: %w", projectID, err)
	}

	nodeByDoc := make(map[string]nodeSummaryRow, len(nodeRows))
	for _, row := range nodeRows {
		nodeByDoc[row.DocumentID] = row
	}
	edgeByDoc := make(map[string]int64, len(edgeRows))
	for _, row := range edgeRows {
		edgeByDoc[row.DocumentID] = row.EdgeCount
	}

	summaries := make([]DocumentSummary, 0, len(workspaceDocs))
	for _, row := range workspaceDocs {
		summary := DocumentSummary{
			DocumentID:   row.DocumentID,
			FeatureID:    row.FeatureID,
			FeatureName:  row.FeatureName,
			DocumentType: row.DocType,
			EdgeCount:    edgeByDoc[row.DocumentID],
		}
		if nodeRow, ok := nodeByDoc[row.DocumentID]; ok {
			summary.NodeCount = nodeRow.NodeCount
			summary.LastUpdated = parseNullableTime(nodeRow.LastUpdated)
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r *Repository) ListNodesByDocumentID(ctx context.Context, documentID string, nodeTypes []string, limit int) ([]RequirementNode, error) {
	query := r.db.WithContext(ctx).
		Model(&RequirementNodeModel{}).
		Where("document_id = ?", strings.TrimSpace(documentID)).
		Order("updated_at DESC")

	if len(nodeTypes) > 0 {
		query = query.Where("type IN ?", nodeTypes)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var models []RequirementNodeModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list requirement nodes by document_id %q: %w", documentID, err)
	}

	nodes := make([]RequirementNode, 0, len(models))
	for _, model := range models {
		node, err := mapRequirementNode(model)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (r *Repository) CountNodesByDocumentID(ctx context.Context, documentID string, nodeTypes []string) (int64, error) {
	query := r.db.WithContext(ctx).
		Model(&RequirementNodeModel{}).
		Where("document_id = ?", strings.TrimSpace(documentID))

	if len(nodeTypes) > 0 {
		query = query.Where("type IN ?", nodeTypes)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count requirement nodes by document_id %q: %w", documentID, err)
	}
	return count, nil
}

func (r *Repository) ListEdgesByDocumentID(ctx context.Context, documentID string, nodeIDs []string) ([]DependencyEdge, error) {
	query := r.db.WithContext(ctx).
		Model(&DependencyEdgeModel{}).
		Where("document_id = ?", strings.TrimSpace(documentID)).
		Order("created_at ASC")

	filteredNodeIDs := nonEmptyUnique(nodeIDs)
	if len(filteredNodeIDs) > 0 {
		query = query.Where("source_id IN ? AND target_id IN ?", filteredNodeIDs, filteredNodeIDs)
	}

	var models []DependencyEdgeModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list dependency edges by document_id %q: %w", documentID, err)
	}

	edges := make([]DependencyEdge, 0, len(models))
	for _, model := range models {
		edges = append(edges, DependencyEdge{
			ID:         model.ID,
			DocumentID: model.DocumentID,
			SourceID:   model.SourceID,
			TargetID:   model.TargetID,
			Type:       model.Type,
			Reason:     model.Reason,
		})
	}
	return edges, nil
}

func (r *Repository) SearchFeatureNodes(ctx context.Context, query, documentID string, limit int) ([]RequirementNode, error) {
	query = strings.TrimSpace(query)
	dbQuery := r.db.WithContext(ctx).
		Model(&RequirementNodeModel{}).
		Where("type IN ?", []string{"FEATURE", "FUNCTIONAL"}).
		Where("(LOWER(summary) LIKE ? OR LOWER(description) LIKE ?)", "%"+strings.ToLower(query)+"%", "%"+strings.ToLower(query)+"%").
		Order("updated_at DESC")

	if documentID = strings.TrimSpace(documentID); documentID != "" {
		dbQuery = dbQuery.Where("document_id = ?", documentID)
	}
	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}

	var models []RequirementNodeModel
	if err := dbQuery.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("search feature nodes: %w", err)
	}

	nodes := make([]RequirementNode, 0, len(models))
	for _, model := range models {
		node, err := mapRequirementNode(model)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func mapRequirementNode(model RequirementNodeModel) (RequirementNode, error) {
	node := RequirementNode{
		ID:          model.ID,
		DocumentID:  model.DocumentID,
		ReferenceID: model.ReferenceID,
		Type:        model.Type,
		Summary:     model.Summary,
		Description: model.Description,
		SourceID:    model.SourceID,
	}
	if len(model.Metadata) == 0 {
		return node, nil
	}
	if err := json.Unmarshal(model.Metadata, &node.Metadata); err != nil {
		return RequirementNode{}, fmt.Errorf("decode node metadata for %q: %w", model.ID, err)
	}
	return node, nil
}

func nonEmptyUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func parseNullableTime(value sql.NullString) time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value.String); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func (r *Repository) SaveGraphSnapshot(ctx context.Context, documentID string, nodes []RequirementNode, edges []DependencyEdge) error {
	documentID = strings.TrimSpace(documentID)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("document_id = ?", documentID).Delete(&DependencyEdgeModel{}).Error; err != nil {
			return fmt.Errorf("delete existing dependency edges for %q: %w", documentID, err)
		}
		if err := tx.Where("document_id = ?", documentID).Delete(&RequirementNodeModel{}).Error; err != nil {
			return fmt.Errorf("delete existing requirement nodes for %q: %w", documentID, err)
		}

		if len(nodes) > 0 {
			models, err := buildRequirementNodeModels(documentID, nodes)
			if err != nil {
				return err
			}
			if err := tx.Create(&models).Error; err != nil {
				return fmt.Errorf("insert requirement nodes for %q: %w", documentID, err)
			}
		}

		if len(edges) > 0 {
			models := buildDependencyEdgeModels(documentID, edges)
			if err := tx.Create(&models).Error; err != nil {
				return fmt.Errorf("insert dependency edges for %q: %w", documentID, err)
			}
		}
		return nil
	})
}

func (r *Repository) UpsertDocumentPatch(ctx context.Context, documentID string, nodes []RequirementNode, edges []DependencyEdge) error {
	documentID = strings.TrimSpace(documentID)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(nodes) > 0 {
			models, err := buildRequirementNodeModels(documentID, nodes)
			if err != nil {
				return err
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"document_id",
					"reference_id",
					"type",
					"summary",
					"description",
					"source_id",
					"metadata",
					"updated_at",
				}),
			}).Create(&models).Error; err != nil {
				return fmt.Errorf("upsert requirement nodes for %q: %w", documentID, err)
			}
		}

		if len(edges) > 0 {
			models := buildDependencyEdgeModels(documentID, edges)
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"document_id",
					"source_id",
					"target_id",
					"type",
					"reason",
				}),
			}).Create(&models).Error; err != nil {
				return fmt.Errorf("upsert dependency edges for %q: %w", documentID, err)
			}
		}
		return nil
	})
}

func (r *Repository) SaveDocumentArtifact(ctx context.Context, artifact DocumentArtifactModel) error {
	artifact.DocumentID = strings.TrimSpace(artifact.DocumentID)
	artifact.DocKind = strings.TrimSpace(artifact.DocKind)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "document_id"}, {Name: "doc_kind"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "content_type", "updated_at"}),
	}).Create(&artifact).Error
}

func (r *Repository) GetDocumentArtifact(ctx context.Context, documentID, docKind string) (*DocumentArtifactModel, error) {
	var model DocumentArtifactModel
	if err := r.db.WithContext(ctx).
		Where("document_id = ? AND doc_kind = ?", strings.TrimSpace(documentID), strings.TrimSpace(docKind)).
		First(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func buildRequirementNodeModels(documentID string, nodes []RequirementNode) ([]RequirementNodeModel, error) {
	now := time.Now().UTC()
	models := make([]RequirementNodeModel, 0, len(nodes))
	for _, node := range nodes {
		meta, err := json.Marshal(node.Metadata)
		if err != nil {
			return nil, fmt.Errorf("encode node metadata for %q: %w", node.ID, err)
		}
		models = append(models, RequirementNodeModel{
			ID:          strings.TrimSpace(node.ID),
			DocumentID:  documentID,
			ReferenceID: strings.TrimSpace(node.ReferenceID),
			Type:        strings.TrimSpace(node.Type),
			Summary:     node.Summary,
			Description: node.Description,
			SourceID:    strings.TrimSpace(node.SourceID),
			Metadata:    meta,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return models, nil
}

func buildDependencyEdgeModels(documentID string, edges []DependencyEdge) []DependencyEdgeModel {
	now := time.Now().UTC()
	models := make([]DependencyEdgeModel, 0, len(edges))
	for _, edge := range edges {
		models = append(models, DependencyEdgeModel{
			ID:         strings.TrimSpace(edge.ID),
			DocumentID: documentID,
			SourceID:   strings.TrimSpace(edge.SourceID),
			TargetID:   strings.TrimSpace(edge.TargetID),
			Type:       strings.TrimSpace(edge.Type),
			Reason:     edge.Reason,
			CreatedAt:  now,
		})
	}
	return models
}
