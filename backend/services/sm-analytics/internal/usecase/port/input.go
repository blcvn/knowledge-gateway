package usecase

import "context"

// Enterprise Usecases for sm-analytics
type SmAnalyticsUseCase interface {
	GenerateReport(ctx context.Context, req interface{}) (interface{}, error)
	TrackEvent(ctx context.Context, req interface{}) (interface{}, error)

}
