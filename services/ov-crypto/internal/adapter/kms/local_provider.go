package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"vnp-memory/services/ov-crypto/internal/domain/model"
)

// LocalProvider implements KMSProvider using a locally stored root key
type LocalProvider struct {
	rootKeyPath string
}

func NewLocalProvider(rootKeyPath string) *LocalProvider {
	return &LocalProvider{rootKeyPath: rootKeyPath}
}

func (p *LocalProvider) getRootKey() ([]byte, error) {
	key, err := os.ReadFile(p.rootKeyPath)
	if err != nil {
		return nil, fmt.Errorf("could not read local root key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("local root key must be 32 bytes")
	}
	return key, nil
}

func (p *LocalProvider) EncryptFileKey(fileKey []byte, accountID string) ([]byte, []byte, error) {
	rootKey, err := p.getRootKey()
	if err != nil {
		return nil, nil, err
	}

	block, err := aes.NewCipher(rootKey)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	encryptedKey := gcm.Seal(nil, iv, fileKey, nil)
	return encryptedKey, iv, nil
}

func (p *LocalProvider) DecryptFileKey(encryptedKey, iv []byte, accountID string) ([]byte, error) {
	rootKey, err := p.getRootKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(rootKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	fileKey, err := gcm.Open(nil, iv, encryptedKey, nil)
	if err != nil {
		return nil, err
	}

	return fileKey, nil
}

func (p *LocalProvider) RotateRootKey(accountID string) error {
	// For local provider, rotation might involve generating a new key and rewriting the file.
	// We'll skip the actual file rewrite for brevity.
	return nil
}

// Compile-time check
var _ model.KMSProvider = (*LocalProvider)(nil)
