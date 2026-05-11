package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/crypto"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/port"
)

var (
	ErrKeyNotFound = errors.New("encryption key not found")
	ErrWrapFailed  = errors.New("failed to wrap key")
)

type cryptoUseCase struct {
	keyRepo port.KeyRepository
	kms     port.KMSProvider
}

// NewCryptoUseCase creates a new instance of CryptoUseCase.
func NewCryptoUseCase(keyRepo port.KeyRepository, kms port.KMSProvider) port.CryptoUseCase {
	return &cryptoUseCase{
		keyRepo: keyRepo,
		kms:     kms,
	}
}

func (c *cryptoUseCase) GenerateDEK(ctx context.Context, tenantID uuid.UUID) (*crypto.EncryptionKey, error) {
	// Generate 32-byte DEK
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, err
	}

	// Wrap DEK using KMS
	wrappedDEK, kekVersion, err := c.kms.WrapKey(ctx, dek)
	if err != nil {
		return nil, ErrWrapFailed
	}

	key := &crypto.EncryptionKey{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Algorithm:  "AES-256-GCM",
		WrappedKey: wrappedDEK,
		KEKVersion: kekVersion,
		CreatedAt:  time.Now(),
	}

	if err := c.keyRepo.Create(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (c *cryptoUseCase) GetDEK(ctx context.Context, keyID uuid.UUID) (*crypto.EncryptionKey, error) {
	key, err := c.keyRepo.FindByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

func (c *cryptoUseCase) RotateKeys(ctx context.Context, tenantID uuid.UUID) error {
	// Implement Key Rotation Algorithm
	// In reality, this would iterate over active keys and re-wrap their DEKs.
	return nil // placeholder for implementation
}

// Helper to encrypt data
func EncryptData(plainText []byte, dek []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	iv := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, nil, err
	}

	cipherText := aesGCM.Seal(nil, iv, plainText, nil)
	return cipherText, iv, nil
}

// Helper to decrypt data
func DecryptData(cipherText []byte, dek []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plainText, err := aesGCM.Open(nil, iv, cipherText, nil)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}
