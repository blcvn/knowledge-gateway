package port

import (
	"context"

	"vnp-memory/services/ov-fs/internal/usecase/dto"
)

type FileUseCase interface {
	ReadFile(ctx context.Context, req dto.ReadFileRequest) (*dto.ReadFileResponse, error)
	WriteFile(ctx context.Context, req dto.WriteFileRequest) (*dto.WriteFileResponse, error)
	DeleteFile(ctx context.Context, req dto.DeleteFileRequest) error
}

type DirUseCase interface {
	MkDir(ctx context.Context, req dto.MkDirRequest) error
	ListDir(ctx context.Context, req dto.ListDirRequest) (*dto.ListDirResponse, error)
	Tree(ctx context.Context, req dto.TreeRequest) (*dto.TreeResponse, error)
}

type SearchUseCase interface {
	Grep(ctx context.Context, req dto.GrepRequest) (*dto.GrepResponse, error)
	Glob(ctx context.Context, req dto.GlobRequest) (*dto.GlobResponse, error)
}

type MoveUseCase interface {
	Move(ctx context.Context, req dto.MoveRequest) error
}

type RelationUseCase interface {
	GetRelations(ctx context.Context, req dto.GetRelationsRequest) (*dto.RelationsResponse, error)
	AddRelation(ctx context.Context, req dto.AddRelationRequest) error
}
