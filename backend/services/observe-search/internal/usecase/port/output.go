package port

import "context"

// IObservationStore is the output port for reading stored observations.
type IObservationStore interface {
    GetRecentSummaries(ctx context.Context, tenantID, project string, limit int) ([]Summary, error)
}

// Summary is a brief narrative of a session.
type Summary struct {
    SessionID string
    Narrative string
}

// IEmbedder is the output port for generating text embeddings.
type IEmbedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}
