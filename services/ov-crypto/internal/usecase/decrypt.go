package usecase

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"

	"vnp-memory/services/ov-crypto/internal/domain"
	"vnp-memory/services/ov-crypto/internal/domain/model"
	"vnp-memory/services/ov-crypto/internal/usecase/dto"
)

func (e *encryptor) Decrypt(ctx context.Context, req dto.DecryptRequest) (*dto.DecryptResponse, error) {
	if len(req.Ciphertext) < 12 { // Minimum magic + headers
		// Backward compatibility: return as-is
		return &dto.DecryptResponse{Plaintext: req.Ciphertext}, nil
	}

	// 1. Check OVE1 magic
	if string(req.Ciphertext[:4]) != model.OVE1Magic {
		// Not an OVE1 envelope, return as-is for backward compatibility
		return &dto.DecryptResponse{Plaintext: req.Ciphertext}, nil
	}

	// 2. Parse envelope header
	// Format: Magic(4) | Version(1) | ProviderType(1) | EFKLen(2) | KIVLen(2) | DIVLen(2)
	offset := 6
	buf := bytes.NewReader(req.Ciphertext[offset : offset+6])
	
	var efkLen, kivLen, divLen uint16
	binary.Read(buf, binary.BigEndian, &efkLen)
	binary.Read(buf, binary.BigEndian, &kivLen)
	binary.Read(buf, binary.BigEndian, &divLen)

	offset += 6

	if len(req.Ciphertext) < offset+int(efkLen)+int(kivLen)+int(divLen) {
		return nil, domain.ErrCorruptedCiphertext
	}

	encryptedFileKey := req.Ciphertext[offset : offset+int(efkLen)]
	offset += int(efkLen)

	keyIV := req.Ciphertext[offset : offset+int(kivLen)]
	offset += int(kivLen)

	dataIV := req.Ciphertext[offset : offset+int(divLen)]
	offset += int(divLen)

	contentCiphertext := req.Ciphertext[offset:]

	// 3. Decrypt File Key via KMS provider
	fileKey, err := e.kms.DecryptFileKey(encryptedFileKey, keyIV, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrKeyMismatch, err)
	}

	// 4. AES-256-GCM decrypt content
	block, err := aes.NewCipher(fileKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, dataIV, contentCiphertext, nil)
	if err != nil {
		return nil, domain.ErrAuthenticationFailed
	}

	return &dto.DecryptResponse{
		Plaintext: plaintext,
	}, nil
}
