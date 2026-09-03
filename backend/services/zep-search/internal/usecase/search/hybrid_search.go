package search

import (
\t"context"
\t"errors"
\t"sync"

\tdomain "vnp-memory/services/zep-search/internal/domain/search"
\tranker "vnp-memory/services/zep-search/internal/usecase/ranker"
)

// SearchRepository defines the output port for searching data via various indexing methods.
type SearchRepository interface {
\tVectorSearch(ctx context.Context, criteria domain.HybridSearchCriteria, limit int) ([]domain.SearchResult, error)
\tBM25Search(ctx context.Context, criteria domain.HybridSearchCriteria, limit int) ([]domain.SearchResult, error)
}

// EmbeddingService defines the port to an external model (e.g. OpenAI) to generate vectors.
type EmbeddingService interface {
\tGenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// SearchUseCase orchestrates the hybrid search pipeline.
type SearchUseCase struct {
\trepo     SearchRepository
\tembedder EmbeddingService
}

// NewSearchUseCase creates a new SearchUseCase instance.
func NewSearchUseCase(repo SearchRepository, embedder EmbeddingService) *SearchUseCase {
\treturn &SearchUseCase{
\t\trepo:     repo,
\t\tembedder: embedder,
\t}
}

// HybridSearch performs a concurrent Vector and BM25 search, then fuses the results using RRF.
func (uc *SearchUseCase) HybridSearch(ctx context.Context, query *domain.SearchQuery) ([]domain.SearchResult, error) {
\tif !query.Valid() {
\t\treturn nil, errors.New("invalid search query parameters")
\t}

\t// 1. Generate Embeddings for the search text
\t// This could be skipped if we only wanted BM25, but this is a Hybrid search.
\tvector, err := uc.embedder.GenerateEmbedding(ctx, query.Text)
\tif err != nil {
\t\treturn nil, err
\t}

\tcriteria := domain.HybridSearchCriteria{
\t\tQueryVector: vector,
\t\tRawText:     query.Text,
\t\tAlpha:       0.5,
\t}

\t// 2. Execute Vector Search and BM25 Search Concurrently
\tvar wg sync.WaitGroup
\tvar vectorResults, bm25Results []domain.SearchResult
\tvar vectorErr, bm25Err error

\twg.Add(2)

\tgo func() {
\t\tdefer wg.Done()
\t\tvectorResults, vectorErr = uc.repo.VectorSearch(ctx, criteria, query.Limit*2) // Fetch more to fuse
\t}()

\tgo func() {
\t\tdefer wg.Done()
\t\tbm25Results, bm25Err = uc.repo.BM25Search(ctx, criteria, query.Limit*2)
\t}()

\t// Wait for both concurrent searches to complete
\twg.Wait()

\t// If both fail, we return an error. If one succeeds, we can still proceed with partial results (Resilience).
\tif vectorErr != nil && bm25Err != nil {
\t\treturn nil, errors.New("both vector and bm25 search failed")
\t}

\t// 3. Prepare data for Reciprocal Rank Fusion (RRF)
\tvar rankedLists [][]ranker.DocumentScore

\tif vectorErr == nil && len(vectorResults) > 0 {
\t\tlist := make([]ranker.DocumentScore, len(vectorResults))
\t\tfor i, res := range vectorResults {
\t\t\tlist[i] = ranker.DocumentScore{DocumentID: res.DocumentID, Score: res.Score, Rank: i + 1}
\t\t}
\t\trankedLists = append(rankedLists, list)
\t}

\tif bm25Err == nil && len(bm25Results) > 0 {
\t\tlist := make([]ranker.DocumentScore, len(bm25Results))
\t\tfor i, res := range bm25Results {
\t\t\tlist[i] = ranker.DocumentScore{DocumentID: res.DocumentID, Score: res.Score, Rank: i + 1}
\t\t}
\t\trankedLists = append(rankedLists, list)
\t}

\t// 4. Execute RRF
\trrfParams := ranker.DefaultRRFParameters()
\tfusedResults := ranker.Fuse(rankedLists, rrfParams)

\t// 5. Map back to SearchResults and apply limits
\tfinalResults := make([]domain.SearchResult, 0, query.Limit)
\tfor i, fused := range fusedResults {
\t\tif i >= query.Limit {
\t\t\tbreak
\t\t}
\t\t
\t\t// In a real scenario, you might want to fetch the actual snippets/content from DB again,
\t\t// or build a lookup map from vectorResults/bm25Results to attach the Content here.
\t\t// For simplicity, we just return the fused score and document ID.
\t\tfinalResults = append(finalResults, domain.SearchResult{
\t\t\tDocumentID: fused.DocumentID,
\t\t\tScore:      fused.Score,
\t\t})
\t}

\treturn finalResults, nil
}
