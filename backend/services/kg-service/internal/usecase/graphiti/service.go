// Package graphiti implements the Graphiti usecases for kg-service.
//
// IngestUseCase: episode ingestion + entity extraction + graph upsert
// StoreUseCase: CRUD for nodes and edges
// SearchUseCase: semantic + graph + hybrid search
// KnowledgeUseCase: ontology management + subgraph queries
package graphiti

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vnp-memory/services/kg-service/internal/domain/graphiti"
	"vnp-memory/services/kg-service/internal/usecase/port"
)

// ─── Ingest ───────────────────────────────────────────────────────────────

// IngestUseCase handles episode ingestion and knowledge extraction.
type IngestUseCase struct {
	episodes  port.EpisodeRepository
	graph     GraphRepoInterface
	embedder  port.EmbeddingService
	publisher port.EventPublisher
}

// NewIngestUseCase creates an IngestUseCase.
func NewIngestUseCase(ep port.EpisodeRepository, g GraphRepoInterface, emb port.EmbeddingService, pub port.EventPublisher) *IngestUseCase {
	return &IngestUseCase{episodes: ep, graph: g, embedder: emb, publisher: pub}
}


// IngestEpisode creates and stores an episode, extracts knowledge graph entities.
func (uc *IngestUseCase) IngestEpisode(ctx context.Context, req graphiti.IngestRequest) (*graphiti.Episode, error) {
	episode := &graphiti.Episode{
		UUID:      uuid.New().String(),
		Name:      req.Name,
		Content:   req.Content,
		Source:    req.Source,
		SourceID:  req.SourceID,
		TenantID:  req.TenantID,
		CreatedAt: time.Now(),
	}

	// Generate embedding (non-fatal if embedder unavailable)
	if uc.embedder != nil {
		emb, err := uc.embedder.Embed(ctx, req.Content)
		if err == nil {
			episode.Embedding = emb
		}
	}

	// Persist episode
	if err := uc.episodes.Create(ctx, episode); err != nil {
		return nil, fmt.Errorf("persist episode: %w", err)
	}

	// Simple entity extraction (LLM-free heuristic for MVP)
	nodes, edges := extractEntities(episode)
	for _, node := range nodes {
		_ = uc.graph.UpsertNode(ctx, node)
	}
	for _, edge := range edges {
		_ = uc.graph.UpsertEdge(ctx, edge)
	}

	// Publish event
	if uc.publisher != nil {
		_ = uc.publisher.Publish(ctx, "kg.episode.ingested", episode)
	}

	return episode, nil
}

// extractEntities performs simple heuristic entity extraction.
// Full LLM-based extraction is done asynchronously via pipeline.
func extractEntities(ep *graphiti.Episode) ([]*graphiti.Node, []*graphiti.Edge) {
	// MVP: extract capitalized words as potential entities
	var nodes []*graphiti.Node
	words := strings.Fields(ep.Content)
	seen := map[string]string{} // name → uuid

	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'")
		if len(w) > 2 && w == strings.Title(strings.ToLower(w)) {
			if _, ok := seen[w]; !ok {
				nodeID := uuid.New().String()
				seen[w] = nodeID
				nodes = append(nodes, &graphiti.Node{
					UUID:      nodeID,
					Name:      w,
					Type:      "Entity",
					TenantID:  ep.TenantID,
					Episodes:  []string{ep.UUID},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}
		}
	}
	return nodes, nil // edges extracted by LLM pipeline separately
}

// ─── Store ────────────────────────────────────────────────────────────────

// StoreUseCase handles CRUD for graph nodes and edges.
type StoreUseCase struct {
	graph GraphRepoInterface
}

// NewStoreUseCase creates a StoreUseCase.
func NewStoreUseCase(g GraphRepoInterface) *StoreUseCase {
	return &StoreUseCase{graph: g}
}

func (uc *StoreUseCase) GetNode(ctx context.Context, tenantID, nodeUUID string) (*graphiti.Node, error) {
	node, err := uc.graph.GetNode(ctx, tenantID, nodeUUID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return node, nil
}

func (uc *StoreUseCase) GetEdge(ctx context.Context, tenantID, edgeUUID string) (*graphiti.Edge, error) {
	edge, err := uc.graph.GetEdge(ctx, tenantID, edgeUUID)
	if err != nil {
		return nil, fmt.Errorf("get edge: %w", err)
	}
	return edge, nil
}

func (uc *StoreUseCase) GetNeighbors(ctx context.Context, tenantID, nodeUUID string, depth int) ([]*graphiti.Node, []*graphiti.Edge, error) {
	if depth <= 0 {
		depth = 2
	}
	return uc.graph.GetNeighbors(ctx, tenantID, nodeUUID, depth)
}

// ─── Search ───────────────────────────────────────────────────────────────

// SearchUseCase handles graph search.
type SearchUseCase struct {
	episodes port.EpisodeRepository
	graph    GraphRepoInterface
	embedder port.EmbeddingService
}

// NewSearchUseCase creates a SearchUseCase.
func NewSearchUseCase(ep port.EpisodeRepository, g GraphRepoInterface, emb port.EmbeddingService) *SearchUseCase {
	return &SearchUseCase{episodes: ep, graph: g, embedder: emb}
}

// Search runs semantic, graph, or hybrid search based on query mode.
func (uc *SearchUseCase) Search(ctx context.Context, q graphiti.SearchQuery) (*graphiti.SearchResult, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}

	result := &graphiti.SearchResult{}

	switch q.Mode {
	case "semantic":
		episodes, err := uc.semanticSearch(ctx, q)
		if err != nil {
			return nil, err
		}
		result.Episodes = episodes

	case "graph":
		nodes, edges, err := uc.graph.QuerySubgraph(ctx, q.TenantID, q.Query)
		if err != nil {
			return nil, fmt.Errorf("graph search: %w", err)
		}
		result.Nodes = nodes
		result.Edges = edges

	default: // hybrid
		episodes, _ := uc.semanticSearch(ctx, q)
		nodes, edges, _ := uc.graph.QuerySubgraph(ctx, q.TenantID, q.Query)
		result.Episodes = episodes
		result.Nodes = nodes
		result.Edges = edges
	}

	return result, nil
}

func (uc *SearchUseCase) semanticSearch(ctx context.Context, q graphiti.SearchQuery) ([]*graphiti.Episode, error) {
	if uc.embedder != nil {
		emb, err := uc.embedder.Embed(ctx, q.Query)
		if err == nil {
			return uc.episodes.SemanticSearch(ctx, q.TenantID, emb, q.Limit)
		}
	}
	// Fallback: text search
	return uc.episodes.TextSearch(ctx, q.TenantID, q.Query, q.Limit)
}

// ─── Knowledge ────────────────────────────────────────────────────────────

// KnowledgeUseCase manages ontology and subgraph queries.
type KnowledgeUseCase struct {
	graph GraphRepoInterface
}

// NewKnowledgeUseCase creates a KnowledgeUseCase.
func NewKnowledgeUseCase(g GraphRepoInterface) *KnowledgeUseCase {
	return &KnowledgeUseCase{graph: g}
}

func (uc *KnowledgeUseCase) GetOntology(ctx context.Context, tenantID string) (*graphiti.Ontology, error) {
	return uc.graph.GetOntology(ctx, tenantID)
}

func (uc *KnowledgeUseCase) UpdateOntology(ctx context.Context, ontology *graphiti.Ontology) error {
	return uc.graph.UpdateOntology(ctx, ontology)
}

func (uc *KnowledgeUseCase) QuerySubgraph(ctx context.Context, tenantID, query string) (*graphiti.SearchResult, error) {
	nodes, edges, err := uc.graph.QuerySubgraph(ctx, tenantID, query)
	if err != nil {
		return nil, fmt.Errorf("subgraph query: %w", err)
	}
	return &graphiti.SearchResult{Nodes: nodes, Edges: edges}, nil
}
