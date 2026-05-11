package usecase

import (
	"context"
	"fmt"
	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/dto"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type ragCompleteUseCase struct {
	searchUseCase port.SearchUseCase
	llmClient     port.LLMClient
}

func NewRAGCompleteUseCase(searchUseCase port.SearchUseCase, llmClient port.LLMClient) port.RAGCompleteUseCase {
	return &ragCompleteUseCase{
		searchUseCase: searchUseCase,
		llmClient:     llmClient,
	}
}

func (uc *ragCompleteUseCase) Execute(ctx context.Context, req dto.RAGRequest) (*dto.RAGResponse, error) {
	if req.Query == "" {
		return nil, domain.ErrEmptyQuery
	}

	// Step 1: Perform search
	searchReq := dto.SearchRequest{
		Query:      req.Query,
		Strategies: req.Strategies,
		TopK:       req.TopK,
		Rerank:     true, // Default to reranking for better RAG quality
		Filters:    req.Filters,
	}

	searchRes, err := uc.searchUseCase.Execute(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	// Step 2: Construct prompt with sources
	contextText := ""
	for i, res := range searchRes.Results {
		contextText += fmt.Sprintf("Source %d:\n%s\n\n", i+1, res.Content)
	}

	prompt := fmt.Sprintf("Given the following context, answer the question.\n\nContext:\n%s\nQuestion: %s\nAnswer:", contextText, req.Query)

	// Step 3: Call LLM
	answer, err := uc.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &dto.RAGResponse{
		Answer:  answer,
		Sources: searchRes.Results,
	}, nil
}
