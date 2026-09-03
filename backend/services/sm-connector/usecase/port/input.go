package usecase

import "context"

// Enterprise Usecases for sm-connector
type SmConnectorUseCase interface {
	SyncData(ctx context.Context, req interface{}) (interface{}, error)
	ConfigureConnection(ctx context.Context, req interface{}) (interface{}, error)

}
