package usecase

import (
	"context"

	"github.com/vnp-memory/services/observe-service/internal/domain"
	"github.com/vnp-memory/services/observe-service/internal/observe"
	"github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type ObserveUseCase struct {
	pipeline    *observe.Pipeline
	sessionRepo port.ISessionRepo
	obsRepo     port.IObservationRepo
}

func (uc *ObserveUseCase) Execute(ctx context.Context, req observe.ObserveRequest) (*observe.ObserveResponse, error) {
	// Validate session exists
	if _, err := uc.sessionRepo.GetByID(ctx, req.SessionID); err != nil {
		return nil, err
	}
	return uc.pipeline.Execute(ctx, req)
}

func (uc *ObserveUseCase) GetObservations(ctx context.Context, sessionID string, limit, offset int) ([]domain.CompressedObservation, error) {
	return uc.obsRepo.ListCompressed(ctx, sessionID, limit, offset)
}
