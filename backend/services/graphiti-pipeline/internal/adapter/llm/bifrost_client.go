package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
	"graphiti-pipeline/internal/domain/knowledge"
	"graphiti-pipeline/internal/infra/config"
	"graphiti-pipeline/internal/usecase/port"
)

type BifrostClient struct {
	client   *http.Client
	endpoint string
	cb       *gobreaker.CircuitBreaker
	logger   *zap.Logger
}

func NewBifrostClient(cfg config.Config, logger *zap.Logger) port.LLMClient {
	st := gobreaker.Settings{
		Name:        "Bifrost-LLM",
		MaxRequests: 5,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
	}
	return &BifrostClient{
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: cfg.LLM.Endpoint,
		cb:       gobreaker.NewCircuitBreaker(st),
		logger:   logger,
	}
}

func (c *BifrostClient) ExtractEntities(ctx context.Context, content string) ([]knowledge.ExtractedEntity, error) {
	reqBody, _ := json.Marshal(map[string]string{"prompt": "Extract entities from: " + content})

	result, err := c.cb.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		resp, err := c.client.Do(req)
		if err != nil {
			c.logger.Error("LLM API call failed", zap.Error(err))
			return nil, fmt.Errorf("LLM API call failed: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("LLM API returned status: %d", resp.StatusCode)
		}

		return []knowledge.ExtractedEntity{{Name: "MockEntity", Label: "MockLabel", Summary: "MockSummary"}}, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]knowledge.ExtractedEntity), nil
}

func (c *BifrostClient) ResolveEntities(ctx context.Context, existing []knowledge.ExtractedEntity, newEntities []knowledge.ExtractedEntity) ([]knowledge.Resolution, error) {
	return []knowledge.Resolution{{Decision: knowledge.DecisionCreate}}, nil
}

func (c *BifrostClient) ExtractEdges(ctx context.Context, content string, entities []knowledge.ExtractedEntity) ([]knowledge.ExtractedEdge, error) {
	return []knowledge.ExtractedEdge{{Source: "A", Target: "B", Relation: "MockRelation"}}, nil
}

func (c *BifrostClient) ResolveEdges(ctx context.Context, existing []knowledge.ExtractedEdge, newEdges []knowledge.ExtractedEdge) ([]knowledge.ExtractedEdge, error) {
	return newEdges, nil
}

func (c *BifrostClient) UpdateCommunity(ctx context.Context, community knowledge.CommunityNode) (knowledge.CommunityNode, error) {
	return community, nil
}
