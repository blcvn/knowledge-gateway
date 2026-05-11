package repository

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/model"
)

type GrepMatch struct {
	Path       string
	LineNumber int
	Content    string
	Score      float64
}

type FileRepository interface {
	ReadFile(ctx context.Context, accountID, path string) (*model.File, error)
	WriteFile(ctx context.Context, file *model.File, createParents bool) error
	DeleteFile(ctx context.Context, accountID, path string, recursive bool) error
	MkDir(ctx context.Context, accountID, path string, createParents bool) error
	ListDir(ctx context.Context, accountID, path string, recursive, includeMetadata bool) ([]*model.DirEntry, error)
	Tree(ctx context.Context, accountID, root string, opts model.TreeOptions) (*model.TreeNode, error)
	Grep(ctx context.Context, accountID, pattern, path string, caseInsensitive bool, maxResults int32) ([]*GrepMatch, error)
	Glob(ctx context.Context, accountID, pattern, root string) ([]string, error)
	Move(ctx context.Context, accountID, source, destination string, overwrite bool) error
}
