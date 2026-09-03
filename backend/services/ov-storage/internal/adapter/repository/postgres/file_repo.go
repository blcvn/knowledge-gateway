package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/fs"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/port"
)

type fileRepository struct {
	// db *sql.DB
}

func NewFileRepository() port.FileRepository {
	return &fileRepository{}
}

func (r *fileRepository) Create(ctx context.Context, file *fs.File) error { return nil }
func (r *fileRepository) FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*fs.File, error) { return nil, nil }
func (r *fileRepository) Update(ctx context.Context, file *fs.File) error { return nil }
func (r *fileRepository) Delete(ctx context.Context, tenantID uuid.UUID, path string) error { return nil }
func (r *fileRepository) ListByDirectory(ctx context.Context, tenantID uuid.UUID, dirPath string) ([]fs.File, error) { return nil, nil }
