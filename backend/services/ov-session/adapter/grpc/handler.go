package grpc

import (
	"context"

	"github.com/vnp-memory/services/ov-session/usecase"
	"github.com/vnp-memory/services/ov-session/usecase/port"
)

type OvSessionHandler struct {
	sessionUC port.SessionUseCase
	wmUC      usecase.WorkingMemoryUseCase
	commitUC  usecase.CommitUseCase
}

func NewOvSessionHandler(
	sessionUC port.SessionUseCase,
	wmUC usecase.WorkingMemoryUseCase,
	commitUC usecase.CommitUseCase,
) *OvSessionHandler {
	return &OvSessionHandler{
		sessionUC: sessionUC,
		wmUC:      wmUC,
		commitUC:  commitUC,
	}
}

// Mocking gRPC interface endpoints
func (h *OvSessionHandler) CreateSession(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *OvSessionHandler) AddMessage(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *OvSessionHandler) GetMessages(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *OvSessionHandler) CommitSession(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *OvSessionHandler) GetWorkingMemory(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *OvSessionHandler) UpdateWorkingMemory(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}
