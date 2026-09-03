package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/domain/repository"
	"vnp-memory/ov-search/internal/usecase/dto"
	"vnp-memory/ov-search/internal/usecase/port"
)

type embeddingUseCase struct {
	vectorRepo repository.VectorRepository
	embedder   port.EmbedderPort
}

func NewEmbeddingUseCase(vr repository.VectorRepository, emb port.EmbedderPort) port.EmbeddingUseCase {
	return &embeddingUseCase{
		vectorRepo: vr,
		embedder:   emb,
	}
}

func (u *embeddingUseCase) Upsert(ctx context.Context, req dto.UpsertRequest) error {
	dense, sparse, err := u.embedder.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(req.Content))
	contentHash := hex.EncodeToString(hash[:])

	vec := model.EmbeddingVector{
		Vector:       dense,
		SparseVector: sparse,
	}

	payload := model.UpsertPayload{
		Path:         req.Path,
		AccountID:    req.AccountID,
		UserID:       req.UserID,
		ContentHash:  contentHash,
		ContextLevel: req.ContextLevel,
		UpdatedAt:    time.Now(),
	}

	return u.vectorRepo.Upsert(ctx, vec, payload)
}

func (u *embeddingUseCase) Delete(ctx context.Context, req dto.DeleteRequest) error {
	return u.vectorRepo.Delete(ctx, req.Path, req.AccountID)
}
