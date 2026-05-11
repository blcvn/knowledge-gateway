package client

import (
	"context"

	"vnp-memory/ov-search/internal/usecase/port"
)

type fsClient struct {
	addr string
}

func NewFsClient(addr string) port.FileReaderPort {
	return &fsClient{addr: addr}
}

func (c *fsClient) ReadContext(ctx context.Context, path string, level string) (string, error) {
	// Mock gRPC call to ov-fs
	// L0: abstract, L1: overview, L2: full content
	return "Mocked content from ov-fs at level " + level, nil
}
