package data

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type registryRepo struct {
	data *Data
	log  *log.Helper
}

// NewRegistryRepo .
func NewRegistryRepo(data *Data, logger log.Logger) *registryRepo {
	return &registryRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *registryRepo) CreateApp(ctx context.Context, app *biz.App) error {
	// 1. Transaction in Postgres
	err := r.data.db.WithContext(ctx).Create(app).Error
	if err != nil {
		return err
	}

	// 2. Reserve Namespace in Neo4j
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MERGE (n:__KGS_Namespace {app_id: $app_id})
			ON CREATE SET n.created_at = datetime()
			RETURN n
		`
		return tx.Run(ctx, query, map[string]any{"app_id": app.AppID})
	})

	if err != nil {
		r.log.Errorf("Failed to reserve namespace in Neo4j for app %s: %v", app.AppID, err)
		// We could potentially rollback Postgres here (saga pattern)
		return err
	}

	return nil
}

func (r *registryRepo) DeleteApp(ctx context.Context, appID string) error {
	// 1. Delete from Postgres
	err := r.data.db.WithContext(ctx).Where("app_id = ?", appID).Delete(&biz.App{}).Error
	if err != nil {
		return err
	}

	// 2. Remove Namespace metadata in Neo4j (soft delete or hard delete node metadata)
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n:__KGS_Namespace {app_id: $app_id})
			SET n.status = 'DEPRECATED', n.deleted_at = datetime()
			RETURN n
		`
		return tx.Run(ctx, query, map[string]any{"app_id": appID})
	})

	if err != nil {
		r.log.Errorf("Failed to mark namespace as deleted in Neo4j: %v", err)
	}

	return nil
}
