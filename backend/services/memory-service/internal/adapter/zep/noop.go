// Package zep implements a no-op ZepClient for when ZEP_ENABLED=false.
package zep

import (
	"context"
	"fmt"

	zepdomain "vnp-memory/services/memory-service/internal/domain/zep"
)

// NoopClient is a no-op implementation of port.ZepClient.
type NoopClient struct{}

func (c *NoopClient) CreateUser(_ context.Context, userID, _, _, _ string, _ map[string]any) (*zepdomain.ZepUser, error) {
	return nil, fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) GetUser(_ context.Context, _ string) (*zepdomain.ZepUser, error) {
	return nil, fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) UpdateUser(_ context.Context, _ string, _ map[string]any) (*zepdomain.ZepUser, error) {
	return nil, fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) PutMemory(_ context.Context, _ string, _ *zepdomain.ZepMemory) error {
	return fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) GetMemory(_ context.Context, _ string) (*zepdomain.ZepMemory, error) {
	return nil, fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) GraphSearch(_ context.Context, _, _ string, _ int) ([]*zepdomain.GraphFact, error) {
	return nil, fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) SessionSearch(_ context.Context, _, _ string, _ int) ([]*zepdomain.ZepMessage, error) {
	return nil, fmt.Errorf("zep: not enabled")
}
func (c *NoopClient) AddFact(_ context.Context, _ string, _ *zepdomain.GraphFact) error {
	return fmt.Errorf("zep: not enabled")
}
