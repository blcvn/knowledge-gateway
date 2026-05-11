package usecase_test

import (
	"context"
	"testing"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type mockGraphDriver struct {
	domain.GraphDriver
	saveNodeFunc func(ctx context.Context, node domain.EntityNode) error
	getNodeFunc  func(ctx context.Context, groupID, uuid string) (*domain.EntityNode, error)
}

func (m *mockGraphDriver) SaveNode(ctx context.Context, node domain.EntityNode) error {
	if m.saveNodeFunc != nil {
		return m.saveNodeFunc(ctx, node)
	}
	return nil
}

func (m *mockGraphDriver) GetNode(ctx context.Context, groupID, uuid string) (*domain.EntityNode, error) {
	if m.getNodeFunc != nil {
		return m.getNodeFunc(ctx, groupID, uuid)
	}
	return nil, domain.ErrNodeNotFound
}

func TestNodeService_SaveNode(t *testing.T) {
	ctx := context.Background()
	driver := &mockGraphDriver{
		saveNodeFunc: func(ctx context.Context, node domain.EntityNode) error {
			if node.Name == "" {
				return domain.ErrMissingName
			}
			return nil
		},
	}
	service := usecase.NewNodeService(driver)

	t.Run("Valid Node", func(t *testing.T) {
		req := port.SaveNodeRequest{
			UUID:    "123",
			Name:    "Test",
			GroupID: "tenant-1",
		}
		node, err := service.SaveNode(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if node.UUID != "123" {
			t.Errorf("expected UUID 123, got %s", node.UUID)
		}
	})

	t.Run("Invalid Node Missing Name", func(t *testing.T) {
		req := port.SaveNodeRequest{
			UUID:    "123",
			GroupID: "tenant-1",
		}
		_, err := service.SaveNode(ctx, req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
