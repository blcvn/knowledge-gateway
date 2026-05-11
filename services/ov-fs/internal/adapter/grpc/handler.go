package grpc

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"

	"vnp-memory/services/ov-fs/internal/usecase/dto"
	"vnp-memory/services/ov-fs/internal/usecase/port"
)

type OvFsHandler struct {
	// pb.UnimplementedOvFsServiceServer
	fileUC     port.FileUseCase
	dirUC      port.DirUseCase
	searchUC   port.SearchUseCase
	moveUC     port.MoveUseCase
	relationUC port.RelationUseCase
}

func NewOvFsHandler(
	fileUC port.FileUseCase,
	dirUC port.DirUseCase,
	searchUC port.SearchUseCase,
	moveUC port.MoveUseCase,
	relationUC port.RelationUseCase,
) *OvFsHandler {
	return &OvFsHandler{
		fileUC:     fileUC,
		dirUC:      dirUC,
		searchUC:   searchUC,
		moveUC:     moveUC,
		relationUC: relationUC,
	}
}

func extractAccountID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata missing")
	}
	vals := md.Get("x-tenant-id")
	if len(vals) == 0 {
		return "", status.Error(codes.Unauthenticated, "x-tenant-id missing")
	}
	return vals[0], nil
}

// Mock ReadFile RPC implementation
func (h *OvFsHandler) ReadFile(ctx context.Context, req interface{}) (interface{}, error) {
	accountID, err := extractAccountID(ctx)
	if err != nil {
		return nil, err
	}

	dtoReq := MapReadFileRequest(accountID, req)
	resp, err := h.fileUC.ReadFile(ctx, dtoReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read file failed: %v", err)
	}

	return resp, nil
}

// Other handlers (WriteFile, DeleteFile, MkDir, ListDir, Tree, Grep, Glob, Move, GetRelations)
// would follow the same pattern: Extract accountID -> Map to DTO -> Call Usecase -> Return mapped proto response.
