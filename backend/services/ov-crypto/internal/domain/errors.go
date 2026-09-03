package domain

import "errors"

var (
	// ErrAuthenticationFailed indicates an AES-GCM auth tag mismatch or invalid credentials
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrCorruptedCiphertext indicates the envelope could not be parsed
	ErrCorruptedCiphertext = errors.New("corrupted ciphertext envelope")

	// ErrInvalidMagic indicates the data does not start with the OVE1 magic string
	ErrInvalidMagic = errors.New("invalid envelope magic")

	// ErrKeyMismatch indicates the KMS was unable to decrypt the File Key
	ErrKeyMismatch = errors.New("key mismatch during decryption")

	// ErrNotFound indicates the requested key or metadata was not found
	ErrNotFound = errors.New("resource not found")
)
