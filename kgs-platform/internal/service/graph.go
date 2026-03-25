package service

import (
	"context"
	"encoding/json"

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
	// Dummy appID for now
	appID := "system"
	var props map[string]any
	if req.PropertiesJson != "" {
		if err := json.Unmarshal([]byte(req.PropertiesJson), &props); err != nil {
			return nil, err
		}
	} else {
		props = make(map[string]any)
	}

	res, err := s.uc.CreateNode(ctx, appID, req.Label, props)
	if err != nil {
		return nil, err
	}

	resJSON, _ := json.Marshal(res)
	return &pb.CreateNodeReply{
		NodeId:         res["id"].(string),
		Label:          req.Label,
		PropertiesJson: string(resJSON),
	}, nil
}

func (s *GraphService) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeReply, error) {
	return &pb.GetNodeReply{}, nil
}

func (s *GraphService) UpdateNode(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.UpdateNodeReply, error) {
	props := make(map[string]any)
	for k, v := range req.Properties {
		props[k] = v
	}
	
	res, err := s.uc.UpdateNode(ctx, req.AppId, req.NodeId, props)
	if err != nil {
		return nil, err
	}
	
	resJSON, _ := json.Marshal(res)
	return &pb.UpdateNodeReply{
		Node: &pb.GraphNode{
			Id: req.NodeId,
			PropertiesJson: string(resJSON),
		},
	}, nil
}

func (s *GraphService) DeleteNode(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.DeleteNodeReply, error) {
	err := s.uc.DeleteNode(ctx, req.AppId, req.NodeId)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteNodeReply{Success: true}, nil
}

func (s *GraphService) CreateEdge(ctx context.Context, req *pb.CreateEdgeRequest) (*pb.CreateEdgeReply, error) {
	// Dummy appID for now
	appID := "system"
	var props map[string]any
	if req.PropertiesJson != "" {
		if err := json.Unmarshal([]byte(req.PropertiesJson), &props); err != nil {
			return nil, err
		}
	} else {
		props = make(map[string]any)
	}

	res, err := s.uc.CreateEdge(ctx, appID, req.RelationType, req.SourceNodeId, req.TargetNodeId, props)
	if err != nil {
		return nil, err
	}

	resJSON, _ := json.Marshal(res)
	return &pb.CreateEdgeReply{
		SourceNodeId:   req.SourceNodeId,
		TargetNodeId:   req.TargetNodeId,
		RelationType:   req.RelationType,
		PropertiesJson: string(resJSON),
	}, nil
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
