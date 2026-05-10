// Package pgvector implements the VectorStore interface using PostgreSQL + pgvector.
//
// Requires: CREATE EXTENSION IF NOT EXISTS vector;
// Uses HNSW indexing for approximate nearest neighbor search.
package pgvector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/pkg/vectorstore"
)

// Adapter implements vectorstore.VectorStore using pgvector.
type Adapter struct {
	pool *pgxpool.Pool
}

// New creates a new pgvector adapter.
func New(pool *pgxpool.Pool) *Adapter {
	return &Adapter{pool: pool}
}

func (a *Adapter) EnsureCollection(ctx context.Context, cfg vectorstore.CollectionConfig) error {
	distOp := cosineOp(cfg.DistanceMetric)

	// Create table
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			embedding vector(%d),
			metadata JSONB DEFAULT '{}',
			content TEXT DEFAULT ''
		)`, cfg.Name, cfg.Dimension)

	if _, err := a.pool.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("pgvector: create table %s: %w", cfg.Name, err)
	}

	// Create HNSW index for fast ANN search
	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_embedding_idx
		ON %s USING hnsw (embedding %s)
		WITH (m = 16, ef_construction = 200)`,
		cfg.Name, cfg.Name, distOp)

	if _, err := a.pool.Exec(ctx, indexSQL); err != nil {
		return fmt.Errorf("pgvector: create index on %s: %w", cfg.Name, err)
	}

	return nil
}

func (a *Adapter) Upsert(ctx context.Context, collection string, docs []vectorstore.Document) error {
	for _, doc := range docs {
		sql := fmt.Sprintf(`
			INSERT INTO %s (id, embedding, metadata, content)
			VALUES ($1, $2::vector, $3, $4)
			ON CONFLICT (id) DO UPDATE SET
				embedding = EXCLUDED.embedding,
				metadata = EXCLUDED.metadata,
				content = EXCLUDED.content`,
			collection)

		if _, err := a.pool.Exec(ctx, sql, doc.ID, pgVec(doc.Vector), doc.Metadata, doc.Content); err != nil {
			return fmt.Errorf("pgvector: upsert %s/%s: %w", collection, doc.ID, err)
		}
	}
	return nil
}

func (a *Adapter) Search(ctx context.Context, params vectorstore.SearchParams) ([]vectorstore.Document, error) {
	sql := fmt.Sprintf(`
		SELECT id, embedding::text, metadata, content,
			1 - (embedding <=> $1::vector) AS score
		FROM %s
		WHERE 1 - (embedding <=> $1::vector) >= $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3`,
		params.Collection)

	rows, err := a.pool.Query(ctx, sql, pgVec(params.Vector), params.MinScore, params.TopK)
	if err != nil {
		return nil, fmt.Errorf("pgvector: search %s: %w", params.Collection, err)
	}
	defer rows.Close()

	var results []vectorstore.Document
	for rows.Next() {
		var doc vectorstore.Document
		var vecStr string
		if err := rows.Scan(&doc.ID, &vecStr, &doc.Metadata, &doc.Content, &doc.Score); err != nil {
			return nil, fmt.Errorf("pgvector: scan result: %w", err)
		}
		results = append(results, doc)
	}
	return results, rows.Err()
}

func (a *Adapter) Delete(ctx context.Context, collection string, ids []string) error {
	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ANY($1)", collection)
	if _, err := a.pool.Exec(ctx, sql, ids); err != nil {
		return fmt.Errorf("pgvector: delete from %s: %w", collection, err)
	}
	return nil
}

func (a *Adapter) DropCollection(ctx context.Context, collection string) error {
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", collection)
	if _, err := a.pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("pgvector: drop %s: %w", collection, err)
	}
	return nil
}

func (a *Adapter) Close() error {
	a.pool.Close()
	return nil
}

// cosineOp maps DistanceMetric to pgvector operator class.
func cosineOp(metric vectorstore.DistanceMetric) string {
	switch metric {
	case vectorstore.Euclidean:
		return "vector_l2_ops"
	case vectorstore.DotProduct:
		return "vector_ip_ops"
	default:
		return "vector_cosine_ops"
	}
}

// pgVec converts a float32 slice to pgvector-compatible string format.
func pgVec(v vectorstore.Vector) string {
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%f", f)
	}
	return s + "]"
}
