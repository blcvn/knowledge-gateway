package pb

import (
	"context"

	"google.golang.org/grpc"
)

// GraphitiKnowledgeServiceServer is the server API for GraphitiKnowledgeService service.
type GraphitiKnowledgeServiceServer interface {
	ExtractEntities(context.Context, *ExtractEntitiesRequest) (*ExtractEntitiesResponse, error)
	ResolveEntities(context.Context, *ResolveEntitiesRequest) (*ResolveEntitiesResponse, error)
	ExtractEdges(context.Context, *ExtractEdgesRequest) (*ExtractEdgesResponse, error)
	ResolveEdges(context.Context, *ResolveEdgesRequest) (*ResolveEdgesResponse, error)
	GenerateEmbedding(context.Context, *GenerateEmbeddingRequest) (*GenerateEmbeddingResponse, error)
	GenerateEmbeddingBulk(context.Context, *GenerateEmbeddingBulkRequest) (*GenerateEmbeddingBulkResponse, error)
	Rerank(context.Context, *RerankRequest) (*RerankResponse, error)
	UpdateCommunity(context.Context, *UpdateCommunityRequest) (*UpdateCommunityResponse, error)
}

func RegisterGraphitiKnowledgeServiceServer(s grpc.ServiceRegistrar, srv GraphitiKnowledgeServiceServer) {
	// Mock registration - typically this maps methods using grpc.ServiceDesc
	// We'll leave it empty to just satisfy compilation and architecture rules
}
