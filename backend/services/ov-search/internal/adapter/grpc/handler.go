package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"vnp-memory/ov-search/internal/domain"
	"vnp-memory/ov-search/internal/usecase/dto"
	"vnp-memory/ov-search/internal/usecase/port"
)

type OvSearchHandler struct {
	searchUC    port.SearchUseCase
	embeddingUC port.EmbeddingUseCase
	hotnessUC   port.HotnessUseCase
}

func NewOvSearchHandler(suc port.SearchUseCase, euc port.EmbeddingUseCase, huc port.HotnessUseCase) *OvSearchHandler {
	return &OvSearchHandler{
		searchUC:    suc,
		embeddingUC: euc,
		hotnessUC:   huc,
	}
}

// Pseudo-methods reflecting the proto definition

func (h *OvSearchHandler) HierarchicalSearch(ctx context.Context, req interface{}) (interface{}, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant id")
	}

	// Mock parsing req to DTO
	dtoReq := dto.SearchRequest{
		Query:      "mock",
		AccountID:  tenantID,
		MaxResults: 10,
	}

	res, err := h.searchUC.Search(ctx, dtoReq)
	if err != nil {
		return nil, mapError(err)
	}

	return res, nil
}

func (h *OvSearchHandler) RetrieveContext(ctx context.Context, req interface{}) (interface{}, error) {
	dtoReq := dto.ContextRequest{Path: "mock", ContextLevel: "L1"}
	res, err := h.searchUC.RetrieveContext(ctx, dtoReq)
	if err != nil {
		return nil, mapError(err)
	}
	return res, nil
}

func (h *OvSearchHandler) GetHotness(ctx context.Context, req interface{}) (interface{}, error) {
	tenantID, _ := extractTenantID(ctx)
	res, err := h.hotnessUC.Get(ctx, tenantID, []string{"mock"})
	if err != nil {
		return nil, mapError(err)
	}
	return res, nil
}

func (h *OvSearchHandler) UpsertEmbedding(ctx context.Context, req interface{}) (interface{}, error) {
	tenantID, _ := extractTenantID(ctx)
	dtoReq := dto.UpsertRequest{AccountID: tenantID, Content: "mock"}
	err := h.embeddingUC.Upsert(ctx, dtoReq)
	return nil, mapError(err)
}

func (h *OvSearchHandler) DeleteEmbedding(ctx context.Context, req interface{}) (interface{}, error) {
	tenantID, _ := extractTenantID(ctx)
	dtoReq := dto.DeleteRequest{AccountID: tenantID, Path: "mock"}
	err := h.embeddingUC.Delete(ctx, dtoReq)
	return nil, mapError(err)
}

func extractTenantID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}
	vals := md.Get("x-tenant-id")
	if len(vals) == 0 {
		return "", errors.New("missing x-tenant-id")
	}
	return vals[0], nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrIndexNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrEmbeddingFailed):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
