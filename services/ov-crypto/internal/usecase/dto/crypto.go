package dto

import "time"

type EncryptRequest struct {
	AccountID string
	Plaintext []byte
}

type EncryptResponse struct {
	Ciphertext []byte
	KeyVersion int
}

type DecryptRequest struct {
	AccountID  string
	Ciphertext []byte
}

type DecryptResponse struct {
	Plaintext []byte
}

type RotateKeyRequest struct {
	AccountID string
	Reason    string
}

type RotateKeyResponse struct {
	NewVersion       int
	AffectedAccounts int
}

type KeyStatusResponse struct {
	Version     int
	Provider    string
	CreatedAt   time.Time
	LastRotated *time.Time
	Status      string
}

type ValidateAPIKeyRequest struct {
	AccountID string
	RawKey    string
}

type ValidateAPIKeyResponse struct {
	IsValid bool
	Role    string
}
