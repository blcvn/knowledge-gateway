package grpc

import (
\t"context"

\tpb "vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
\t"google.golang.org/grpc/codes"
\t"google.golang.org/grpc/status"
)

// UsecasePort defines the interface for the knowledge extraction business logic.
type UsecasePort interface {
\tExtractEntities(ctx context.Context, content string, previousEpisodes []string, entityTypes []string) ([]*pb.ExtractedEntity, error)
\tResolveEntities(ctx context.Context, entities []*pb.ExtractedEntity, groupID string) ([]*pb.Resolution, error)
}

// Handler implements the generated pb.GraphitiKnowledgeServiceServer interface.
type Handler struct {
\tpb.UnimplementedGraphitiKnowledgeServiceServer
\tusecase UsecasePort
}

// NewHandler returns a new instance of the gRPC Handler.
func NewHandler(uc UsecasePort) *Handler {
\treturn &Handler{
\t\tusecase: uc,
\t}
}

// ExtractEntities handles the extraction of entities from raw text content.
func (h *Handler) ExtractEntities(ctx context.Context, req *pb.ExtractEntitiesRequest) (*pb.ExtractEntitiesResponse, error) {
\tif req.Content == "" {
\t\treturn nil, status.Error(codes.InvalidArgument, "content cannot be empty")
\t}

\tentities, err := h.usecase.ExtractEntities(ctx, req.Content, req.PreviousEpisodes, req.EntityTypes)
\tif err != nil {
\t\treturn nil, status.Errorf(codes.Internal, "failed to extract entities: %v", err)
\t}

\treturn &pb.ExtractEntitiesResponse{
\t\tEntities: entities,
\t}, nil
}

// ResolveEntities handles deduplication and resolution of entities against existing graphs.
func (h *Handler) ResolveEntities(ctx context.Context, req *pb.ResolveEntitiesRequest) (*pb.ResolveEntitiesResponse, error) {
\tif len(req.ExtractedEntities) == 0 {
\t\treturn nil, status.Error(codes.InvalidArgument, "extracted_entities cannot be empty")
\t}
\tif req.GroupId == "" {
\t\treturn nil, status.Error(codes.InvalidArgument, "group_id is required")
\t}

\tresolutions, err := h.usecase.ResolveEntities(ctx, req.ExtractedEntities, req.GroupId)
\tif err != nil {
\t\treturn nil, status.Errorf(codes.Internal, "failed to resolve entities: %v", err)
\t}

\treturn &pb.ResolveEntitiesResponse{
\t\tResolutions: resolutions,
\t}, nil
}

// ... Additional endpoint implementations (ExtractEdges, Rerank, etc.) will follow ...
