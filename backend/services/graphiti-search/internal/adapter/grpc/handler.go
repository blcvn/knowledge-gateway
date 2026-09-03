package grpc

import (
	"context"

	pb "vnp-memory/services/graphiti-search/internal/adapter/grpc/pb"
	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type SearchServiceServer struct {
	pb.UnimplementedGraphitiSearchServiceServer
	hybridUC    *usecase.HybridSearchUseCase
	nodeUC      *usecase.NodeSearchUseCase
	edgeUC      *usecase.EdgeSearchUseCase
	communityUC *usecase.CommunitySearchUseCase
}

func NewSearchServiceServer(
	hybridUC *usecase.HybridSearchUseCase,
	nodeUC *usecase.NodeSearchUseCase,
	edgeUC *usecase.EdgeSearchUseCase,
	communityUC *usecase.CommunitySearchUseCase,
) pb.GraphitiSearchServiceServer {
	return &SearchServiceServer{
		hybridUC:    hybridUC,
		nodeUC:      nodeUC,
		edgeUC:      edgeUC,
		communityUC: communityUC,
	}
}

func (s *SearchServiceServer) extractGroupID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata is not provided")
	}
	values := md.Get("x-tenant-id")
	if len(values) == 0 || values[0] == "" {
		return "", status.Error(codes.Unauthenticated, "x-tenant-id is required")
	}
	return values[0], nil
}

func (s *SearchServiceServer) Search(ctx context.Context, req *pb.GraphSearchRequest) (*pb.GraphSearchResponse, error) {
	groupID, err := s.extractGroupID(ctx)
	if err != nil {
		return nil, err
	}
	
	methods := []domain.SearchMethod{domain.MethodCosine, domain.MethodBM25}
	rerankers := []domain.RerankerType{domain.RerankerRRF}

	query := domain.SearchQuery{
		Query:     req.Query,
		GroupID:   groupID,
		Methods:   methods,
		Rerankers: rerankers,
		Limit:     int(req.Limit),
		EntityLabels: req.EntityTypes,
	}

	res, err := s.hybridUC.Execute(ctx, query)
	if err != nil {
		return nil, MapError(err)
	}

	return MapDomainRankedResultsToProto(res), nil
}
