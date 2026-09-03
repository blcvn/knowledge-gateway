package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"vnp-memory/services/graphiti-knowledge/domain"
	"vnp-memory/services/graphiti-knowledge/usecase/port"
)

type bifrostClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	semaphore  chan struct{}
	cb         *gobreaker.CircuitBreaker
	tracer     trace.Tracer
}

func NewBifrostClient(baseURL string, apiKey string, maxConcurrent int) port.LLMClient {
	st := gobreaker.Settings{
		Name:        "bifrost-llm",
		MaxRequests: uint32(maxConcurrent),
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	}
	return &bifrostClient{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
		semaphore:  make(chan struct{}, maxConcurrent),
		cb:         gobreaker.NewCircuitBreaker(st),
		tracer:     otel.Tracer("bifrost-llm-client"),
	}
}

type bifrostRequest struct {
	Model    string `json:"model"`
	Messages []msg  `json:"messages"`
}
type msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bifrostResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *bifrostClient) Complete(ctx context.Context, prompt string, model string) (string, domain.TokenUsage, error) {
	ctx, span := c.tracer.Start(ctx, "Complete")
	defer span.End()
	span.SetAttributes(attribute.String("llm.model", model))

	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	res, err := c.cb.Execute(func() (interface{}, error) {
		var respData bifrostResponse

		err := retry.Do(
			func() error {
				reqBody := bifrostRequest{
					Model: model,
					Messages: []msg{
						{Role: "user", Content: prompt},
					},
				}
				bodyBytes, _ := json.Marshal(reqBody)

				req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(bodyBytes))
				if err != nil {
					return err
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+c.apiKey)

				resp, err := c.httpClient.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode >= 400 {
					return fmt.Errorf("HTTP error: %d", resp.StatusCode)
				}

				respBytes, _ := io.ReadAll(resp.Body)
				return json.Unmarshal(respBytes, &respData)
			},
			retry.Attempts(3),
			retry.DelayType(retry.BackOffDelay),
			retry.Delay(100*time.Millisecond),
			retry.Context(ctx),
		)

		if err != nil {
			return nil, err
		}
		return respData, nil
	})

	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	data := res.(bifrostResponse)
	content := ""
	if len(data.Choices) > 0 {
		content = data.Choices[0].Message.Content
	}

	usage := domain.TokenUsage{
		PromptTokens:     data.Usage.PromptTokens,
		CompletionTokens: data.Usage.CompletionTokens,
		TotalTokens:      data.Usage.TotalTokens,
		Model:            model,
	}

	return content, usage, nil
}
