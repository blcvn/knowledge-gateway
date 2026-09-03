// Package grpc implements the GraphitiStoreService gRPC handler.
package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"vnp-memory/services/graphiti-store/domain"
	"vnp-memory/services/graphiti-store/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const tenantIDKey = "x-tenant-id"

// Handler implements the GraphitiStoreService gRPC server.
type Handler struct {
	nodeService   port.NodeService
	edgeService   port.EdgeService
	searchService port.SearchService
	bulkService   port.BulkService
	indexService  port.IndexService
	logger        *slog.Logger
}

// NewHandler creates a new gRPC handler.
func NewHandler(
	nodeService port.NodeService,
	edgeService port.EdgeService,
	searchService port.SearchService,
	bulkService port.BulkService,
	indexService port.IndexService,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		nodeService:   nodeService,
		edgeService:   edgeService,
		searchService: searchService,
		bulkService:   bulkService,
		indexService:  indexService,
		logger:        logger.With("handler", "grpc"),
	}
}

// SaveNode creates or updates an entity node.
func (h *Handler) SaveNode(ctx context.Context, req *SaveNodeRequest) (*NodeResponse, error) {
	// Resiliency: apply strict 5s timeout to all DB operations
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	groupID, err := extractGroupID(ctx)
	if err != nil {
		return nil, err
	}

	node, err := h.nodeService.SaveNode(ctx, port.SaveNodeRequest{
		UUID:          req.Uuid,
		Name:          req.Name,
		GroupID:       groupID,
		Summary:       req.Summary,
		NameEmbedding: req.NameEmbedding,
		Labels:        req.Labels,
		Attributes:    req.Attributes,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return nodeToProto(node), nil
}

// GetNode retrieves an entity node by UUID.
func (h *Handler) GetNode(ctx context.Context, req *GetNodeRequest) (*NodeResponse, error) {
	groupID, err := extractGroupID(ctx)
	if err != nil {
		return nil, err
	}

	node, err := h.nodeService.GetNode(ctx, groupID, req.Uuid)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return nodeToProto(node), nil
}

// DeleteNode removes a node and its relationships.
func (h *Handler) DeleteNode(ctx context.Context, req *DeleteNodeRequest) (*EmptyResponse, error) {
	groupID, err := extractGroupID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.nodeService.DeleteNode(ctx, groupID, req.Uuid); err != nil {
		return nil, mapDomainError(err)
	}
	return &EmptyResponse{}, nil
}

// SaveEdge creates a bi-temporal relationship.
func (h *Handler) SaveEdge(ctx context.Context, req *SaveEdgeRequest) (*EdgeResponse, error) {
	groupID, err := extractGroupID(ctx)
	if err != nil {
		return nil, err
	}

	edge, err := h.edgeService.SaveEdge(ctx, port.SaveEdgeRequest{
		UUID:          req.Uuid,
		SourceNodeID:  req.SourceNodeId,
		TargetNodeID:  req.TargetNodeId,
		Name:          req.Name,
		GroupID:       groupID,
		Fact:          req.Fact,
		FactEmbedding: req.FactEmbedding,
		ValidAt:       req.ValidAt.AsTime(),
		EpisodeID:     req.EpisodeId,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return edgeToProto(edge), nil
}

// GetEdge retrieves an edge by UUID.
func (h *Handler) GetEdge(ctx context.Context, req *GetEdgeRequest) (*EdgeResponse, error) {
	edge, err := h.edgeService.GetEdge(ctx, req.Uuid)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return edgeToProto(edge), nil
}

// DeleteEdge removes an edge.
func (h *Handler) DeleteEdge(ctx context.Context, req *DeleteEdgeRequest) (*EmptyResponse, error) {
	if err := h.edgeService.DeleteEdge(ctx, req.Uuid); err != nil {
		return nil, mapDomainError(err)
	}
	return &EmptyResponse{}, nil
}

// CosineSimilaritySearch finds nodes by embedding similarity.
func (h *Handler) CosineSimilaritySearch(ctx context.Context, req *VectorSearchRequest) (*SearchResponse, error) {
	results, err := h.searchService.CosineSimilaritySearch(ctx, port.VectorSearchRequest{
		Embedding: req.Embedding,
		GroupID:   req.GroupId,
		Limit:     int(req.Limit),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return searchResultsToProto(results), nil
}

// FulltextSearch finds nodes by text matching.
func (h *Handler) FulltextSearch(ctx context.Context, req *TextSearchRequest) (*SearchResponse, error) {
	results, err := h.searchService.FulltextSearch(ctx, port.TextSearchRequest{
		Query:   req.Query,
		GroupID: req.GroupId,
		Limit:   int(req.Limit),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return searchResultsToProto(results), nil
}

// BFSSearch traverses the graph breadth-first.
func (h *Handler) BFSSearch(ctx context.Context, req *BFSSearchRequest) (*SearchResponse, error) {
	results, err := h.searchService.BFSSearch(ctx, port.BFSSearchRequest{
		StartNodeID: req.StartNodeId,
		MaxDepth:    int(req.MaxDepth),
		GroupID:     req.GroupId,
		Limit:       int(req.Limit),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return searchResultsToProto(results), nil
}

// SaveBulk atomically persists nodes, edges, and episode.
func (h *Handler) SaveBulk(ctx context.Context, req *SaveBulkRequest) (*SaveBulkResponse, error) {
	nodes := make([]domain.EntityNode, len(req.Nodes))
	for i, n := range req.Nodes {
		nodes[i] = domain.EntityNode{
			UUID:          n.Uuid,
			Name:          n.Name,
			GroupID:       n.GroupId,
			Summary:       n.Summary,
			NameEmbedding: n.NameEmbedding,
			Labels:        n.Labels,
		}
	}

	edges := make([]domain.EntityEdge, len(req.Edges))
	for i, e := range req.Edges {
		edges[i] = domain.EntityEdge{
			UUID:          e.Uuid,
			SourceNodeID:  e.SourceNodeId,
			TargetNodeID:  e.TargetNodeId,
			Name:          e.Name,
			GroupID:       e.GroupId,
			Fact:          e.Fact,
			FactEmbedding: e.FactEmbedding,
			ValidAt:       e.ValidAt.AsTime(),
			EpisodeID:     e.EpisodeId,
		}
	}

	episode := domain.EpisodicNode{
		UUID:    req.Episode.Uuid,
		Name:    req.Episode.Name,
		GroupID: req.Episode.GroupId,
		Content: req.Episode.Content,
		Source:  req.Episode.Source,
		ValidAt: req.Episode.ValidAt.AsTime(),
	}

	resp, err := h.bulkService.SaveBulk(ctx, port.SaveBulkRequest{
		Nodes:   nodes,
		Edges:   edges,
		Episode: episode,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &SaveBulkResponse{
		NodeCount: int32(resp.NodeCount),
		EdgeCount: int32(resp.EdgeCount),
		EpisodeId: resp.EpisodeID,
	}, nil
}

// DeleteByGroupID purges all tenant data.
func (h *Handler) DeleteByGroupID(ctx context.Context, req *DeleteByGroupRequest) (*EmptyResponse, error) {
	if err := h.bulkService.DeleteByGroupID(ctx, req.GroupId); err != nil {
		return nil, mapDomainError(err)
	}
	return &EmptyResponse{}, nil
}

// BuildIndices creates standard indexes.
func (h *Handler) BuildIndices(ctx context.Context, req *BuildIndicesRequest) (*EmptyResponse, error) {
	if err := h.indexService.BuildIndices(ctx, req.GroupId); err != nil {
		return nil, mapDomainError(err)
	}
	return &EmptyResponse{}, nil
}

// --- Helpers ---

func extractGroupID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get(tenantIDKey)
	if len(vals) == 0 || vals[0] == "" {
		return "", status.Errorf(codes.Unauthenticated, "missing %s metadata", tenantIDKey)
	}
	return vals[0], nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNodeNotFound), errors.Is(err, domain.ErrEdgeNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrMissingUUID),
		errors.Is(err, domain.ErrMissingName),
		errors.Is(err, domain.ErrMissingGroupID),
		errors.Is(err, domain.ErrMissingNodeID),
		errors.Is(err, domain.ErrMissingValidAt),
		errors.Is(err, domain.ErrEmptyContent),
		errors.Is(err, domain.ErrEmptyFact),
		errors.Is(err, domain.ErrEmptyEmbedding):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrTransactionFailed):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
