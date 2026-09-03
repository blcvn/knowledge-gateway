package wire

import (
	"vnp-memory/services/graphiti-knowledge/adapter/client"
	"vnp-memory/services/graphiti-knowledge/adapter/embedder"
	grpcAdapter "vnp-memory/services/graphiti-knowledge/adapter/grpc"
	"vnp-memory/services/graphiti-knowledge/adapter/llm"
	"vnp-memory/services/graphiti-knowledge/infra/config"
	"vnp-memory/services/graphiti-knowledge/usecase"
	"vnp-memory/services/graphiti-knowledge/usecase/port"
)

func provideBifrostClient(cfg *config.Config) port.LLMClient {
	return llm.NewBifrostClient(cfg.LLMProvider, cfg.LLMAPIKey, 10)
}

func provideBifrostEmbedder(cfg *config.Config) port.EmbedderClient {
	return embedder.NewBifrostEmbedder(cfg.EmbedderURL, cfg.EmbedderAPIKey, 1536)
}

func provideParser() interface{ ParseJSON(string) string } {
	return llm.NewResponseParser()
}

// InitializeHandler manually wires up the application dependencies.
func InitializeHandler(cfg *config.Config) *grpcAdapter.Handler {
	llmClient := provideBifrostClient(cfg)
	embedderClient := provideBifrostEmbedder(cfg)
	promptReg := llm.NewPromptRegistry()
	parser := provideParser()
	storeClient := client.NewStoreClient()

	extractEntities := usecase.NewExtractEntitiesUseCase(llmClient, promptReg, parser)
	resolveEntities := usecase.NewResolveEntitiesUseCase(llmClient, embedderClient, storeClient, promptReg, parser)
	extractEdges := usecase.NewExtractEdgesUseCase(llmClient, promptReg, parser)
	resolveEdges := usecase.NewResolveEdgesUseCase(llmClient, promptReg, parser)
	genEmbed := usecase.NewGenerateEmbeddingUseCase(embedderClient)
	updateCommunity := usecase.NewUpdateCommunityUseCase(llmClient, promptReg, parser)
	rerank := usecase.NewRerankUseCase()

	return grpcAdapter.NewHandler(
		extractEntities,
		resolveEntities,
		extractEdges,
		resolveEdges,
		genEmbed,
		updateCommunity,
		rerank,
	)
}
