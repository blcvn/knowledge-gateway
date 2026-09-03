package usecase

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"vnp-memory/services/ov-crypto/internal/domain"
	"vnp-memory/services/ov-crypto/internal/domain/model"
	"vnp-memory/services/ov-crypto/internal/domain/repository"
	"vnp-memory/services/ov-crypto/internal/usecase/dto"
)

type encryptor struct {
	kms  model.KMSProvider
	repo repository.KeyRepository
}

func NewEncryptor(kms model.KMSProvider, repo repository.KeyRepository) *encryptor {
	return &encryptor{kms: kms, repo: repo}
}

func (e *encryptor) Encrypt(ctx context.Context, req dto.EncryptRequest) (*dto.EncryptResponse, error) {
	if len(req.Plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}

	// 1. Get active account key metadata to determine provider type and version
	keyMeta, err := e.repo.GetActiveAccountKey(ctx, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account key: %w", err)
	}

	// 2. Generate random 32-byte File Key
	fileKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, fileKey); err != nil {
		return nil, fmt.Errorf("failed to generate file key: %w", err)
	}

	// 3. Generate random 12-byte Data IV
	dataIV := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, dataIV); err != nil {
		return nil, fmt.Errorf("failed to generate data iv: %w", err)
	}

	// 4. AES-256-GCM encrypt plaintext with (File Key, Data IV)
	block, err := aes.NewCipher(fileKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	contentCiphertext := gcm.Seal(nil, dataIV, req.Plaintext, nil)

	// 5. Encrypt File Key with Account Key (via KMS provider)
	encryptedFileKey, keyIV, err := e.kms.EncryptFileKey(fileKey, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt file key via KMS: %w", err)
	}

	// 6. Build OVE1 envelope
	var pType model.ProviderType
	switch keyMeta.ProviderType {
	case "local":
		pType = model.ProviderTypeLocal
	case "vault":
		pType = model.ProviderTypeVault
	case "aws_kms", "gcp_kms":
		pType = model.ProviderTypeCloud
	default:
		pType = model.ProviderTypeUnknown
	}

	buf := new(bytes.Buffer)
	buf.WriteString(model.OVE1Magic)
	buf.WriteByte(0x01) // Version
	buf.WriteByte(byte(pType))

	if err := binary.Write(buf, binary.BigEndian, uint16(len(encryptedFileKey))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, uint16(len(keyIV))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, uint16(len(dataIV))); err != nil {
		return nil, err
	}

	buf.Write(encryptedFileKey)
	buf.Write(keyIV)
	buf.Write(dataIV)
	buf.Write(contentCiphertext)

	return &dto.EncryptResponse{
		Ciphertext: buf.Bytes(),
		KeyVersion: int(keyMeta.KeyVersion),
	}, nil
}
