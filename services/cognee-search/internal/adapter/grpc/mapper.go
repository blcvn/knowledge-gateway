package grpc

import (
	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/infrastructure/grpc/pb"
	"vnp-memory/services/cognee-search/internal/usecase/dto"
)

func mapToSearchRequestDTO(req *pb.SearchRequest) dto.SearchRequest {
	return dto.SearchRequest{
		Query:      req.Query,
		Strategies: req.Strategies,
		TopK:       int(req.TopK),
		Rerank:     req.Rerank,
		Filters:    domain.SearchFilters{}, // In a real app, map the actual filters
	}
}

func mapToSearchResponsePB(res *dto.SearchResponse) *pb.SearchResponse {
	var pbResults []*pb.SearchResult
	for _, r := range res.Results {
		pbResults = append(pbResults, &pb.SearchResult{
			Id:      r.ID,
			Content: r.Content,
			Score:   float32(r.Score),
		})
	}
	return &pb.SearchResponse{
		Results: pbResults,
	}
}

func mapToRAGRequestDTO(req *pb.RAGRequest) dto.RAGRequest {
	return dto.RAGRequest{
		Query:      req.Query,
		Strategies: req.Strategies,
		TopK:       int(req.TopK),
	}
}

func mapToRAGResponsePB(res *dto.RAGResponse) *pb.RAGResponse {
	var pbSources []*pb.SearchResult
	for _, r := range res.Sources {
		pbSources = append(pbSources, &pb.SearchResult{
			Id:      r.ID,
			Content: r.Content,
			Score:   float32(r.Score),
		})
	}
	return &pb.RAGResponse{
		Answer:  res.Answer,
		Sources: pbSources,
	}
}
