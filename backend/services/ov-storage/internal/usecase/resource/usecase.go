package resource

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/resource"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/port"
)

type resourceUseCase struct {
	resRepo port.ResourceRepository
	pub     port.EventPublisher
	fsUC    port.FsUseCase
}

// NewResourceUseCase creates a new instance of ResourceUseCase.
func NewResourceUseCase(resRepo port.ResourceRepository, pub port.EventPublisher, fsUC port.FsUseCase) port.ResourceUseCase {
	return &resourceUseCase{
		resRepo: resRepo,
		pub:     pub,
		fsUC:    fsUC,
	}
}

func (u *resourceUseCase) ParseResource(ctx context.Context, tenantID uuid.UUID, path string, parserType resource.ParserType) (*resource.Resource, error) {
	// Call external parser implementation (tree-sitter, etc)
	res := &resource.Resource{
		ID:         uuid.New(),
		TenantID:   tenantID,
		SourcePath: path,
		ParserType: parserType,
		CreatedAt:  time.Now(),
	}

	err := u.resRepo.Create(ctx, res)
	if err != nil {
		return nil, err
	}

	u.pub.PublishResourceParsed(ctx, tenantID, res.ID)

	return res, nil
}

func (u *resourceUseCase) GetResource(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	return u.resRepo.FindByID(ctx, id)
}

func (u *resourceUseCase) WatchPath(ctx context.Context, tenantID uuid.UUID, path string) (<-chan resource.WatchEvent, error) {
	// Implementation placeholder
	ch := make(chan resource.WatchEvent)
	return ch, nil
}
