package model

// KMSProvider represents the domain-level abstraction for Key Management Services.
type KMSProvider interface {
	// EncryptFileKey encrypts the provided file key using the account's master key.
	// Returns the encrypted key, initialization vector (if applicable), and an error.
	EncryptFileKey(fileKey []byte, accountID string) (encryptedKey, iv []byte, err error)

	// DecryptFileKey decrypts the file key using the account's master key.
	DecryptFileKey(encryptedKey, iv []byte, accountID string) (fileKey []byte, err error)

	// RotateRootKey rotates the root key or the account-specific key.
	RotateRootKey(accountID string) error
}
