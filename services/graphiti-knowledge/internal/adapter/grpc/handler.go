package grpc

import (
	"context"

	"vnp-memory/services/graphiti-knowledge/internal/adapter/grpc/pb"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type Handler struct {
	extractEntities port.ExtractEntitiesUseCase
	resolveEntities port.ResolveEntitiesUseCase
	extractEdges    port.ExtractEdgesUseCase
	resolveEdges    port.ResolveEdgesUseCase
	genEmbed        port.GenerateEmbeddingUseCase
	updateCommunity port.UpdateCommunityUseCase
	rerank          port.RerankUseCase
}

func NewHandler(
	ee port.ExtractEntitiesUseCase,
	re port.ResolveEntitiesUseCase,
	exEdges port.ExtractEdgesUseCase,
	rsEdges port.ResolveEdgesUseCase,
	genEmbed port.GenerateEmbeddingUseCase,
	upComm port.UpdateCommunityUseCase,
	rerank port.RerankUseCase,
) *Handler {
	return &Handler{
		extractEntities: ee,
		resolveEntities: re,
		extractEdges:    exEdges,
		resolveEdges:    rsEdges,
		genEmbed:        genEmbed,
		updateCommunity: upComm,
		rerank:          rerank,
	}
}

func (h *Handler) ExtractEntities(ctx context.Context, req *pb.ExtractEntitiesRequest) (*pb.ExtractEntitiesResponse, error) {
	_, _, err := h.extractEntities.Execute(ctx, dto.ExtractEntitiesRequest{
		Content:          req.Content,
		PreviousEpisodes: req.PreviousEpisodes,
		EntityTypes:      req.EntityTypes,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ExtractEntitiesResponse{Entities: []*pb.ExtractedEntity{}}, nil
}

func (h *Handler) ResolveEntities(ctx context.Context, req *pb.ResolveEntitiesRequest) (*pb.ResolveEntitiesResponse, error) {
	_, err := h.resolveEntities.Execute(ctx, dto.ResolveEntitiesRequest{
		GroupID: req.GroupId,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ResolveEntitiesResponse{Resolutions: []*pb.Resolution{}}, nil
}

func (h *Handler) ExtractEdges(ctx context.Context, req *pb.ExtractEdgesRequest) (*pb.ExtractEdgesResponse, error) {
	_, _, err := h.extractEdges.Execute(ctx, dto.ExtractEdgesRequest{})
	if err != nil {
		return nil, err
	}
	return &pb.ExtractEdgesResponse{}, nil
}

func (h *Handler) ResolveEdges(ctx context.Context, req *pb.ResolveEdgesRequest) (*pb.ResolveEdgesResponse, error) {
	_, err := h.resolveEdges.Execute(ctx, dto.ResolveEdgesRequest{})
	if err != nil {
		return nil, err
	}
	return &pb.ResolveEdgesResponse{}, nil
}

func (h *Handler) GenerateEmbedding(ctx context.Context, req *pb.GenerateEmbeddingRequest) (*pb.GenerateEmbeddingResponse, error) {
	_, err := h.genEmbed.Execute(ctx, dto.GenerateEmbeddingRequest{})
	if err != nil {
		return nil, err
	}
	return &pb.GenerateEmbeddingResponse{}, nil
}

func (h *Handler) GenerateEmbeddingBulk(ctx context.Context, req *pb.GenerateEmbeddingBulkRequest) (*pb.GenerateEmbeddingBulkResponse, error) {
	return &pb.GenerateEmbeddingBulkResponse{}, nil
}

func (h *Handler) Rerank(ctx context.Context, req *pb.RerankRequest) (*pb.RerankResponse, error) {
	_, err := h.rerank.Execute(ctx, dto.RerankRequest{})
	if err != nil {
		return nil, err
	}
	return &pb.RerankResponse{}, nil
}

func (h *Handler) UpdateCommunity(ctx context.Context, req *pb.UpdateCommunityRequest) (*pb.UpdateCommunityResponse, error) {
	_, err := h.updateCommunity.Execute(ctx, dto.UpdateCommunityRequest{})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateCommunityResponse{}, nil
}
