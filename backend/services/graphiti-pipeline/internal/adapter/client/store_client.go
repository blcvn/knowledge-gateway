package client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"graphiti-pipeline/internal/infra/config"
	"graphiti-pipeline/internal/usecase/port"
)

type StoreGRPCClient struct {
	conn *grpc.ClientConn
}

func NewStoreClient(cfg config.Config) port.StoreClient {
	endpoint := cfg.Store.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9024"
	}
	conn, _ := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return &StoreGRPCClient{conn: conn}
}

func (c *StoreGRPCClient) SaveBulk(ctx context.Context, req port.SaveBulkRequest) error {
	return nil
}

func (c *StoreGRPCClient) RollbackBulk(ctx context.Context, episodeID string) error {
	return nil
}
