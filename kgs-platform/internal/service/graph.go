package service

import (
	"context"

	pb "kgs-platform/api/graph/v1"
	"kgs-platform/internal/biz"
)

type GraphService struct {
	pb.UnimplementedGraphServer
	uc *biz.GraphUsecase
}

func NewGraphService(uc *biz.GraphUsecase) *GraphService {
	return &GraphService{
		uc: uc,
	}
}

func (s *GraphService) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.CreateNodeReply, error) {
	return &pb.CreateNodeReply{}, nil
}
func (s *GraphService) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeReply, error) {
	return &pb.GetNodeReply{}, nil
}
func (s *GraphService) CreateEdge(ctx context.Context, req *pb.CreateEdgeRequest) (*pb.CreateEdgeReply, error) {
	return &pb.CreateEdgeReply{}, nil
}

func (s *GraphService) GetContext(ctx context.Context, req *pb.GetContextRequest) (*pb.GraphReply, error) {
	// Dummy appID for now; in reality, extract from AppContext injected by Auth Middleware
	appID := "demo-app"
	_, err := s.uc.GetContext(ctx, appID, req.NodeId, int(req.Depth), req.Direction)
	if err != nil {
		return nil, err
	}
	// TODO: map map[string]any to *pb.GraphReply
	return &pb.GraphReply{}, nil
}

func (s *GraphService) GetImpact(ctx context.Context, req *pb.GetImpactRequest) (*pb.GraphReply, error) {
	appID := "demo-app"
	_, err := s.uc.GetImpact(ctx, appID, req.NodeId, int(req.MaxDepth))
	if err != nil {
		return nil, err
	}
	// TODO: map map[string]any to *pb.GraphReply
	return &pb.GraphReply{}, nil
}

func (s *GraphService) GetCoverage(ctx context.Context, req *pb.GetCoverageRequest) (*pb.GraphReply, error) {
	appID := "demo-app"
	_, err := s.uc.GetCoverage(ctx, appID, req.NodeId, int(req.MaxDepth))
	if err != nil {
		return nil, err
	}
	// TODO: map map[string]any to *pb.GraphReply
	return &pb.GraphReply{}, nil
}

func (s *GraphService) GetSubgraph(ctx context.Context, req *pb.GetSubgraphRequest) (*pb.GraphReply, error) {
	appID := "demo-app"
	_, err := s.uc.GetSubgraph(ctx, appID, req.NodeIds)
	if err != nil {
		return nil, err
	}
	// TODO: map map[string]any to *pb.GraphReply
	return &pb.GraphReply{}, nil
}
