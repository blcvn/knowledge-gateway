package grpc

import (
	"context"

	"graphiti-pipeline/internal/usecase/ingest"
)

type IngestionHandler struct {
	// pb.UnimplementedGraphitiIngestionServiceServer
	usecase *ingest.IngestEpisodeUseCase
	bulk    *ingest.BulkIngestUseCase
}

func NewIngestionHandler(uc *ingest.IngestEpisodeUseCase, bulk *ingest.BulkIngestUseCase) *IngestionHandler {
	return &IngestionHandler{
		usecase: uc,
		bulk:    bulk,
	}
}

func (h *IngestionHandler) IngestEpisode(ctx context.Context, req interface{}) (interface{}, error) {
	// episode := ToDomainEpisode(req)
	// err := h.usecase.Execute(ctx, episode)
	return nil, nil
}

func (h *IngestionHandler) BulkIngest(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *IngestionHandler) GetEpisodeStatus(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *IngestionHandler) ListEpisodes(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *IngestionHandler) RemoveEpisode(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}
