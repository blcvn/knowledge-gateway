package knowledge

import (
	"context"

	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/domain/knowledge"
	"graphiti-pipeline/internal/usecase/port"
)

type KnowledgeUsecase struct {
	llm      port.LLMClient
	embedder port.EmbedderClient
}

func NewKnowledgeUsecase(llm port.LLMClient, embedder port.EmbedderClient) *KnowledgeUsecase {
	return &KnowledgeUsecase{
		llm:      llm,
		embedder: embedder,
	}
}

func (u *KnowledgeUsecase) ExtractEntities(ctx context.Context, episode ingestion.Episode) error {
	entities, err := u.llm.ExtractEntities(ctx, episode.Content)
	if err != nil {
		return err
	}
	_ = entities // Store context
	return nil
}

func (u *KnowledgeUsecase) ResolveEntities(ctx context.Context, episode ingestion.Episode) error {
	_, err := u.llm.ResolveEntities(ctx, nil, nil)
	return err
}

func (u *KnowledgeUsecase) ExtractEdges(ctx context.Context, episode ingestion.Episode) error {
	_, err := u.llm.ExtractEdges(ctx, episode.Content, nil)
	return err
}

func (u *KnowledgeUsecase) ResolveEdges(ctx context.Context, episode ingestion.Episode) error {
	_, err := u.llm.ResolveEdges(ctx, nil, nil)
	return err
}

func (u *KnowledgeUsecase) GenerateEmbeddings(ctx context.Context, episode ingestion.Episode) error {
	_, err := u.embedder.GenerateEmbedding(ctx, knowledge.EmbeddingRequest{Text: episode.Content, Model: "default"})
	return err
}

func (u *KnowledgeUsecase) UpdateCommunity(ctx context.Context, episode ingestion.Episode) error {
	return nil
}
