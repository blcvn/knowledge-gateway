package service

import (
	"context"
	"encoding/json"

	pb "kgs-platform/api/graph/v1"
	"kgs-platform/internal/biz"

	"google.golang.org/grpc/metadata"
)

// extractAppID reads the x-kgs-app-id header from incoming gRPC metadata.
// Falls back to "system" if not present (unauthenticated / legacy calls).
func extractAppID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if vals := md.Get("x-kgs-app-id"); len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return "system"
}

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
	// Extract app_id from gRPC incoming metadata (set by ui-knowledge-service / doc_to_kg)
	appID := extractAppID(ctx)
	// Fallback: if app_id is embedded in properties JSON, prefer that
	var props map[string]any
	if req.PropertiesJson != "" {
		if err := json.Unmarshal([]byte(req.PropertiesJson), &props); err != nil {
			return nil, err
		}
		// If client injected app_id directly into props (doc_to_kg pattern), use it
		if v, ok := props["app_id"].(string); ok && v != "" {
			appID = v
		}
	} else {
		props = make(map[string]any)
	}

	res, err := s.uc.CreateNode(ctx, appID, req.Label, props)
	if err != nil {
		return nil, err
	}

	resJSON, _ := json.Marshal(res)
	// Safely extract nodeId — may come from props["id"] if not returned by Neo4j internal id
	nodeId, _ := res["id"].(string)
	if nodeId == "" {
		nodeId, _ = props["id"].(string)
	}
	return &pb.CreateNodeReply{
		NodeId:         nodeId,
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
			Id:             req.NodeId,
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
	// Extract app_id from gRPC incoming metadata
	appID := extractAppID(ctx)
	var props map[string]any
	if req.PropertiesJson != "" {
		if err := json.Unmarshal([]byte(req.PropertiesJson), &props); err != nil {
			return nil, err
		}
		// Prefer app_id embedded in props if present
		if v, ok := props["app_id"].(string); ok && v != "" {
			appID = v
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

// ── MergeNode ──────────────────────────────────────────────────────────────────
// MergeNode upserts a node by (app_id, label, name/id key). Delegates to
// CreateNode which already uses Cypher MERGE under the hood.
func (s *GraphService) MergeNode(ctx context.Context, req *pb.MergeNodeRequest) (*pb.MergeNodeReply, error) {
	appID := req.AppId
	if appID == "" {
		appID = extractAppID(ctx)
	}

	var props map[string]any
	if req.PropertiesJson != "" {
		if err := json.Unmarshal([]byte(req.PropertiesJson), &props); err != nil {
			return nil, err
		}
	} else {
		props = make(map[string]any)
	}
	// Allow app_id embedded in props to override
	if v, ok := props["app_id"].(string); ok && v != "" {
		appID = v
	}

	res, err := s.uc.CreateNode(ctx, appID, req.Label, props)
	if err != nil {
		return nil, err
	}

	resJSON, _ := json.Marshal(res)
	nodeID, _ := res["id"].(string)
	if nodeID == "" {
		nodeID, _ = props["id"].(string)
	}
	return &pb.MergeNodeReply{
		NodeId:         nodeID,
		Label:          req.Label,
		PropertiesJson: string(resJSON),
	}, nil
}

// ── BatchMergeNodes ────────────────────────────────────────────────────────────
func (s *GraphService) BatchMergeNodes(ctx context.Context, req *pb.BatchMergeNodesRequest) (*pb.BatchMergeNodesReply, error) {
	appID := req.AppId
	if appID == "" {
		appID = extractAppID(ctx)
	}

	ids := make([]string, 0, len(req.Nodes))
	for _, spec := range req.Nodes {
		var props map[string]any
		if spec.PropertiesJson != "" {
			if err := json.Unmarshal([]byte(spec.PropertiesJson), &props); err != nil {
				return nil, err
			}
		} else {
			props = make(map[string]any)
		}
		res, err := s.uc.CreateNode(ctx, appID, spec.Label, props)
		if err != nil {
			return nil, err
		}
		id, _ := res["id"].(string)
		ids = append(ids, id)
	}
	return &pb.BatchMergeNodesReply{NodeIds: ids}, nil
}

// ── MergeEdge ──────────────────────────────────────────────────────────────────
func (s *GraphService) MergeEdge(ctx context.Context, req *pb.MergeEdgeRequest) (*pb.MergeEdgeReply, error) {
	appID := req.AppId
	if appID == "" {
		appID = extractAppID(ctx)
	}

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
	edgeID, _ := res["id"].(string)
	return &pb.MergeEdgeReply{EdgeId: edgeID}, nil
}

// ── Query ──────────────────────────────────────────────────────────────────────
func (s *GraphService) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryReply, error) {
	appID := req.AppId
	if appID == "" {
		appID = extractAppID(ctx)
	}

	var params map[string]any
	if req.ParamsJson != "" {
		if err := json.Unmarshal([]byte(req.ParamsJson), &params); err != nil {
			return nil, err
		}
	}

	res, err := s.uc.ExecuteQuery(ctx, appID, req.Cypher, params)
	if err != nil {
		return nil, err
	}

	resJSON, _ := json.Marshal(res)
	return &pb.QueryReply{ResultJson: string(resJSON)}, nil
}
