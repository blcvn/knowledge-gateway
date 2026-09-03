package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vnp-memory/services/ov-fs/internal/domain"
	"vnp-memory/services/ov-fs/internal/domain/model"
	"vnp-memory/services/ov-fs/internal/domain/repository"
	"vnp-memory/services/ov-fs/internal/usecase/dto"
	"vnp-memory/services/ov-fs/internal/usecase/port"
)

type fileUseCase struct {
	fileRepo   repository.FileRepository
	absRepo    repository.AbstractRepository
	crypto     port.EncryptionPort
	eventPub   port.EventPublisherPort
}

func NewFileUseCase(
	fileRepo repository.FileRepository,
	absRepo repository.AbstractRepository,
	crypto port.EncryptionPort,
	eventPub port.EventPublisherPort,
) port.FileUseCase {
	return &fileUseCase{
		fileRepo: fileRepo,
		absRepo:  absRepo,
		crypto:   crypto,
		eventPub: eventPub,
	}
}

func (uc *fileUseCase) ReadFile(ctx context.Context, req dto.ReadFileRequest) (*dto.ReadFileResponse, error) {
	file, err := uc.fileRepo.ReadFile(ctx, req.AccountID, req.Path)
	if err != nil {
		return nil, err
	}

	content := file.Content
	// If reading L2 (full content), we might need to decrypt
	if req.ContextLevel == model.ContextLevelL2 {
		// Assuming checking OVE1 magic header logic can be put here or in crypto port
		decrypted, err := uc.crypto.Decrypt(ctx, req.AccountID, content)
		if err == nil {
			content = decrypted
		}
	} else if req.ContextLevel == model.ContextLevelL1 {
		l1, err := uc.absRepo.GetAbstract(ctx, req.AccountID, req.Path, model.ContextLevelL1)
		if err == nil {
			content = []byte(l1)
		}
	} else if req.ContextLevel == model.ContextLevelL0 {
		l0, err := uc.absRepo.GetAbstract(ctx, req.AccountID, req.Path, model.ContextLevelL0)
		if err == nil {
			content = []byte(l0)
		}
	}

	return &dto.ReadFileResponse{
		Content: content,
		Metadata: &model.FileMetadata{
			AccountID: file.AccountID,
			Path:      file.Path,
			SizeBytes: file.SizeBytes,
			MimeType:  file.MimeType,
			Checksum:  file.Checksum,
			IsDir:     file.IsDir,
		},
		ContextLevel: req.ContextLevel,
	}, nil
}

func (uc *fileUseCase) WriteFile(ctx context.Context, req dto.WriteFileRequest) (*dto.WriteFileResponse, error) {
	encrypted, err := uc.crypto.Encrypt(ctx, req.AccountID, req.Content)
	if err != nil {
		return nil, err
	}

	checksum := sha256.Sum256(req.Content)
	hashStr := hex.EncodeToString(checksum[:])

	file := &model.File{
		AccountID: req.AccountID,
		UserID:    req.UserID,
		Path:      req.Path,
		Content:   encrypted,
		SizeBytes: int64(len(req.Content)),
		Checksum:  hashStr,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.ContextAbstracts != nil {
		file.L0Abstract = req.ContextAbstracts.L0
		file.L1Abstract = req.ContextAbstracts.L1
	}

	err = uc.fileRepo.WriteFile(ctx, file, req.CreateParents)
	if err != nil {
		return nil, err
	}

	if req.ContextAbstracts != nil {
		_ = uc.absRepo.StoreAbstract(ctx, req.AccountID, req.Path, model.ContextLevelL0, req.ContextAbstracts.L0)
		_ = uc.absRepo.StoreAbstract(ctx, req.AccountID, req.Path, model.ContextLevelL1, req.ContextAbstracts.L1)
	}

	_ = uc.eventPub.PublishContentWritten(ctx, domain.ContentWritten{
		Path:      req.Path,
		AccountID: req.AccountID,
		SizeBytes: file.SizeBytes,
		Checksum:  hashStr,
	})

	return &dto.WriteFileResponse{
		Path:      req.Path,
		SizeBytes: file.SizeBytes,
		Encrypted: true,
	}, nil
}

func (uc *fileUseCase) DeleteFile(ctx context.Context, req dto.DeleteFileRequest) error {
	err := uc.fileRepo.DeleteFile(ctx, req.AccountID, req.Path, req.Recursive)
	if err != nil {
		return err
	}

	_ = uc.eventPub.PublishContentDeleted(ctx, domain.ContentDeleted{
		Path:      req.Path,
		AccountID: req.AccountID,
	})

	return nil
}
