package usecase

import "context"

// Enterprise Usecases for sm-engine
type SmEngineUseCase interface {
	CalculateRetention(ctx context.Context, req interface{}) (interface{}, error)
	ScheduleReview(ctx context.Context, req interface{}) (interface{}, error)

}
