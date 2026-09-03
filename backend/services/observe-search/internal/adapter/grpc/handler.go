package grpc

import (
	"context"

	searchpb "github.com/vnp-memory/api/proto/search/v1"
	"github.com/vnp-memory/services/observe-search/internal/index"
	intsearch "github.com/vnp-memory/services/observe-search/internal/search"
	pkg_search "github.com/vnp-memory/pkg/search"
)

// Handler implements the ObserveSearchService gRPC server.
type Handler struct {
	searchpb.UnimplementedObserveSearchServiceServer
	smartSearch *intsearch.SmartSearch
	indexMgr    *index.Manager
	bm25        *pkg_search.BM25Index
	vector      *pkg_search.VectorIndex
}

func NewHandler(smartSearch *intsearch.SmartSearch, indexMgr *index.Manager, bm25 *pkg_search.BM25Index, vector *pkg_search.VectorIndex) *Handler {
	return &Handler{smartSearch: smartSearch, indexMgr: indexMgr, bm25: bm25, vector: vector}
}

func (h *Handler) SmartSearch(ctx context.Context, req *searchpb.SmartSearchRequest) (*searchpb.SmartSearchResponse, error) {
	res, err := h.smartSearch.Execute(ctx, intsearch.SmartSearchRequest{
		Query:    req.Query,
		TenantID: req.TenantId,
		Limit:    int(req.Limit),
	})
	if err != nil {
		return nil, err
	}
	resp := &searchpb.SmartSearchResponse{TookMs: res.TookMs}
	for _, r := range res.Results {
		resp.Results = append(resp.Results, &searchpb.SearchResultProto{
			Id:            r.DocID,
			SessionId:     r.SessionID,
			CombinedScore: r.CombinedScore,
		})
	}
	return resp, nil
}

func (h *Handler) IndexAdd(ctx context.Context, req *searchpb.IndexAddRequest) (*searchpb.IndexAddResponse, error) {
	err := h.indexMgr.Add(ctx, index.IndexAddRequest{
		ObsID:     req.GetObsId(),
		SessionID: req.GetSessionId(),
		AgentID:   req.GetAgentId(),
		Title:     req.GetTitle(),
		Facts:     req.GetFacts(),
		Concepts:  req.GetConcepts(),
	})
	return &searchpb.IndexAddResponse{}, err
}

func (h *Handler) IndexRemove(ctx context.Context, req *searchpb.IndexRemoveRequest) (*searchpb.IndexRemoveResponse, error) {
	err := h.indexMgr.Remove(ctx, req.GetDocId())
	return &searchpb.IndexRemoveResponse{}, err
}
