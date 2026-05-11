package persistence

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/model"
	"vnp-memory/services/ov-fs/internal/domain/repository"
)

type vikingFSRepo struct {
	// db *sql.DB
}

var _ repository.FileRepository = (*vikingFSRepo)(nil)

func NewVikingFSRepo() repository.FileRepository {
	return &vikingFSRepo{}
}

func (r *vikingFSRepo) ReadFile(ctx context.Context, accountID, path string) (*model.File, error) {
	// SELECT * FROM ov_files WHERE account_id = $1 AND path = $2
	return &model.File{}, nil
}

func (r *vikingFSRepo) WriteFile(ctx context.Context, file *model.File, createParents bool) error {
	// INSERT INTO ov_files ... ON CONFLICT (account_id, path) DO UPDATE ...
	return nil
}

func (r *vikingFSRepo) DeleteFile(ctx context.Context, accountID, path string, recursive bool) error {
	// UPDATE ov_files SET deleted_at = NOW() WHERE account_id = $1 AND path = $2
	return nil
}

func (r *vikingFSRepo) MkDir(ctx context.Context, accountID, path string, createParents bool) error {
	return nil
}

func (r *vikingFSRepo) ListDir(ctx context.Context, accountID, path string, recursive, includeMetadata bool) ([]*model.DirEntry, error) {
	return nil, nil
}

func (r *vikingFSRepo) Tree(ctx context.Context, accountID, root string, opts model.TreeOptions) (*model.TreeNode, error) {
	return &model.TreeNode{}, nil
}

func (r *vikingFSRepo) Grep(ctx context.Context, accountID, pattern, path string, caseInsensitive bool, maxResults int32) ([]*repository.GrepMatch, error) {
	return nil, nil
}

func (r *vikingFSRepo) Glob(ctx context.Context, accountID, pattern, root string) ([]string, error) {
	return nil, nil
}

func (r *vikingFSRepo) Move(ctx context.Context, accountID, source, destination string, overwrite bool) error {
	return nil
}
