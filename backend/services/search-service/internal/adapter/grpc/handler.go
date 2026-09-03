package grpc

import (
    "context"

    searchpb "github.com/vnp-memory/api/proto/search/v1"
    "vnp-memory/services/search-service/internal/usecase"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type ObserveSearchHandler struct {
    searchpb.UnimplementedObserveSearchServiceServer
    smartSearch  *usecase.SmartSearchUseCase
    buildContext *usecase.BuildContextUseCase
    indexAdd     *usecase.IndexAddUseCase
    bm25         interface{ DocCount() int }
    vector       interface{ DocCount() int }
}

func (h *ObserveSearchHandler) SmartSearch(ctx context.Context, req *searchpb.SmartSearchRequest) (*searchpb.SmartSearchResponse, error) {
    resp, err := h.smartSearch.Execute(ctx, usecase.SmartSearchRequest{
        Query: req.Query, TenantID: req.TenantId, Project: req.Project,
        Limit: int(req.Limit),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "smart search: %v", err) }
    return mapSmartSearchResponse(resp), nil
}

func (h *ObserveSearchHandler) BuildContext(ctx context.Context, req *searchpb.ContextRequest) (*searchpb.ContextResponse, error) {
    resp, err := h.buildContext.Execute(ctx, usecase.ContextRequest{
        TenantID: req.TenantId, Project: req.Project, SessionID: req.SessionId,
        Query: req.Query, TokenBudget: int(req.TokenBudget),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "build context: %v", err) }
    return mapContextResponse(resp), nil
}

func (h *ObserveSearchHandler) IndexAdd(ctx context.Context, req *searchpb.IndexAddRequest) (*searchpb.IndexAddResponse, error) {
    err := h.indexAdd.Execute(ctx, usecase.IndexAddRequest{
        ObsID: req.ObsId, SessionID: req.SessionId, AgentID: req.AgentId,
        TenantID: req.TenantId, Title: req.Title, Facts: req.Facts, Concepts: req.Concepts,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "index add: %v", err) }
    return &searchpb.IndexAddResponse{Ok: true}, nil
}

func (h *ObserveSearchHandler) GetIndexStats(ctx context.Context, req *searchpb.GetIndexStatsRequest) (*searchpb.GetIndexStatsResponse, error) {
    return &searchpb.GetIndexStatsResponse{
        Bm25Documents:   int32(h.bm25.DocCount()),
        VectorDocuments: int32(h.vector.DocCount()),
        Bm25Loaded:      h.bm25.DocCount() > 0,
        VectorLoaded:    h.vector.DocCount() > 0,
    }, nil
}
