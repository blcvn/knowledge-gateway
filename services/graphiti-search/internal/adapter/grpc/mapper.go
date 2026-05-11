package grpc

import (
	"fmt"

	pb "vnp-memory/services/graphiti-search/internal/adapter/grpc/pb"
	"vnp-memory/services/graphiti-search/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapAttributes(meta map[string]any) map[string]string {
	if meta == nil {
		return nil
	}
	res := make(map[string]string, len(meta))
	for k, v := range meta {
		res[k] = fmt.Sprintf("%v", v)
	}
	return res
}

func MapError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case domain.ErrNoResults:
		return status.Error(codes.NotFound, err.Error())
	case domain.ErrInvalidQuery:
		return status.Error(codes.InvalidArgument, err.Error())
	case domain.ErrCacheUnavailable:
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func MapDomainRankedResultsToProto(results []domain.RankedResult) *pb.GraphSearchResponse {
	var protoNodes []*pb.Node
	for _, r := range results {
		protoNodes = append(protoNodes, &pb.Node{
			Id: r.EntityID,
			Summary: r.Content,
            Attributes: mapAttributes(r.Metadata),
		})
	}
	return &pb.GraphSearchResponse{Nodes: protoNodes}
}

func MapDomainSearchResultsToProto(results []domain.SearchResult) *pb.GraphSearchResponse {
	var protoNodes []*pb.Node
	for _, r := range results {
		protoNodes = append(protoNodes, &pb.Node{
			Id: r.EntityID,
			Summary: r.Content,
            Attributes: mapAttributes(r.Metadata),
		})
	}
	return &pb.GraphSearchResponse{Nodes: protoNodes}
}
