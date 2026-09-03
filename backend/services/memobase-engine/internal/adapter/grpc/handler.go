package grpc

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type EngineHandler struct {
	// pb.UnimplementedMemobaseEngineServiceServer
	processor port.BufferProcessor
}

// NewEngineHandler creates a new gRPC handler.
func NewEngineHandler(processor port.BufferProcessor) *EngineHandler {
	return &EngineHandler{
		processor: processor,
	}
}

// ProcessBuffer handles manual/retry trigger for buffer processing
func (h *EngineHandler) ProcessBuffer(ctx context.Context, req interface{}) (interface{}, error) {
	// Call h.processor.ProcessBuffer
	return nil, nil
}

// GetPipelineStatus returns status of processing
func (h *EngineHandler) GetPipelineStatus(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

// GetProfileConfig gets project config
func (h *EngineHandler) GetProfileConfig(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

// UpdateProfileConfig updates project config
func (h *EngineHandler) UpdateProfileConfig(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}
