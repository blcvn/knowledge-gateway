package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/vnp-memory/services/zep-core/domain/memory"
)

type GetMemoryUsecase struct {
	msgRepo      memory.Repository
	searchClient SearchClient
	logger       *zap.Logger
}

type SearchClient interface {
	SearchFacts(ctx context.Context, groupID string, query string, limit int) ([]map[string]interface{}, error)
}

func NewGetMemoryUsecase(msgRepo memory.Repository, searchClient SearchClient, logger *zap.Logger) *GetMemoryUsecase {
	return &GetMemoryUsecase{
		msgRepo:      msgRepo,
		searchClient: searchClient,
		logger:       logger,
	}
}

// Execute retrieves the context payload under the <200ms SLA.
func (uc *GetMemoryUsecase) Execute(ctx context.Context, threadUUID uuid.UUID, sessionID string, userID *string) (*memory.MemoryContext, error) {
	start := time.Now()
	var (
		recentMessages []*memory.Message
		facts          []map[string]interface{}
		group          errgroup.Group
	)

	// Determine GroupID strategy
	groupID := sessionID
	if userID != nil && *userID != "" {
		groupID = *userID
	}

	// Concurrently fetch local messages and query zep-search for facts
	group.Go(func() error {
		var err error
		// Fetch last max(N, 4) messages
		recentMessages, err = uc.msgRepo.GetLastN(ctx, threadUUID, 4)
		if err != nil {
			return fmt.Errorf("failed fetching local messages: %w", err)
		}
		return nil
	})

	group.Go(func() error {
		var err error
		// In a real scenario, the query is built from the recent messages.
		facts, err = uc.searchClient.SearchFacts(ctx, groupID, "context search", 5)
		if err != nil {
			uc.logger.Warn("search client failed, degrading gracefully", zap.Error(err))
			// Do not return error, degrade gracefully and return messages only
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	uc.logger.Info("GetMemory completed", zap.Duration("latency", time.Since(start)))

	return &memory.MemoryContext{
		Messages:      recentMessages,
		RelevantFacts: facts,
	}, nil
}
