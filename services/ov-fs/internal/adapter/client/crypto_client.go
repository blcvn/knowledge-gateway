package client

import (
	"context"

	"vnp-memory/services/ov-fs/internal/usecase/port"
)

type cryptoClient struct {
	// grpcClient pb.OvCryptoServiceClient
}

// Ensure interface is implemented
var _ port.EncryptionPort = (*cryptoClient)(nil)

func NewCryptoClient() port.EncryptionPort {
	return &cryptoClient{}
}

func (c *cryptoClient) Encrypt(ctx context.Context, accountID string, plaintext []byte) ([]byte, error) {
	// Simulate gRPC call to ov-crypto
	// resp, err := c.grpcClient.Encrypt(ctx, &pb.EncryptRequest{AccountId: accountID, Plaintext: plaintext})
	// return resp.Ciphertext, err
	
	// returning plaintext as a mock for now
	return plaintext, nil
}

func (c *cryptoClient) Decrypt(ctx context.Context, accountID string, ciphertext []byte) ([]byte, error) {
	// Simulate gRPC call to ov-crypto
	// resp, err := c.grpcClient.Decrypt(ctx, &pb.DecryptRequest{AccountId: accountID, Ciphertext: ciphertext})
	// return resp.Plaintext, err

	// returning ciphertext as a mock for now
	return ciphertext, nil
}
