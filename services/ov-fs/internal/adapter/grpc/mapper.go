package grpc

import (
	"vnp-memory/services/ov-fs/internal/domain/model"
	"vnp-memory/services/ov-fs/internal/usecase/dto"
	// Assuming the proto path based on architecture doc
	// pb "vnp-memory/api/gen/go/openviking/fs/v1"
)

// This mapper would map proto requests to dto requests.
// Since we don't have the exact pb types, these functions act as placeholders for the logic.

func MapContextLevel(level string) model.ContextLevel {
	switch level {
	case "L0":
		return model.ContextLevelL0
	case "L1":
		return model.ContextLevelL1
	case "L2":
		return model.ContextLevelL2
	default:
		return model.ContextLevelL2
	}
}

// MapReadFileRequest maps from proto to DTO
func MapReadFileRequest(accountID string, req interface{}) dto.ReadFileRequest {
	// mock implementation
	return dto.ReadFileRequest{
		AccountID:    accountID,
		Path:         "mock",
		ContextLevel: model.ContextLevelL2,
	}
}
