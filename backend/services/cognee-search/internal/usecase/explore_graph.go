package usecase

import (
	"context"
	"fmt"
	"vnp-memory/services/cognee-search/internal/usecase/dto"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type exploreGraphUseCase struct {
	graphSearcher port.GraphSearcher
}

func NewExploreGraphUseCase(graphSearcher port.GraphSearcher) port.ExploreGraphUseCase {
	return &exploreGraphUseCase{
		graphSearcher: graphSearcher,
	}
}

func (uc *exploreGraphUseCase) Execute(ctx context.Context, req dto.ExploreRequest) (*dto.ExploreResponse, error) {
	// Using the GraphSearcher port to find the neighborhood of a node
	// In a real implementation, the Cypher query would be constructed here based on depth
	
	// A simple mock of constructing a depth-based query:
	query := "MATCH (n)-[*1.." + fmt.Sprint(req.Depth) + "]-(m) WHERE id(n) = $nodeId RETURN m"
	params := map[string]interface{}{
		"nodeId": req.NodeID,
	}

	results, err := uc.graphSearcher.ExecuteCypher(ctx, query, params)
	if err != nil {
		return nil, err
	}

	return &dto.ExploreResponse{
		Results: results,
	}, nil
}
