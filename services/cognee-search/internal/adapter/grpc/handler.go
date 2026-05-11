package grpc

import (
	"context"

	"vnp-memory/services/cognee-search/internal/infrastructure/grpc/pb"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type handler struct {
	searchUseCase port.SearchUseCase
	ragUseCase    port.RAGCompleteUseCase
}

func NewHandler(searchUseCase port.SearchUseCase, ragUseCase port.RAGCompleteUseCase) pb.CogneeSearchServiceServer {
	return &handler{
		searchUseCase: searchUseCase,
		ragUseCase:    ragUseCase,
	}
}

func (h *handler) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	dtoReq := mapToSearchRequestDTO(req)
	
	res, err := h.searchUseCase.Execute(ctx, dtoReq)
	if err != nil {
		return nil, err
	}

	return mapToSearchResponsePB(res), nil
}

func (h *handler) RAGComplete(ctx context.Context, req *pb.RAGRequest) (*pb.RAGResponse, error) {
	dtoReq := mapToRAGRequestDTO(req)
	
	res, err := h.ragUseCase.Execute(ctx, dtoReq)
	if err != nil {
		return nil, err
	}

	return mapToRAGResponsePB(res), nil
}

func (h *handler) GetChunks(ctx context.Context, req *pb.GetChunksRequest) (*pb.ChunksResponse, error) {
	// Simple implementation
	return &pb.ChunksResponse{}, nil
}
