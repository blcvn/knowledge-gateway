package usecase

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/repository"
	"vnp-memory/services/ov-fs/internal/usecase/dto"
	"vnp-memory/services/ov-fs/internal/usecase/port"
	"vnp-memory/services/ov-fs/internal/domain/model"
	"time"
)

type relationUseCase struct {
	relationRepo repository.RelationRepository
}

func NewRelationUseCase(relationRepo repository.RelationRepository) port.RelationUseCase {
	return &relationUseCase{relationRepo: relationRepo}
}

func (uc *relationUseCase) GetRelations(ctx context.Context, req dto.GetRelationsRequest) (*dto.RelationsResponse, error) {
	relations, err := uc.relationRepo.GetRelations(ctx, req.AccountID, req.Path, req.RelationType)
	if err != nil {
		return nil, err
	}
	return &dto.RelationsResponse{Relations: relations}, nil
}

func (uc *relationUseCase) AddRelation(ctx context.Context, req dto.AddRelationRequest) error {
	relation := &model.FileRelation{
		SourceFileID: req.SourcePath, // Usually resolving to ID, but using path here as an abstraction
		TargetFileID: req.TargetPath,
		RelationType: req.RelationType,
		AccountID:    req.AccountID,
		Metadata:     req.Metadata,
		CreatedAt:    time.Now(),
	}
	return uc.relationRepo.AddRelation(ctx, relation)
}
