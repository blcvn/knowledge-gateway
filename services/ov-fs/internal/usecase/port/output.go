package port

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain"
)

type EncryptionPort interface {
	Encrypt(ctx context.Context, accountID string, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, accountID string, ciphertext []byte) ([]byte, error)
}

type EventPublisherPort interface {
	PublishContentWritten(ctx context.Context, event domain.ContentWritten) error
	PublishContentDeleted(ctx context.Context, event domain.ContentDeleted) error
}

type AbstractGeneratorPort interface {
	GenerateAbstracts(ctx context.Context, content string) (string, string, error) // returns L0, L1
}
