// Package port defines input/output port interfaces for ov-storage.
//
// Consolidated from: ov-fs + ov-crypto + ov-resource
// Key optimization: Encryption is transparent within FS write operations.
package port

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/crypto"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/fs"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/resource"
)

// --- Input Ports ---

// FsUseCase handles VikingFS file operations with transparent encryption.
type FsUseCase interface {
	WriteFile(ctx context.Context, tenantID uuid.UUID, path string, content io.Reader, encrypt bool) (*fs.File, error)
	ReadFile(ctx context.Context, tenantID uuid.UUID, path string) (io.ReadCloser, *fs.File, error)
	DeleteFile(ctx context.Context, tenantID uuid.UUID, path string) error
	ListDirectory(ctx context.Context, tenantID uuid.UUID, path string) ([]fs.File, []fs.Directory, error)
	MoveFile(ctx context.Context, tenantID uuid.UUID, srcPath, dstPath string) (*fs.File, error)
	StatFile(ctx context.Context, tenantID uuid.UUID, path string) (*fs.File, error)

	// Lock management for concurrent access
	AcquireLock(ctx context.Context, tenantID uuid.UUID, path string, lockType fs.LockType) (*fs.PathLock, error)
	ReleaseLock(ctx context.Context, lockID uuid.UUID) error
}

// CryptoUseCase handles envelope encryption key management.
type CryptoUseCase interface {
	GenerateDEK(ctx context.Context, tenantID uuid.UUID) (*crypto.EncryptionKey, error)
	GetDEK(ctx context.Context, keyID uuid.UUID) (*crypto.EncryptionKey, error)
	RotateKeys(ctx context.Context, tenantID uuid.UUID) error
}

// ResourceUseCase handles content parsing and indexing.
type ResourceUseCase interface {
	ParseResource(ctx context.Context, tenantID uuid.UUID, path string, parserType resource.ParserType) (*resource.Resource, error)
	GetResource(ctx context.Context, id uuid.UUID) (*resource.Resource, error)
	WatchPath(ctx context.Context, tenantID uuid.UUID, path string) (<-chan resource.WatchEvent, error)
}

// --- Output Ports ---

// FileRepository persists file metadata.
type FileRepository interface {
	Create(ctx context.Context, file *fs.File) error
	FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*fs.File, error)
	Update(ctx context.Context, file *fs.File) error
	Delete(ctx context.Context, tenantID uuid.UUID, path string) error
	ListByDirectory(ctx context.Context, tenantID uuid.UUID, dirPath string) ([]fs.File, error)
}

// DirectoryRepository persists directory metadata.
type DirectoryRepository interface {
	Create(ctx context.Context, dir *fs.Directory) error
	FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*fs.Directory, error)
	ListChildren(ctx context.Context, tenantID uuid.UUID, path string) ([]fs.Directory, error)
}

// LockRepository persists path locks.
type LockRepository interface {
	Acquire(ctx context.Context, lock *fs.PathLock) error
	Release(ctx context.Context, id uuid.UUID) error
	FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*fs.PathLock, error)
}

// KeyRepository persists encryption keys.
type KeyRepository interface {
	Create(ctx context.Context, key *crypto.EncryptionKey) error
	FindByID(ctx context.Context, id uuid.UUID) (*crypto.EncryptionKey, error)
	FindActiveByTenant(ctx context.Context, tenantID uuid.UUID) (*crypto.EncryptionKey, error)
}

// ResourceRepository persists parsed resources.
type ResourceRepository interface {
	Create(ctx context.Context, res *resource.Resource) error
	FindByID(ctx context.Context, id uuid.UUID) (*resource.Resource, error)
	FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*resource.Resource, error)
}

// ObjectStorage abstracts the blob storage backend (MinIO/S3/local).
type ObjectStorage interface {
	Put(ctx context.Context, bucket, key string, data io.Reader) error
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, bucket, key string) error
}

// KMSProvider abstracts the Key Management Service.
type KMSProvider interface {
	WrapKey(ctx context.Context, plainDEK []byte) ([]byte, int, error) // wrappedDEK, keyVersion, error
	UnwrapKey(ctx context.Context, wrappedDEK []byte, keyVersion int) ([]byte, error)
}

// EventPublisher publishes storage events to NATS.
type EventPublisher interface {
	PublishFileWritten(ctx context.Context, tenantID uuid.UUID, path string) error
	PublishResourceParsed(ctx context.Context, tenantID, resourceID uuid.UUID) error
}
