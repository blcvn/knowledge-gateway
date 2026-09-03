package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"vnp-memory/services/ov-crypto/internal/domain"
	"vnp-memory/services/ov-crypto/internal/domain/model"
	"vnp-memory/services/ov-crypto/internal/usecase/dto"
	"vnp-memory/services/ov-crypto/internal/usecase/port"
)

type keyRotator struct {
	kms       model.KMSProvider
	repo      repository.KeyRepository
	publisher port.EventPublisherPort
}

func NewKeyRotator(kms model.KMSProvider, repo repository.KeyRepository, publisher port.EventPublisherPort) *keyRotator {
	return &keyRotator{kms: kms, repo: repo, publisher: publisher}
}

func (kr *keyRotator) RotateKey(ctx context.Context, req dto.RotateKeyRequest) (*dto.RotateKeyResponse, error) {
	start := time.Now()

	// In a real implementation, we would lock or ensure atomicity
	keyMeta, err := kr.repo.GetActiveAccountKey(ctx, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	oldVersion := keyMeta.KeyVersion
	newVersion := oldVersion + 1

	// Orchestrate rotation via KMS
	err = kr.kms.RotateRootKey(req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("KMS rotation failed: %w", err)
	}

	// Update DB
	err = kr.repo.UpdateAccountKeyStatus(ctx, req.AccountID, newVersion, model.KeyStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to update key status: %w", err)
	}

	// Audit Log
	logID := uuid.New().String()
	logEntry := &model.KeyRotationLog{
		ID:          logID,
		AccountID:   req.AccountID,
		OldVersion:  oldVersion,
		NewVersion:  newVersion,
		Reason:      req.Reason,
		InitiatedBy: "system", // Should come from context context
		Status:      "completed",
		DurationMs:  int(time.Since(start).Milliseconds()),
		CreatedAt:   time.Now(),
	}
	_ = kr.repo.RecordRotation(ctx, logEntry)

	// Publish NATS event
	event := domain.KeyRotated{
		AccountID:  req.AccountID,
		OldVersion: int(oldVersion),
		NewVersion: int(newVersion),
	}
	_ = kr.publisher.PublishKeyRotated(ctx, event)

	return &dto.RotateKeyResponse{
		NewVersion:       int(newVersion),
		AffectedAccounts: 1, // Simplifying for this scope
	}, nil
}
