package batch

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const (
	MaxBatchSize = 1000
)

var (
	ErrEmptyBatch = errors.New("empty batch")
	ErrMaxBatch   = errors.New("batch exceeds maximum size")
)

type Entity struct {
	Label      string
	Properties map[string]any
}

type BatchUpsertRequest struct {
	AppID    string
	TenantID string
	Entities []Entity
}

type BatchUpsertResult struct {
	Created int
	Updated int
	Skipped int
}

type Writer interface {
	BulkCreate(ctx context.Context, appID, tenantID string, entities []Entity) (int, error)
}

type Deduper interface {
	Dedup(ctx context.Context, appID, tenantID string, entities []Entity) (unique []Entity, skipped int, err error)
}

type VectorIndexer interface {
	IndexEntities(ctx context.Context, appID, tenantID string, entities []Entity) error
}

type EntityValidator interface {
	ValidateEntity(ctx context.Context, appID, label string, properties map[string]any) error
}

type Usecase struct {
	writer    Writer
	deduper   Deduper
	indexer   VectorIndexer
	validator EntityValidator
}

func NewUsecase(writer Writer, deduper Deduper) *Usecase {
	return NewUsecaseWithIndexer(writer, deduper, nil, nil)
}

func NewUsecaseWithIndexer(writer Writer, deduper Deduper, indexer VectorIndexer, validator EntityValidator) *Usecase {
	return &Usecase{
		writer:    writer,
		deduper:   deduper,
		indexer:   indexer,
		validator: validator,
	}
}

func (u *Usecase) Execute(ctx context.Context, req BatchUpsertRequest) (*BatchUpsertResult, error) {
	if len(req.Entities) == 0 {
		return nil, ErrEmptyBatch
	}
	if len(req.Entities) > MaxBatchSize {
		return nil, ErrMaxBatch
	}

	unique, skipped, err := u.deduper.Dedup(ctx, req.AppID, req.TenantID, req.Entities)
	if err != nil {
		return nil, err
	}

	for i := range unique {
		if unique[i].Label == "" {
			return nil, fmt.Errorf("entity[%d] missing label", i)
		}
		if unique[i].Properties == nil {
			unique[i].Properties = map[string]any{}
		}
		if _, ok := unique[i].Properties["id"].(string); !ok {
			unique[i].Properties["id"] = uuid.NewString()
		}
	}
	if u.validator != nil {
		for i := range unique {
			if err := u.validator.ValidateEntity(ctx, req.AppID, unique[i].Label, unique[i].Properties); err != nil {
				return nil, fmt.Errorf("entity[%d] ontology validation: %w", i, err)
			}
		}
	}

	created, err := u.writer.BulkCreate(ctx, req.AppID, req.TenantID, unique)
	if err != nil {
		return nil, err
	}
	if u.indexer != nil {
		if err := u.indexer.IndexEntities(ctx, req.AppID, req.TenantID, unique); err != nil {
			return nil, err
		}
	}

	return &BatchUpsertResult{
		Created: created,
		Skipped: skipped,
		Updated: 0,
	}, nil
}
