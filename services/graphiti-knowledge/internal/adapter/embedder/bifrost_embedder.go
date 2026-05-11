package embedder

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
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

const EMBEDDER_BATCH_SIZE = 100

var ErrInvalidEmbeddingDimension = fmt.Errorf("invalid embedding dimension")

type bifrostEmbedder struct {
	httpClient        *http.Client
	baseURL           string
	apiKey            string
	expectedDimension int
	cb                *gobreaker.CircuitBreaker
	tracer            trace.Tracer
}

func NewBifrostEmbedder(baseURL, apiKey string, expectedDimension int) port.EmbedderClient {
	st := gobreaker.Settings{
		Name:        "bifrost-embedder",
		MaxRequests: 10,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
	}
	return &bifrostEmbedder{
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		baseURL:           baseURL,
		apiKey:            apiKey,
		expectedDimension: expectedDimension,
		cb:                gobreaker.NewCircuitBreaker(st),
		tracer:            otel.Tracer("bifrost-embedder-client"),
	}
}

type embedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (e *bifrostEmbedder) Embed(ctx context.Context, text string, model string) (domain.EmbeddingVector, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text}, model)
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return vecs[0], nil
}

func (e *bifrostEmbedder) EmbedBatch(ctx context.Context, texts []string, model string) ([]domain.EmbeddingVector, error) {
	ctx, span := e.tracer.Start(ctx, "EmbedBatch")
	defer span.End()
	span.SetAttributes(attribute.Int("batch.size", len(texts)))

	var results []domain.EmbeddingVector

	for i := 0; i < len(texts); i += EMBEDDER_BATCH_SIZE {
		end := i + EMBEDDER_BATCH_SIZE
		if end > len(texts) {
			end = len(texts)
		}
		chunk := texts[i:end]

		res, err := e.cb.Execute(func() (interface{}, error) {
			var respData embedResponse
			err := retry.Do(
				func() error {
					reqBody := embedRequest{Input: chunk, Model: model}
					bodyBytes, _ := json.Marshal(reqBody)

					req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL, bytes.NewBuffer(bodyBytes))
					if err != nil {
						return err
					}
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+e.apiKey)

					resp, err := e.httpClient.Do(req)
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
			return nil, err
		}

		data := res.(embedResponse)
		for _, d := range data.Data {
			vec := domain.EmbeddingVector(d.Embedding)
			if err := vec.Validate(e.expectedDimension); err != nil {
				return nil, ErrInvalidEmbeddingDimension
			}
			results = append(results, vec)
		}
	}

	return results, nil
}
