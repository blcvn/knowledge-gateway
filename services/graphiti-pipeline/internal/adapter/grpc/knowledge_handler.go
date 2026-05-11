package grpc

import (
	"context"

	"graphiti-pipeline/internal/usecase/knowledge"
)

type KnowledgeHandler struct {
	// pb.UnimplementedGraphitiKnowledgeServiceServer
	usecase *knowledge.KnowledgeUsecase
}

func NewKnowledgeHandler(uc *knowledge.KnowledgeUsecase) *KnowledgeHandler {
	return &KnowledgeHandler{
		usecase: uc,
	}
}

func (h *KnowledgeHandler) ExtractEntities(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *KnowledgeHandler) ResolveEntities(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *KnowledgeHandler) ExtractEdges(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *KnowledgeHandler) ResolveEdges(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *KnowledgeHandler) GenerateEmbedding(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *KnowledgeHandler) Rerank(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *KnowledgeHandler) UpdateCommunity(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}
