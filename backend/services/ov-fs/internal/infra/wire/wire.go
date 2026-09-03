package wire

import (
	"vnp-memory/services/ov-fs/internal/adapter/client"
	"vnp-memory/services/ov-fs/internal/adapter/event"
	"vnp-memory/services/ov-fs/internal/adapter/grpc"
	"vnp-memory/services/ov-fs/internal/infra/persistence"
	"vnp-memory/services/ov-fs/internal/usecase"
)

func InitializeHandler() *grpc.OvFsHandler {
	// fileUC := usecase.NewFileUseCase(persistence.NewVikingFSRepo(), persistence.NewAbstractRepo(), client.NewCryptoClient(), event.NewNatsPublisher())
	// dirUC := usecase.NewDirUseCase(persistence.NewVikingFSRepo())
	// searchUC := usecase.NewSearchUseCase(persistence.NewVikingFSRepo())
	// moveUC := usecase.NewMoveUseCase(persistence.NewVikingFSRepo())
	// relationUC := usecase.NewRelationUseCase(persistence.NewRelationRepo())

	// return grpc.NewOvFsHandler(fileUC, dirUC, searchUC, moveUC, relationUC)
	return nil
}
