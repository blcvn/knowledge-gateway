package index

import (
    "context"
    "strings"

    pkg_search "github.com/vnp-memory/pkg/search"
    "github.com/vnp-memory/services/observe-search/internal/search"
)

// IndexAddRequest contains data for adding a new document to the index.
type IndexAddRequest struct {
    ObsID     string
    SessionID string
    AgentID   string
    Title     string
    Facts     []string
    Concepts  []string
    Text      string
}

// Manager manages the lifecycle of the BM25 and vector indexes.
type Manager struct {
    bm25      *pkg_search.BM25Index
    vector    *pkg_search.VectorIndex
    embedder  search.IEmbedder
    persister *pkg_search.IndexPersister
}

func NewManager(bm25 *pkg_search.BM25Index, vector *pkg_search.VectorIndex, embedder search.IEmbedder, persister *pkg_search.IndexPersister) *Manager {
    return &Manager{bm25: bm25, vector: vector, embedder: embedder, persister: persister}
}

// Add adds an observation to the search indexes.
func (m *Manager) Add(ctx context.Context, req IndexAddRequest) error {
    text := req.Text
    if text == "" {
        text = req.Title + " " + strings.Join(req.Facts, " ") + " " + strings.Join(req.Concepts, " ")
    }
    m.bm25.Add(req.ObsID, req.SessionID, req.AgentID, "", text)

    if m.embedder != nil {
        vec, err := m.embedder.Embed(ctx, text)
        if err == nil && vec != nil {
            _ = m.vector.Add(req.ObsID, req.SessionID, vec)
        }
    }

    m.persister.Schedule()
    return nil
}

// Remove removes a document from all indexes.
func (m *Manager) Remove(_ context.Context, obsID string) error {
    m.bm25.Remove(obsID)
    m.vector.Remove(obsID)
    m.persister.Schedule()
    return nil
}
