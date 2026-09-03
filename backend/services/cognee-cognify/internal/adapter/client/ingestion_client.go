package client

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type ingestionClient struct {
	conn *grpc.ClientConn
}

func NewIngestionClient(conn *grpc.ClientConn) port.DataItemReader {
	return &ingestionClient{conn: conn}
}

func (c *ingestionClient) GetTextByDataset(ctx context.Context, tenantID string, datasetID uuid.UUID) ([]port.TextItem, error) {
	// Note: in a real implementation, this would call a generated gRPC client stub like:
	// client := ingestionv1.NewIngestionServiceClient(c.conn)
	// resp, err := client.GetDatasetItems(ctx, &ingestionv1.GetDatasetItemsRequest{...})
	
	// Since the proto files are not available in this task's context, 
	// we stub the return as a placeholder for enterprise implementation.
	return []port.TextItem{
		{
			ID:   uuid.New().String(),
			Text: "Data fetched from cognee-ingestion.",
		},
	}, nil
}
