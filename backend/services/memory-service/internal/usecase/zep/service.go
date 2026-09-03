// Package zep implements Zep memory usecases.
//
// Absorbed from: zep-user, zep-thread, zep-memory, zep-search, zep-graph
// Uses port.ZepClient which can be backed by Zep Cloud SDK or HTTP client.
// (MERGE-P2-T3)
package zep

import (
	"context"
	"fmt"

	"vnp-memory/services/memory-service/internal/domain/zep"
	"vnp-memory/services/memory-service/internal/usecase/port"
)

// UserUseCase manages Zep users.
type UserUseCase struct {
	client  port.ZepClient
	enabled bool
}

// NewUserUseCase creates a UserUseCase.
func NewUserUseCase(client port.ZepClient, enabled bool) *UserUseCase {
	return &UserUseCase{client: client, enabled: enabled}
}

func (uc *UserUseCase) checkEnabled() error {
	if !uc.enabled {
		return fmt.Errorf("zep: feature disabled (set ZEP_ENABLED=true and ZEP_API_KEY)")
	}
	return nil
}

// CreateUser creates a user in Zep Cloud.
func (uc *UserUseCase) CreateUser(ctx context.Context, userID, email, firstName, lastName string, meta map[string]any) (*zep.ZepUser, error) {
	if err := uc.checkEnabled(); err != nil {
		return nil, err
	}
	return uc.client.CreateUser(ctx, userID, email, firstName, lastName, meta)
}

// GetUser retrieves a user from Zep Cloud.
func (uc *UserUseCase) GetUser(ctx context.Context, userID string) (*zep.ZepUser, error) {
	if err := uc.checkEnabled(); err != nil {
		return nil, err
	}
	return uc.client.GetUser(ctx, userID)
}

// UpdateUser updates a user's metadata in Zep Cloud.
func (uc *UserUseCase) UpdateUser(ctx context.Context, userID string, updates map[string]any) (*zep.ZepUser, error) {
	if err := uc.checkEnabled(); err != nil {
		return nil, err
	}
	return uc.client.UpdateUser(ctx, userID, updates)
}

// MemoryUseCase handles Zep session memory.
type MemoryUseCase struct {
	client  port.ZepClient
	enabled bool
}

// NewMemoryUseCase creates a MemoryUseCase.
func NewMemoryUseCase(client port.ZepClient, enabled bool) *MemoryUseCase {
	return &MemoryUseCase{client: client, enabled: enabled}
}

// PutMemory stores messages in a Zep session.
func (uc *MemoryUseCase) PutMemory(ctx context.Context, sessionID string, mem *zep.ZepMemory) error {
	if !uc.enabled {
		return fmt.Errorf("zep: feature disabled")
	}
	return uc.client.PutMemory(ctx, sessionID, mem)
}

// GetMemory retrieves messages + summary + facts from a Zep session.
func (uc *MemoryUseCase) GetMemory(ctx context.Context, sessionID string) (*zep.ZepMemory, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("zep: feature disabled")
	}
	return uc.client.GetMemory(ctx, sessionID)
}

// SessionSearch searches messages within a session.
func (uc *MemoryUseCase) SessionSearch(ctx context.Context, sessionID, query string, limit int) ([]*zep.ZepMessage, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("zep: feature disabled")
	}
	if limit <= 0 {
		limit = 10
	}
	return uc.client.SessionSearch(ctx, sessionID, query, limit)
}

// GraphUseCase manages Zep knowledge graph.
type GraphUseCase struct {
	client  port.ZepClient
	enabled bool
}

// NewGraphUseCase creates a GraphUseCase.
func NewGraphUseCase(client port.ZepClient, enabled bool) *GraphUseCase {
	return &GraphUseCase{client: client, enabled: enabled}
}

// GraphSearch searches the Zep user knowledge graph.
func (uc *GraphUseCase) GraphSearch(ctx context.Context, userID, query string, limit int) ([]*zep.GraphFact, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("zep: feature disabled")
	}
	if limit <= 0 {
		limit = 10
	}
	return uc.client.GraphSearch(ctx, userID, query, limit)
}

// AddFact adds a knowledge graph fact for a user.
func (uc *GraphUseCase) AddFact(ctx context.Context, userID string, fact *zep.GraphFact) error {
	if !uc.enabled {
		return fmt.Errorf("zep: feature disabled")
	}
	return uc.client.AddFact(ctx, userID, fact)
}
