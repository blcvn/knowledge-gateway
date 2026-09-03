package grpc

import (
	pb "github.com/vnp-memory/api/proto/search/v1"
	"vnp-memory/services/search-service/internal/usecase"
)

func mapSmartSearchResponse(resp *usecase.SmartSearchResponse) *pb.SmartSearchResponse {
    var results []*pb.SearchResultProto
    for _, r := range resp.Results {
        results = append(results, &pb.SearchResultProto{
            Id:            r.DocID,
            SessionId:     r.SessionID,
            ObsType:       r.ObsType,
            Title:         r.Title,
            Narrative:     r.Narrative,
            Facts:         r.Facts,
            Concepts:      r.Concepts,
            CombinedScore: r.CombinedScore,
            Bm25Score:     r.BM25Score,
            VectorScore:   r.VectorScore,
        })
    }
    return &pb.SmartSearchResponse{
        Results: results,
        TookMs:  resp.TookMs,
    }
}

func mapContextResponse(resp *usecase.ContextResponse) *pb.ContextResponse {
    var blocks []*pb.ContextBlock
    for _, b := range resp.Blocks {
        blocks = append(blocks, &pb.ContextBlock{
            Type:    b.Type,
            Content: b.Content,
            Tokens:  int32(b.Tokens),
            Source:  b.Source,
        })
    }
    return &pb.ContextResponse{
        Blocks:      blocks,
        TotalTokens: int32(resp.TotalTokens),
        Formatted:   resp.Formatted,
    }
}
