package usecase

import (
    "context"
    "strings"

    pkgsearch "github.com/vnp-memory/pkg/search"
    "vnp-memory/services/search-service/internal/usecase/port"
)

type IndexAddUseCase struct {
    bm25      *pkgsearch.BM25Index
    vector    *pkgsearch.VectorIndex
    embedder  port.IEmbedder
    persister *pkgsearch.IndexPersister
}

type IndexAddRequest struct {
    ObsID     string
    SessionID string
    AgentID   string
    TenantID  string
    Title     string
    Facts     []string
    Concepts  []string
}

func (uc *IndexAddUseCase) Execute(ctx context.Context, req IndexAddRequest) error {
    // Build indexable text: title + facts + concepts
    text := req.Title + " " + strings.Join(req.Facts, " ") + " " + strings.Join(req.Concepts, " ")

    // Add to BM25
    uc.bm25.Add(req.ObsID, req.SessionID, req.AgentID, req.TenantID, text)

    // Add to vector (if embedder available)
    if uc.embedder != nil {
        if vec, err := uc.embedder.Embed(ctx, text); err == nil && vec != nil {
            uc.vector.Add(req.ObsID, req.SessionID, vec)
        }
    }

    // Debounced persist
    uc.persister.Schedule()
    return nil
}
