package client

import (
	"context"
	"fmt"

	"openviking.com/ov-resource/internal/domain/model"
)

type fsClient struct {
	addr string
}

func NewFsClient(addr string) *fsClient {
	return &fsClient{addr: addr}
}

func (c *fsClient) WriteChunks(ctx context.Context, path, accountID string, chunks []model.Chunk) error {
	// Stub gRPC call to ov-fs
	fmt.Printf("[ov-fs] Writing %d chunks to %s (Account: %s)\n", len(chunks), path, accountID)
	return nil
}
