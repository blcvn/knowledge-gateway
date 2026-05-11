package grpc_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/grpc"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase/port"
	"google.golang.org/grpc/metadata"
)

type mockNodeService struct {
	port.NodeService
	saveNodeFunc func(ctx context.Context, req port.SaveNodeRequest) (*domain.EntityNode, error)
}

func (m *mockNodeService) SaveNode(ctx context.Context, req port.SaveNodeRequest) (*domain.EntityNode, error) {
	if m.saveNodeFunc != nil {
		return m.saveNodeFunc(ctx, req)
	}
	return nil, nil
}

func TestHandler_SaveNode(t *testing.T) {
	ns := &mockNodeService{
		saveNodeFunc: func(ctx context.Context, req port.SaveNodeRequest) (*domain.EntityNode, error) {
			return &domain.EntityNode{
				UUID:    req.UUID,
				Name:    req.Name,
				GroupID: req.GroupID,
			}, nil
		},
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := grpc.NewHandler(ns, nil, nil, nil, nil, logger)

	// Context with x-tenant-id metadata
	md := metadata.Pairs("x-tenant-id", "tenant-1")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &grpc.SaveNodeRequest{
		Uuid: "123",
		Name: "Test Node",
	}

	resp, err := handler.SaveNode(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Uuid != "123" || resp.Name != "Test Node" || resp.GroupId != "tenant-1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestHandler_SaveNode_MissingTenant(t *testing.T) {
	ns := &mockNodeService{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := grpc.NewHandler(ns, nil, nil, nil, nil, logger)

	// Context without metadata
	ctx := context.Background()

	req := &grpc.SaveNodeRequest{
		Uuid: "123",
		Name: "Test Node",
	}

	_, err := handler.SaveNode(ctx, req)
	if err == nil {
		t.Fatal("expected error due to missing tenant, got nil")
	}
}
