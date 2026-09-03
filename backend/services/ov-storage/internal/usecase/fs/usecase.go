package fs

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/fs"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/port"
)

type fsUseCase struct {
	fileRepo port.FileRepository
	dirRepo  port.DirectoryRepository
	storage  port.ObjectStorage
	crypto   port.CryptoUseCase
	kms      port.KMSProvider
	pub      port.EventPublisher
	lockRepo port.LockRepository
}

// NewFsUseCase creates a new instance of FsUseCase.
func NewFsUseCase(
	fileRepo port.FileRepository,
	dirRepo port.DirectoryRepository,
	storage port.ObjectStorage,
	crypto port.CryptoUseCase,
	kms port.KMSProvider,
	pub port.EventPublisher,
	lockRepo port.LockRepository,
) port.FsUseCase {
	return &fsUseCase{
		fileRepo: fileRepo,
		dirRepo:  dirRepo,
		storage:  storage,
		crypto:   crypto,
		kms:      kms,
		pub:      pub,
		lockRepo: lockRepo,
	}
}

func (u *fsUseCase) WriteFile(ctx context.Context, tenantID uuid.UUID, path string, content io.Reader, encrypt bool) (*fs.File, error) {
	// 1. PathLock
	lock, err := u.AcquireLock(ctx, tenantID, path, fs.LockPoint)
	if err != nil {
		return nil, err
	}
	defer u.ReleaseLock(ctx, lock.ID)

	var dataReader io.Reader = content

	// Encryption logic will go here
	// Read full content into memory for simplicity in this implementation
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}

	isEncrypted := false
	if encrypt {
		dekKey, err := u.crypto.GenerateDEK(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		
		plainDEK, err := u.kms.UnwrapKey(ctx, dekKey.WrappedKey, dekKey.KEKVersion)
		if err != nil {
			return nil, err
		}

		cipherText, _, err := EncryptData(data, plainDEK)
		if err != nil {
			return nil, err
		}
		// Write OVE1 Envelope + CipherText to storage
		// In a real implementation we would stream this
		data = cipherText
		isEncrypted = true
	}

	err = u.storage.Put(ctx, tenantID.String(), path, byteReader(data))
	if err != nil {
		return nil, err
	}

	file := &fs.File{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Path:      path,
		Name:      extractName(path),
		SizeBytes: int64(len(data)),
		Encrypted: isEncrypted,
		TierLevel: fs.TierL0Hot,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = u.fileRepo.Create(ctx, file)
	if err != nil {
		return nil, err
	}

	u.pub.PublishFileWritten(ctx, tenantID, path)

	return file, nil
}

func (u *fsUseCase) ReadFile(ctx context.Context, tenantID uuid.UUID, path string) (io.ReadCloser, *fs.File, error) {
	file, err := u.fileRepo.FindByPath(ctx, tenantID, path)
	if err != nil {
		return nil, nil, err
	}

	dataStream, err := u.storage.Get(ctx, tenantID.String(), path)
	if err != nil {
		return nil, nil, err
	}

	if file.Encrypted {
		// Decrypt stream
		// ... implementation omitted for brevity
	}

	return dataStream, file, nil
}

func (u *fsUseCase) DeleteFile(ctx context.Context, tenantID uuid.UUID, path string) error {
	lock, err := u.AcquireLock(ctx, tenantID, path, fs.LockPoint)
	if err != nil {
		return err
	}
	defer u.ReleaseLock(ctx, lock.ID)

	err = u.storage.Delete(ctx, tenantID.String(), path)
	if err != nil {
		return err
	}

	return u.fileRepo.Delete(ctx, tenantID, path)
}

func (u *fsUseCase) ListDirectory(ctx context.Context, tenantID uuid.UUID, path string) ([]fs.File, []fs.Directory, error) {
	// Implementation
	return nil, nil, nil
}

func (u *fsUseCase) MoveFile(ctx context.Context, tenantID uuid.UUID, srcPath, dstPath string) (*fs.File, error) {
	// Implementation with MoveLock
	return nil, nil
}

func (u *fsUseCase) StatFile(ctx context.Context, tenantID uuid.UUID, path string) (*fs.File, error) {
	return u.fileRepo.FindByPath(ctx, tenantID, path)
}

func (u *fsUseCase) AcquireLock(ctx context.Context, tenantID uuid.UUID, path string, lockType fs.LockType) (*fs.PathLock, error) {
	lock := &fs.PathLock{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Path:       path,
		LockType:   lockType,
		AcquiredAt: time.Now(),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	err := u.lockRepo.Acquire(ctx, lock)
	return lock, err
}

func (u *fsUseCase) ReleaseLock(ctx context.Context, id uuid.UUID) error {
	return u.lockRepo.Release(ctx, id)
}

func extractName(path string) string {
	return "filename" // simplified
}

type byteReader []byte
func (b byteReader) Read(p []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, io.EOF
	}
	n = copy(p, b)
	b = b[n:]
	return n, nil
}

// Copy of crypto helpers for usecase encapsulation
func EncryptData(plainText []byte, dek []byte) ([]byte, []byte, error) {
	return nil, nil, nil // placeholder to fix compile
}
