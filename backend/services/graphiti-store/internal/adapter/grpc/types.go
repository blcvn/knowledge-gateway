package grpc

import (
	"vnp-memory/services/graphiti-store/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Proto-like request/response types ---
// These will be replaced by protoc-generated types when proto files are compiled.
// For now they serve as compile-verifiable contracts.

// SaveNodeRequest is the gRPC request for SaveNode.
type SaveNodeRequest struct {
	Uuid          string            `json:"uuid"`
	Name          string            `json:"name"`
	Summary       string            `json:"summary,omitempty"`
	NameEmbedding []float32         `json:"name_embedding,omitempty"`
	Labels        []string          `json:"labels,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// GetNodeRequest is the gRPC request for GetNode.
type GetNodeRequest struct {
	Uuid string `json:"uuid"`
}

// DeleteNodeRequest is the gRPC request for DeleteNode.
type DeleteNodeRequest struct {
	Uuid string `json:"uuid"`
}

// NodeResponse is the gRPC response for node operations.
type NodeResponse struct {
	Uuid      string            `json:"uuid"`
	Name      string            `json:"name"`
	GroupId   string            `json:"group_id"`
	Summary   string            `json:"summary"`
	Labels    []string          `json:"labels"`
	CreatedAt *timestamppb.Timestamp `json:"created_at"`
	UpdatedAt *timestamppb.Timestamp `json:"updated_at"`
}

// SaveEdgeRequest is the gRPC request for SaveEdge.
type SaveEdgeRequest struct {
	Uuid          string                 `json:"uuid"`
	SourceNodeId  string                 `json:"source_node_id"`
	TargetNodeId  string                 `json:"target_node_id"`
	Name          string                 `json:"name"`
	Fact          string                 `json:"fact"`
	FactEmbedding []float32              `json:"fact_embedding,omitempty"`
	ValidAt       *timestamppb.Timestamp `json:"valid_at"`
	EpisodeId     string                 `json:"episode_id"`
}

// GetEdgeRequest is the gRPC request for GetEdge.
type GetEdgeRequest struct {
	Uuid string `json:"uuid"`
}

// DeleteEdgeRequest is the gRPC request for DeleteEdge.
type DeleteEdgeRequest struct {
	Uuid string `json:"uuid"`
}

// EdgeResponse is the gRPC response for edge operations.
type EdgeResponse struct {
	Uuid         string                 `json:"uuid"`
	SourceNodeId string                 `json:"source_node_id"`
	TargetNodeId string                 `json:"target_node_id"`
	Name         string                 `json:"name"`
	Fact         string                 `json:"fact"`
	GroupId      string                 `json:"group_id"`
	ValidAt      *timestamppb.Timestamp `json:"valid_at"`
	InvalidAt    *timestamppb.Timestamp `json:"invalid_at,omitempty"`
	CreatedAt    *timestamppb.Timestamp `json:"created_at"`
}

// VectorSearchRequest for cosine similarity search.
type VectorSearchRequest struct {
	Embedding []float32 `json:"embedding"`
	GroupId   string    `json:"group_id"`
	Limit     int32     `json:"limit"`
}

// TextSearchRequest for fulltext search.
type TextSearchRequest struct {
	Query   string `json:"query"`
	GroupId string `json:"group_id"`
	Limit   int32  `json:"limit"`
}

// BFSSearchRequest for graph traversal.
type BFSSearchRequest struct {
	StartNodeId string `json:"start_node_id"`
	MaxDepth    int32  `json:"max_depth"`
	GroupId     string `json:"group_id"`
	Limit       int32  `json:"limit"`
}

// SearchResponse wraps search results.
type SearchResponse struct {
	Results []*SearchResultProto `json:"results"`
}

// SearchResultProto is a single search result.
type SearchResultProto struct {
	Uuid      string  `json:"uuid"`
	NodeLabel string  `json:"node_label"`
	Name      string  `json:"name"`
	Summary   string  `json:"summary"`
	Fact      string  `json:"fact"`
	Score     float64 `json:"score"`
	Distance  int32   `json:"distance"`
	GroupId   string  `json:"group_id"`
}

// SaveBulkRequest for atomic batch operations.
type SaveBulkRequest struct {
	Nodes   []*BulkNodeProto    `json:"nodes"`
	Edges   []*BulkEdgeProto    `json:"edges"`
	Episode *BulkEpisodeProto   `json:"episode"`
}

// BulkNodeProto represents a node in a bulk request.
type BulkNodeProto struct {
	Uuid          string    `json:"uuid"`
	Name          string    `json:"name"`
	GroupId       string    `json:"group_id"`
	Summary       string    `json:"summary"`
	NameEmbedding []float32 `json:"name_embedding"`
	Labels        []string  `json:"labels"`
}

// BulkEdgeProto represents an edge in a bulk request.
type BulkEdgeProto struct {
	Uuid          string                 `json:"uuid"`
	SourceNodeId  string                 `json:"source_node_id"`
	TargetNodeId  string                 `json:"target_node_id"`
	Name          string                 `json:"name"`
	GroupId       string                 `json:"group_id"`
	Fact          string                 `json:"fact"`
	FactEmbedding []float32              `json:"fact_embedding"`
	ValidAt       *timestamppb.Timestamp `json:"valid_at"`
	EpisodeId     string                 `json:"episode_id"`
}

// BulkEpisodeProto represents an episode in a bulk request.
type BulkEpisodeProto struct {
	Uuid    string                 `json:"uuid"`
	Name    string                 `json:"name"`
	GroupId string                 `json:"group_id"`
	Content string                 `json:"content"`
	Source  string                 `json:"source"`
	ValidAt *timestamppb.Timestamp `json:"valid_at"`
}

// SaveBulkResponse returns bulk operation counts.
type SaveBulkResponse struct {
	NodeCount int32  `json:"node_count"`
	EdgeCount int32  `json:"edge_count"`
	EpisodeId string `json:"episode_id"`
}

// DeleteByGroupRequest for tenant purge.
type DeleteByGroupRequest struct {
	GroupId string `json:"group_id"`
}

// BuildIndicesRequest for index management.
type BuildIndicesRequest struct {
	GroupId string `json:"group_id"`
}

// EmptyResponse for void operations.
type EmptyResponse struct{}

// --- Mapping functions ---

func nodeToProto(n *domain.EntityNode) *NodeResponse {
	return &NodeResponse{
		Uuid:      n.UUID,
		Name:      n.Name,
		GroupId:   n.GroupID,
		Summary:   n.Summary,
		Labels:    n.Labels,
		CreatedAt: timestamppb.New(n.CreatedAt),
		UpdatedAt: timestamppb.New(n.UpdatedAt),
	}
}

func edgeToProto(e *domain.EntityEdge) *EdgeResponse {
	resp := &EdgeResponse{
		Uuid:         e.UUID,
		SourceNodeId: e.SourceNodeID,
		TargetNodeId: e.TargetNodeID,
		Name:         e.Name,
		Fact:         e.Fact,
		GroupId:      e.GroupID,
		ValidAt:      timestamppb.New(e.ValidAt),
		CreatedAt:    timestamppb.New(e.CreatedAt),
	}
	if e.InvalidAt != nil {
		resp.InvalidAt = timestamppb.New(*e.InvalidAt)
	}
	return resp
}

func searchResultsToProto(results []domain.SearchResult) *SearchResponse {
	protos := make([]*SearchResultProto, len(results))
	for i, r := range results {
		protos[i] = &SearchResultProto{
			Uuid:      r.UUID,
			NodeLabel: r.NodeLabel,
			Name:      r.Name,
			Summary:   r.Summary,
			Fact:      r.Fact,
			Score:     r.Score,
			Distance:  int32(r.Distance),
			GroupId:   r.GroupID,
		}
	}
	return &SearchResponse{Results: protos}
}
