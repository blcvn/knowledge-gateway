package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gorm.io/gorm"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/domain"
)

type DocumentRepository interface {
	CreateDocument(ctx context.Context, doc *domain.Document) error
	GetDocumentMetrics(ctx context.Context, docID string) (*domain.Document, error)
	UpdateDocument(ctx context.Context, doc *domain.Document) error
	DeleteDocument(ctx context.Context, docID string) error
	CreateRelationship(ctx context.Context, sourceID, targetID, relType string) error
	GetTraceability(ctx context.Context, docID string) ([]map[string]interface{}, error)
}

type documentRepository struct {
	pgDB        *gorm.DB
	neo4jDriver neo4j.DriverWithContext
}

func NewDocumentRepository(pgDB *gorm.DB, driver neo4j.DriverWithContext) DocumentRepository {
	return &documentRepository{
		pgDB:        pgDB,
		neo4jDriver: driver,
	}
}

func (r *documentRepository) CreateDocument(ctx context.Context, doc *domain.Document) error {
	// 1. Save to Postgres
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	if err := r.pgDB.WithContext(ctx).Create(doc).Error; err != nil {
		return fmt.Errorf("failed to create document in Postgres: %w", err)
	}

	// 2. Save to Neo4j
	session := r.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := fmt.Sprintf("MERGE (n:%s {id: $id}) SET n.title = $title, n.version = $version, n.status = $status", doc.DocType)
	params := map[string]interface{}{
		"id":      doc.ID,
		"title":   doc.Title,
		"version": doc.Version,
		"status":  doc.Status,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		// Rollback Postgres? Ideally use distributed transaction or Saga
		return fmt.Errorf("failed to create node in Neo4j: %w", err)
	}

	return nil
}

func (r *documentRepository) GetDocumentMetrics(ctx context.Context, docID string) (*domain.Document, error) {
	var doc domain.Document
	if err := r.pgDB.WithContext(ctx).First(&doc, "id = ?", docID).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *documentRepository) UpdateDocument(ctx context.Context, doc *domain.Document) error {
	doc.UpdatedAt = time.Now()
	if err := r.pgDB.WithContext(ctx).Save(doc).Error; err != nil {
		return err
	}

	session := r.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := "MATCH (n {id: $id}) SET n.title = $title, n.status = $status, n.version = $version"
	params := map[string]interface{}{
		"id":      doc.ID,
		"title":   doc.Title,
		"status":  doc.Status,
		"version": doc.Version,
	}
	_, err := session.Run(ctx, query, params)
	return err
}

func (r *documentRepository) DeleteDocument(ctx context.Context, docID string) error {
	if err := r.pgDB.WithContext(ctx).Delete(&domain.Document{}, "id = ?", docID).Error; err != nil {
		return err
	}

	session := r.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := "MATCH (n {id: $id}) DETACH DELETE n"
	_, err := session.Run(ctx, query, map[string]interface{}{"id": docID})
	return err
}

func (r *documentRepository) CreateRelationship(ctx context.Context, sourceID, targetID, relType string) error {
	session := r.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Example: MATCH (a {id: $sid}), (b {id: $tid}) MERGE (a)-[:GENERATES]->(b)
	query := fmt.Sprintf("MATCH (a {id: $sid}), (b {id: $tid}) MERGE (a)-[:%s]->(b)", relType)
	params := map[string]interface{}{
		"sid": sourceID,
		"tid": targetID,
	}
	_, err := session.Run(ctx, query, params)
	return err
}

func (r *documentRepository) GetTraceability(ctx context.Context, docID string) ([]map[string]interface{}, error) {
	session := r.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Trace downstream dependencies: (ThisDoc)-[:GENERATES|EXPANDS_TO*]->(Others)
	query := `
		MATCH (start {id: $id})-[r*]->(end)
		RETURN start.id as source, type(last(r)) as relationship, end.id as target, labels(end) as target_type
	`
	params := map[string]interface{}{
		"id": docID,
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var trace []map[string]interface{}
	for result.Next(ctx) {
		record := result.Record()
		source, _ := record.Get("source")
		relationship, _ := record.Get("relationship")
		target, _ := record.Get("target")
		targetType, _ := record.Get("target_type")

		trace = append(trace, map[string]interface{}{
			"source":       source,
			"relationship": relationship,
			"target":       target,
			"target_type":  targetType,
		})
	}

	return trace, nil
}
