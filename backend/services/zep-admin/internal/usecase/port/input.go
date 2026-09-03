package usecase

import "context"

// Enterprise Usecases for zep-admin
type ZepAdminUseCase interface {
	CreateProject(ctx context.Context, req interface{}) (interface{}, error)
	ListProjects(ctx context.Context, req interface{}) (interface{}, error)

}
