package postgres

import (
\t"context"
\t"database/sql"
\t"errors"
\t"fmt"

\t"vnp-memory/services/zep-search/internal/domain/search"

\t"github.com/pgvector/pgvector-go"
)

// SearchRepositoryImpl implements the SearchRepository port using PostgreSQL and pgvector.
type SearchRepositoryImpl struct {
\tdb *sql.DB
}

// NewSearchRepository returns a new instance of SearchRepositoryImpl.
func NewSearchRepository(db *sql.DB) *SearchRepositoryImpl {
\treturn &SearchRepositoryImpl{
\t\tdb: db,
\t}
}

// VectorSearch executes a semantic search using pgvector's cosine distance operator (<=>).
func (r *SearchRepositoryImpl) VectorSearch(ctx context.Context, criteria search.HybridSearchCriteria, limit int) ([]search.SearchResult, error) {
\tif len(criteria.QueryVector) == 0 {
\t\treturn nil, errors.New("query vector is required for vector search")
\t}

\t// Convert []float32 to pgvector.Vector
\tvector := pgvector.NewVector(criteria.QueryVector)

\t// Execute pgvector similarity search
\t// We assume the table 'document_chunks' has a 'embedding' column of type vector(1536)
\t// The operator <=> computes cosine distance. We order by distance ascending.
\tquery := `
\t\tSELECT id, document_id, content, 1 - (embedding <=> $1) AS similarity_score
\t\tFROM document_chunks
\t\tORDER BY embedding <=> $1
\t\tLIMIT $2;
\t`

\trows, err := r.db.QueryContext(ctx, query, vector, limit)
\tif err != nil {
\t\treturn nil, fmt.Errorf("failed to execute vector search: %w", err)
\t}
\tdefer rows.Close()

\tvar results []search.SearchResult
\tfor rows.Next() {
\t\tvar res search.SearchResult
\t\tif err := rows.Scan(&res.ChunkID, &res.DocumentID, &res.Content, &res.Score); err != nil {
\t\t\treturn nil, fmt.Errorf("failed to scan row: %w", err)
\t\t}
\t\tresults = append(results, res)
\t}

\tif err = rows.Err(); err != nil {
\t\treturn nil, err
\t}

\treturn results, nil
}

// BM25Search executes a fulltext search using PostgreSQL's native tsvector/tsquery.
func (r *SearchRepositoryImpl) BM25Search(ctx context.Context, criteria search.HybridSearchCriteria, limit int) ([]search.SearchResult, error) {
\tif criteria.RawText == "" {
\t\treturn nil, errors.New("raw text is required for BM25 search")
\t}

\t// Execute PostgreSQL full text search (BM25 equivalent via ts_rank)
\tquery := `
\t\tSELECT id, document_id, content, ts_rank(fts_vector, websearch_to_tsquery('english', $1)) AS text_score
\t\tFROM document_chunks
\t\tWHERE fts_vector @@ websearch_to_tsquery('english', $1)
\t\tORDER BY text_score DESC
\t\tLIMIT $2;
\t`

\trows, err := r.db.QueryContext(ctx, query, criteria.RawText, limit)
\tif err != nil {
\t\treturn nil, fmt.Errorf("failed to execute BM25 search: %w", err)
\t}
\tdefer rows.Close()

\tvar results []search.SearchResult
\tfor rows.Next() {
\t\tvar res search.SearchResult
\t\tif err := rows.Scan(&res.ChunkID, &res.DocumentID, &res.Content, &res.Score); err != nil {
\t\t\treturn nil, fmt.Errorf("failed to scan row: %w", err)
\t\t}
\t\tresults = append(results, res)
\t}

\treturn results, nil
}
