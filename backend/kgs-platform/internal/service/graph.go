package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/graph/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/analytics"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/batch"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/overlay"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/search"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server/middleware"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphService struct {
	pb.UnimplementedGraphServer
	uc           GraphUsecase
	batchUC      *batch.Usecase
	graphBatchUC *batch.GraphBatchHandler
	entityReader GraphEntityReader
	searchUC     search.SearchEngine
	overlay      overlay.OverlayManager
	version      version.VersionManager
	analytics    analytics.AnalyticsEngine
	projection   projection.ProjectionEngine
	viewResolver *biz.ViewResolver
}

type GraphEntityReader interface {
	GetEntity(ctx context.Context, appID, tenantID, entityID string) (map[string]any, error)
	EnrichWithFreshVersions(ctx context.Context, appID, tenantID string, entities []map[string]any) ([]map[string]any, error)
	ListEntities(
		ctx context.Context,
		appID, tenantID string,
		limit int,
		cursorID string,
		entityType string,
		sourceFile string,
		domain string,
		provenanceType string,
		versionID string,
		propertyKey string,
		propertyValue string,
		isDeleted bool,
	) ([]map[string]any, string, bool, int64, error)
	LookupEntities(
		ctx context.Context,
		appID, tenantID string,
		entityType string,
		sourceFile string,
		matchMode string,
		limit int,
		properties map[string]string,
	) ([]map[string]any, int64, error)
	ListEdges(
		ctx context.Context,
		appID, tenantID string,
		limit int,
		cursorID string,
		relationType string,
		sourceFile string,
		fromEntityID string,
		toEntityID string,
		versionID string,
		isDeleted bool,
	) ([]map[string]any, string, bool, int64, error)
	GetNodesByLabel(
		ctx context.Context,
		appID, tenantID string,
		label string,
		limit int,
		cursorID string,
	) ([]map[string]any, string, bool, int64, error)
	GetNamespaceStats(
		ctx context.Context,
		appID, tenantID string,
		labels []string,
	) (totalNodes int64, totalEdges int64, byLabel map[string]int64, err error)
	QueryNodes(
		ctx context.Context,
		appID, tenantID string,
		filter data.QueryNodesFilter,
	) ([]map[string]any, int64, error)
}

type GraphUsecase interface {
	CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error)
	GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
	UpdateNode(ctx context.Context, appID, tenantID, nodeID string, mergeProperties map[string]any, newLabel string) (map[string]any, error)
	CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error)
	DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (int, error)
	DeleteEdge(ctx context.Context, appID, tenantID, edgeID string) error
	BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (deleted, edgesRemoved int, err error)
	GetContext(ctx context.Context, appID, tenantID string, nodeID string, depth int, direction string) (map[string]any, error)
	GetImpact(ctx context.Context, appID, tenantID string, nodeID string, maxDepth int) (map[string]any, error)
	GetCoverage(ctx context.Context, appID, tenantID string, nodeID string, maxDepth int) (map[string]any, error)
	GetSubgraph(ctx context.Context, appID, tenantID string, nodeIDs []string) (map[string]any, error)
	GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error)
}

func NewGraphService(
	uc GraphUsecase,
	batchUC *batch.Usecase,
	searchUC search.SearchEngine,
	overlayMgr overlay.OverlayManager,
	versionMgr version.VersionManager,
	analyticsEngine analytics.AnalyticsEngine,
	viewResolver *biz.ViewResolver,
	projectionEngine projection.ProjectionEngine,
) *GraphService {
	return &GraphService{
		uc:           uc,
		batchUC:      batchUC,
		searchUC:     searchUC,
		overlay:      overlayMgr,
		version:      versionMgr,
		analytics:    analyticsEngine,
		viewResolver: viewResolver,
		projection:   projectionEngine,
	}
}

func NewGraphServiceWithGraphBatch(
	uc GraphUsecase,
	batchUC *batch.Usecase,
	graphBatchUC *batch.GraphBatchHandler,
	searchUC search.SearchEngine,
	overlayMgr overlay.OverlayManager,
	versionMgr version.VersionManager,
	analyticsEngine analytics.AnalyticsEngine,
	viewResolver *biz.ViewResolver,
	projectionEngine projection.ProjectionEngine,
) *GraphService {
	svc := NewGraphService(uc, batchUC, searchUC, overlayMgr, versionMgr, analyticsEngine, viewResolver, projectionEngine)
	svc.graphBatchUC = graphBatchUC
	return svc
}

func NewGraphServiceWithGraphBatchAndReader(
	uc GraphUsecase,
	batchUC *batch.Usecase,
	graphBatchUC *batch.GraphBatchHandler,
	entityReader GraphEntityReader,
	searchUC search.SearchEngine,
	overlayMgr overlay.OverlayManager,
	versionMgr version.VersionManager,
	analyticsEngine analytics.AnalyticsEngine,
	viewResolver *biz.ViewResolver,
	projectionEngine projection.ProjectionEngine,
) *GraphService {
	svc := NewGraphServiceWithGraphBatch(uc, batchUC, graphBatchUC, searchUC, overlayMgr, versionMgr, analyticsEngine, viewResolver, projectionEngine)
	svc.entityReader = entityReader
	return svc
}

func (s *GraphService) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.CreateNodeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	props, err := parseJSON(req.PropertiesJson)
	if err != nil {
		return nil, err
	}
	out, err := s.uc.CreateNode(ctx, appCtx.AppID, appCtx.TenantID, req.Label, props)
	if err != nil {
		return nil, err
	}
	return &pb.CreateNodeReply{
		NodeId:         mapString(out, "id"),
		Label:          req.Label,
		PropertiesJson: mustJSON(out),
	}, nil
}
func (s *GraphService) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if s.entityReader != nil {
		out, err = s.entityReader.GetEntity(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId)
	} else {
		out, err = s.uc.GetNode(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId)
	}
	if err != nil {
		return nil, err
	}
	out, err = s.applyProjectionToSingleNode(ctx, appCtx, out)
	if err != nil {
		return nil, err
	}
	return &pb.GetNodeReply{
		NodeId:         mapString(out, "id"),
		Label:          mapString(out, "label"),
		PropertiesJson: mustJSON(out),
	}, nil
}
func (s *GraphService) UpdateNode(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.UpdateNodeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	props, err := parseJSON(req.PropertiesJson)
	if err != nil {
		return nil, err
	}
	out, err := s.uc.UpdateNode(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId, props, req.Label)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateNodeReply{
		NodeId:         mapString(out, "id"),
		Label:          mapString(out, "label"),
		PropertiesJson: mustJSON(out),
	}, nil
}
func (s *GraphService) CreateEdge(ctx context.Context, req *pb.CreateEdgeRequest) (*pb.CreateEdgeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	props, err := parseJSON(req.PropertiesJson)
	if err != nil {
		return nil, err
	}
	stdlog.Printf("[KGS][GraphService] CreateEdge start app_id=%s tenant_id=%s relation=%s source=%s target=%s props_keys=%d",
		appCtx.AppID, appCtx.TenantID, req.RelationType, req.SourceNodeId, req.TargetNodeId, len(props))
	out, err := s.uc.CreateEdge(ctx, appCtx.AppID, appCtx.TenantID, req.RelationType, req.SourceNodeId, req.TargetNodeId, props)
	if err != nil {
		stdlog.Printf("[KGS][GraphService] CreateEdge failed app_id=%s tenant_id=%s relation=%s source=%s target=%s err=%v",
			appCtx.AppID, appCtx.TenantID, req.RelationType, req.SourceNodeId, req.TargetNodeId, err)
		return nil, err
	}
	stdlog.Printf("[KGS][GraphService] CreateEdge done app_id=%s tenant_id=%s relation=%s source=%s target=%s edge_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, req.RelationType, req.SourceNodeId, req.TargetNodeId, mapString(out, "id"), time.Since(started))
	return &pb.CreateEdgeReply{
		EdgeId:         mapString(out, "id"),
		SourceNodeId:   req.SourceNodeId,
		TargetNodeId:   req.TargetNodeId,
		RelationType:   req.RelationType,
		PropertiesJson: mustJSON(out),
	}, nil
}

func (s *GraphService) DeleteNode(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.DeleteNodeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	edgesRemoved, err := s.uc.DeleteNode(ctx, appCtx.AppID, appCtx.TenantID, req.GetNodeId())
	if err != nil {
		return nil, err
	}
	return &pb.DeleteNodeReply{
		NodeId:       req.GetNodeId(),
		EdgesRemoved: int32(edgesRemoved),
	}, nil
}

func (s *GraphService) DeleteEdge(ctx context.Context, req *pb.DeleteEdgeRequest) (*pb.DeleteEdgeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DeleteEdge(ctx, appCtx.AppID, appCtx.TenantID, req.GetEdgeId()); err != nil {
		return nil, err
	}
	return &pb.DeleteEdgeReply{EdgeId: req.GetEdgeId()}, nil
}

func (s *GraphService) BatchDeleteNodes(ctx context.Context, req *pb.BatchDeleteNodesRequest) (*pb.BatchDeleteNodesReply, error) {
	if req == nil || len(req.GetNodeIds()) == 0 {
		return nil, kerrors.BadRequest("ERR_BATCH_EMPTY", "node_ids is required")
	}
	if len(req.GetNodeIds()) > 1000 {
		return nil, kerrors.BadRequest("ERR_BATCH_TOO_LARGE", "max 1000 nodes per batch delete")
	}

	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	deleted, edgesRemoved, err := s.uc.BatchDeleteNodes(ctx, appCtx.AppID, appCtx.TenantID, req.GetNodeIds())
	if err != nil {
		return nil, err
	}
	return &pb.BatchDeleteNodesReply{
		Deleted:      int32(deleted),
		EdgesRemoved: int32(edgesRemoved),
		NotFound:     int32(len(req.GetNodeIds()) - deleted),
	}, nil
}

func (s *GraphService) GetContext(ctx context.Context, req *pb.GetContextRequest) (*pb.GraphReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.uc.GetContext(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId, int(req.Depth), req.Direction)
	if err != nil {
		return nil, err
	}
	reply, err := s.enrichGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		return nil, err
	}
	reply, err = s.ensureReplyContainsNodes(ctx, appCtx, reply, []string{req.GetNodeId()})
	if err != nil {
		return nil, err
	}
	projectedReply, err := s.applyProjectionToGraphReply(ctx, appCtx, reply)
	if err != nil {
		return nil, err
	}
	reply, err = applyPagination(ctx, projectedReply, req.PageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *GraphService) GetImpact(ctx context.Context, req *pb.GetImpactRequest) (*pb.GraphReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.uc.GetImpact(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId, int(req.MaxDepth))
	if err != nil {
		return nil, err
	}
	reply, err := s.enrichGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		return nil, err
	}
	projectedReply, err := s.applyProjectionToGraphReply(ctx, appCtx, reply)
	if err != nil {
		return nil, err
	}
	reply, err = applyPagination(ctx, projectedReply, req.PageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *GraphService) GetCoverage(ctx context.Context, req *pb.GetCoverageRequest) (*pb.GraphReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.uc.GetCoverage(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId, int(req.MaxDepth))
	if err != nil {
		return nil, err
	}
	reply, err := s.enrichGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		return nil, err
	}
	projectedReply, err := s.applyProjectionToGraphReply(ctx, appCtx, reply)
	if err != nil {
		return nil, err
	}
	reply, err = applyPagination(ctx, projectedReply, req.PageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *GraphService) GetSubgraph(ctx context.Context, req *pb.GetSubgraphRequest) (*pb.GraphReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	stdlog.Printf("[KGS][GraphService] GetSubgraph start app_id=%s tenant_id=%s requested_node_ids=%d",
		appCtx.AppID, appCtx.TenantID, len(req.NodeIds))
	result, err := s.uc.GetSubgraph(ctx, appCtx.AppID, appCtx.TenantID, req.NodeIds)
	if err != nil {
		stdlog.Printf("[KGS][GraphService] GetSubgraph failed app_id=%s tenant_id=%s requested_node_ids=%d err=%v",
			appCtx.AppID, appCtx.TenantID, len(req.NodeIds), err)
		return nil, err
	}
	reply, err := s.enrichGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		stdlog.Printf("[KGS][GraphService] GetSubgraph enrich failed app_id=%s tenant_id=%s requested_node_ids=%d err=%v",
			appCtx.AppID, appCtx.TenantID, len(req.NodeIds), err)
		return nil, err
	}
	reply, err = s.ensureReplyContainsNodes(ctx, appCtx, reply, req.GetNodeIds())
	if err != nil {
		stdlog.Printf("[KGS][GraphService] GetSubgraph fallback hydrate failed app_id=%s tenant_id=%s requested_node_ids=%d err=%v",
			appCtx.AppID, appCtx.TenantID, len(req.NodeIds), err)
		return nil, err
	}
	reply, err = s.applyProjectionToGraphReply(ctx, appCtx, reply)
	if err != nil {
		stdlog.Printf("[KGS][GraphService] GetSubgraph projection failed app_id=%s tenant_id=%s requested_node_ids=%d err=%v",
			appCtx.AppID, appCtx.TenantID, len(req.NodeIds), err)
		return nil, err
	}
	stdlog.Printf("[KGS][GraphService] GetSubgraph done app_id=%s tenant_id=%s requested_node_ids=%d returned_nodes=%d returned_edges=%d duration=%s",
		appCtx.AppID, appCtx.TenantID, len(req.NodeIds), len(reply.GetNodes()), len(reply.GetEdges()), time.Since(started))
	return reply, nil
}

func (s *GraphService) GetFullGraph(ctx context.Context, req *pb.GetFullGraphRequest) (*pb.GetFullGraphResponse, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		req = &pb.GetFullGraphRequest{}
	}

	appID := strings.TrimSpace(req.GetAppId())
	if appID == "" {
		appID = appCtx.AppID
	}
	tenantID := strings.TrimSpace(req.GetTenantId())
	if tenantID == "" {
		tenantID = appCtx.TenantID
	}
	if appID != appCtx.AppID || tenantID != appCtx.TenantID {
		return nil, kerrors.BadRequest("ERR_SCOPE_MISMATCH", "app_id/tenant_id mismatch with auth context")
	}

	started := time.Now()
	stdlog.Printf("[KGS][GraphService] GetFullGraph start app_id=%s tenant_id=%s node_limit=%d node_offset=%d",
		appID, tenantID, req.GetNodeLimit(), req.GetNodeOffset())

	result, err := s.uc.GetFullGraph(ctx, appID, tenantID, int(req.GetNodeLimit()), int(req.GetNodeOffset()))
	if err != nil {
		stdlog.Printf("[KGS][GraphService] GetFullGraph failed app_id=%s tenant_id=%s err=%v", appID, tenantID, err)
		return nil, err
	}

	reply := &pb.GetFullGraphResponse{
		Nodes:      make([]*pb.GraphNode, 0, len(result.Nodes)),
		Edges:      make([]*pb.GraphEdge, 0, len(result.Edges)),
		TotalNodes: int32(result.TotalNodes),
		TotalEdges: int32(result.TotalEdges),
	}
	for _, node := range result.Nodes {
		reply.Nodes = append(reply.Nodes, &pb.GraphNode{
			Id:             node.ID,
			Label:          primaryNodeLabel(node.Labels),
			PropertiesJson: mustJSON(node.Properties),
			Properties:     stringifyMap(node.Properties),
		})
	}
	for _, edge := range result.Edges {
		reply.Edges = append(reply.Edges, &pb.GraphEdge{
			Id:             edge.ID,
			Source:         edge.SourceNodeID,
			Target:         edge.TargetNodeID,
			Type:           edge.RelationType,
			PropertiesJson: mustJSON(edge.Properties),
			RelationType:   edge.RelationType,
			SourceNodeId:   edge.SourceNodeID,
			TargetNodeId:   edge.TargetNodeID,
			Properties:     stringifyMap(edge.Properties),
		})
	}
	graphReply, err := s.enrichGraphReply(ctx, middleware.AppContext{AppID: appID, TenantID: tenantID}, &pb.GraphReply{
		Nodes: reply.Nodes,
		Edges: reply.Edges,
	})
	if err != nil {
		return nil, err
	}
	reply.Nodes = graphReply.Nodes
	reply.Edges = graphReply.Edges

	stdlog.Printf("[KGS][GraphService] GetFullGraph done app_id=%s tenant_id=%s returned_nodes=%d returned_edges=%d total_nodes=%d total_edges=%d duration=%s",
		appID, tenantID, len(reply.GetNodes()), len(reply.GetEdges()), reply.GetTotalNodes(), reply.GetTotalEdges(), time.Since(started))

	return reply, nil
}

func (s *GraphService) BatchUpsertEntities(ctx context.Context, req *pb.BatchUpsertRequest) (*pb.BatchUpsertReply, error) {
	if s.batchUC == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "batch usecase is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	stdlog.Printf("[KGS][GraphService] BatchUpsert start app_id=%s tenant_id=%s entities=%d labels=%s",
		appCtx.AppID, appCtx.TenantID, len(req.Entities), summarizeBatchLabels(req.Entities))

	entities := make([]batch.Entity, 0, len(req.Entities))
	for i, item := range req.Entities {
		props, err := parseJSON(item.PropertiesJson)
		if err != nil {
			return nil, fmt.Errorf("invalid properties_json at entities[%d]: %w", i, err)
		}
		entities = append(entities, batch.Entity{
			Label:      item.Label,
			Properties: props,
		})
	}

	out, err := s.batchUC.Execute(ctx, batch.BatchUpsertRequest{
		AppID:    appCtx.AppID,
		TenantID: appCtx.TenantID,
		Entities: entities,
	})
	if err != nil {
		stdlog.Printf("[KGS][GraphService] BatchUpsert failed app_id=%s tenant_id=%s entities=%d err=%v",
			appCtx.AppID, appCtx.TenantID, len(req.Entities), err)
		return nil, err
	}
	stdlog.Printf("[KGS][GraphService] BatchUpsert done app_id=%s tenant_id=%s entities=%d created=%d skipped=%d updated=%d duration=%s",
		appCtx.AppID, appCtx.TenantID, len(req.Entities), out.Created, out.Skipped, out.Updated, time.Since(started))

	if tr, ok := transport.FromServerContext(ctx); ok {
		tr.ReplyHeader().Set("X-Batch-Created", fmt.Sprint(out.Created))
	}
	return &pb.BatchUpsertReply{
		Created: int32(out.Created),
		Updated: int32(out.Updated),
		Skipped: int32(out.Skipped),
	}, nil
}

func (s *GraphService) BatchUpsertGraph(ctx context.Context, req *pb.BatchUpsertGraphRequest) (*pb.BatchUpsertGraphReply, error) {
	if s.graphBatchUC == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "graph batch handler is not configured")
	}
	if req == nil {
		req = &pb.BatchUpsertGraphRequest{}
	}

	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}

	entities := make([]batch.Entity, 0, len(req.GetEntities()))
	for i, item := range req.GetEntities() {
		props, err := parseJSON(item.GetPropertiesJson())
		if err != nil {
			return nil, fmt.Errorf("invalid properties_json at entities[%d]: %w", i, err)
		}
		entities = append(entities, batch.Entity{
			Label:      item.GetLabel(),
			Properties: props,
		})
	}

	edges := make([]batch.Edge, 0, len(req.GetEdges()))
	for i, item := range req.GetEdges() {
		props, err := parseJSON(item.GetPropertiesJson())
		if err != nil {
			return nil, fmt.Errorf("invalid properties_json at edges[%d]: %w", i, err)
		}
		edges = append(edges, batch.Edge{
			EdgeID:       item.GetEdgeId(),
			FromEntityID: item.GetFromEntityId(),
			ToEntityID:   item.GetToEntityId(),
			RelationType: item.GetRelationType(),
			Properties:   props,
			Confidence:   item.GetConfidence(),
			VersionID:    item.GetVersionId(),
		})
	}

	batchReq := batch.GraphBatchRequest{
		Entities:       entities,
		Edges:          edges,
		ConflictPolicy: req.GetConflictPolicy(),
	}
	if overlayID := strings.TrimSpace(req.GetOverlayId()); overlayID != "" {
		batchReq.OverlayID = &overlayID
	}

	result, err := s.graphBatchUC.UpsertGraph(ctx, batchReq, appCtx.AppID, appCtx.TenantID)
	reply := &pb.BatchUpsertGraphReply{}
	if result != nil {
		reply = &pb.BatchUpsertGraphReply{
			EntitiesCreated: int32(result.EntitiesCreated),
			EntitiesUpdated: int32(result.EntitiesUpdated),
			EntitiesSkipped: int32(result.EntitiesSkipped),
			EdgesCreated:    int32(result.EdgesCreated),
			EdgesSkipped:    int32(result.EdgesSkipped),
			Conflicted:      int32(result.Conflicted),
			Errors:          append([]string(nil), result.Errors...),
		}
	}
	if err != nil {
		if errors.Is(err, batch.ErrOverlayEntityIDRequired) || errors.Is(err, batch.ErrOverlayEdgeIDRequired) {
			return reply, kerrors.BadRequest("ERR_GRAPH_BATCH_INVALID_INPUT", err.Error())
		}
		return reply, kerrors.Conflict("ERR_GRAPH_BATCH_CONFLICT", err.Error())
	}
	return reply, nil
}

// BatchUpsertGraphHTTP handles HTTP-only graph batch upsert until protobuf API is extended.
func (s *GraphService) BatchUpsertGraphHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.graphBatchUC == nil {
		http.Error(w, "graph batch handler is not configured", http.StatusInternalServerError)
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		appCtx, err = inferAppContextForKGBatch(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}

	var req batch.GraphBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	pbReq := &pb.BatchUpsertGraphRequest{
		Entities:       make([]*pb.BatchUpsertGraphEntity, 0, len(req.Entities)),
		Edges:          make([]*pb.BatchUpsertGraphEdge, 0, len(req.Edges)),
		ConflictPolicy: req.ConflictPolicy,
	}
	if req.OverlayID != nil {
		pbReq.OverlayId = *req.OverlayID
	}
	for _, item := range req.Entities {
		pbReq.Entities = append(pbReq.Entities, &pb.BatchUpsertGraphEntity{
			Label:          item.Label,
			PropertiesJson: mustJSON(item.Properties),
		})
	}
	for _, item := range req.Edges {
		pbReq.Edges = append(pbReq.Edges, &pb.BatchUpsertGraphEdge{
			EdgeId:         item.EdgeID,
			FromEntityId:   item.FromEntityID,
			ToEntityId:     item.ToEntityID,
			RelationType:   item.RelationType,
			PropertiesJson: mustJSON(item.Properties),
			Confidence:     item.Confidence,
			VersionId:      item.VersionID,
		})
	}

	reply, err := s.BatchUpsertGraph(context.WithValue(r.Context(), middleware.AppContextKey, appCtx), pbReq)
	status := http.StatusOK
	if err != nil {
		status = int(kerrors.Code(err))
		if status <= 0 || status == http.StatusInternalServerError {
			status = http.StatusConflict
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(reply)
}

func inferAppContextForKGBatch(r *http.Request) (middleware.AppContext, error) {
	if r == nil {
		return middleware.AppContext{}, fmt.Errorf("missing app context")
	}
	if ns := strings.TrimSpace(r.Header.Get("X-KG-Namespace")); ns != "" {
		return appContextFromNamespace(ns)
	}
	path := strings.TrimSpace(r.URL.Path)
	trimmed := strings.Trim(path, "/")
	if !strings.HasPrefix(trimmed, "kg/") || !strings.HasSuffix(trimmed, "/graph/batch") {
		return middleware.AppContext{}, fmt.Errorf("missing app context")
	}
	nsPart := strings.TrimPrefix(trimmed, "kg/")
	nsPart = strings.TrimSuffix(nsPart, "/graph/batch")
	nsPart = strings.Trim(nsPart, "/")
	if nsPart == "" {
		return middleware.AppContext{}, fmt.Errorf("missing app context")
	}
	if decoded, err := url.PathUnescape(nsPart); err == nil {
		nsPart = decoded
	}
	return appContextFromNamespace(nsPart)
}

func appContextFromNamespace(namespace string) (middleware.AppContext, error) {
	parts := strings.Split(strings.Trim(namespace, "/"), "/")
	if len(parts) == 3 && parts[0] == "graph" {
		return middleware.AppContext{
			AppID:    strings.TrimSpace(parts[1]),
			TenantID: strings.TrimSpace(parts[2]),
		}, nil
	}
	if len(parts) == 4 && parts[0] == "graph" {
		return middleware.AppContext{
			OrgID:    strings.TrimSpace(parts[1]),
			AppID:    strings.TrimSpace(parts[2]),
			TenantID: strings.TrimSpace(parts[3]),
		}, nil
	}
	return middleware.AppContext{}, fmt.Errorf("missing app context")
}

func (s *GraphService) HybridSearch(ctx context.Context, req *pb.HybridSearchRequest) (*pb.HybridSearchReply, error) {
	if s.searchUC == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "search engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	started := time.Now()

	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	stdlog.Printf("[KGS][GraphService] HybridSearch start app_id=%s tenant_id=%s namespace=%s query=%q top_k=%d alpha=%.2f beta=%.2f",
		appCtx.AppID, appCtx.TenantID, namespace, req.Query, req.TopK, req.Alpha, req.Beta)
	results, err := s.searchUC.HybridSearch(ctx, namespace, req.Query, search.Options{
		TopK:            int(req.TopK),
		Alpha:           req.Alpha,
		Beta:            req.Beta,
		EntityTypes:     req.EntityTypes,
		Domains:         req.Domains,
		MinConfidence:   req.MinConfidence,
		ProvenanceTypes: req.ProvenanceTypes,
	})
	if err != nil {
		stdlog.Printf("[KGS][GraphService] HybridSearch failed app_id=%s tenant_id=%s namespace=%s query=%q err=%v",
			appCtx.AppID, appCtx.TenantID, namespace, req.Query, err)
		return nil, err
	}
	if s.entityReader != nil && len(results) > 0 {
		entityMaps := make([]map[string]any, 0, len(results))
		for _, item := range results {
			props := cloneAnyMap(item.Properties)
			props["id"] = item.ID
			props["label"] = item.Label
			entityMaps = append(entityMaps, props)
		}
		fresh, err := s.entityReader.EnrichWithFreshVersions(ctx, appCtx.AppID, appCtx.TenantID, entityMaps)
		if err != nil {
			return nil, err
		}
		freshByID := make(map[string]map[string]any, len(fresh))
		for _, item := range fresh {
			id := mapString(item, "id")
			if id != "" {
				freshByID[id] = item
			}
		}
		for i := range results {
			freshItem, ok := freshByID[results[i].ID]
			if !ok {
				continue
			}
			results[i].Properties = freshItem
			if label := strings.TrimSpace(mapString(freshItem, "label")); label != "" {
				results[i].Label = label
			}
		}
	}

	reply := &pb.HybridSearchReply{
		Results: make([]*pb.HybridSearchResult, 0, len(results)),
	}
	for _, item := range results {
		reply.Results = append(reply.Results, &pb.HybridSearchResult{
			NodeId:         item.ID,
			Label:          item.Label,
			PropertiesJson: mustJSON(item.Properties),
			Score:          item.Score,
			SemanticScore:  item.SemanticScore,
			TextScore:      item.TextScore,
			Centrality:     item.Centrality,
		})
	}
	stdlog.Printf("[KGS][GraphService] HybridSearch done app_id=%s tenant_id=%s namespace=%s query=%q result_count=%d labels=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, namespace, req.Query, len(reply.Results), summarizeSearchLabels(reply.Results), time.Since(started))
	return reply, nil
}

func (s *GraphService) CreateOverlay(ctx context.Context, req *pb.CreateOverlayRequest) (*pb.CreateOverlayReply, error) {
	if s.overlay == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "overlay manager is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	item, err := s.overlay.Create(ctx, namespace, req.SessionId, req.BaseVersion)
	if err != nil {
		return nil, err
	}
	ttl := item.ExpiresAt.Sub(item.CreatedAt).String()
	return &pb.CreateOverlayReply{
		OverlayId:     item.OverlayID,
		Status:        string(item.Status),
		BaseVersionId: item.BaseVersionID,
		Ttl:           ttl,
	}, nil
}

func (s *GraphService) CommitOverlay(ctx context.Context, req *pb.CommitOverlayRequest) (*pb.CommitOverlayReply, error) {
	if s.overlay == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "overlay manager is not configured")
	}
	result, err := s.overlay.Commit(ctx, req.OverlayId, req.ConflictPolicy)
	if err != nil {
		if errors.Is(err, overlay.ErrOverlayConflict) ||
			errors.Is(err, data.ErrAlreadyExists) ||
			errors.Is(err, data.ErrVersionConflict) ||
			errors.Is(err, data.ErrNameConflict) {
			return nil, kerrors.Conflict("ERR_OVERLAY_COMMIT_CONFLICT", err.Error())
		}
		return nil, err
	}
	return &pb.CommitOverlayReply{
		NewVersionId:      result.NewVersionID,
		EntitiesCommitted: int32(result.EntitiesCommitted),
		EdgesCommitted:    int32(result.EdgesCommitted),
		ConflictsResolved: int32(result.ConflictsResolved),
	}, nil
}

func (s *GraphService) DiscardOverlay(ctx context.Context, req *pb.DiscardOverlayRequest) (*pb.DiscardOverlayReply, error) {
	if s.overlay == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "overlay manager is not configured")
	}
	if err := s.overlay.Discard(ctx, req.OverlayId); err != nil {
		return nil, err
	}
	return &pb.DiscardOverlayReply{
		OverlayId: req.OverlayId,
		Status:    string(overlay.StatusDiscarded),
	}, nil
}

func (s *GraphService) DeleteNodeFromOverlay(ctx context.Context, req *pb.DeleteNodeFromOverlayRequest) (*pb.DeleteNodeFromOverlayReply, error) {
	if s.overlay == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "overlay manager is not configured")
	}
	if req == nil || strings.TrimSpace(req.GetOverlayId()) == "" {
		return nil, kerrors.BadRequest("ERR_MISSING_OVERLAY_ID", "overlay_id is required")
	}
	if strings.TrimSpace(req.GetNodeId()) == "" {
		return nil, kerrors.BadRequest("ERR_MISSING_NODE_ID", "node_id is required")
	}
	if err := s.overlay.DeleteEntityDelta(ctx, req.GetOverlayId(), req.GetNodeId()); err != nil {
		return nil, err
	}
	return &pb.DeleteNodeFromOverlayReply{
		OverlayId: req.GetOverlayId(),
		NodeId:    req.GetNodeId(),
	}, nil
}

func (s *GraphService) DeleteEdgeFromOverlay(ctx context.Context, req *pb.DeleteEdgeFromOverlayRequest) (*pb.DeleteEdgeFromOverlayReply, error) {
	if s.overlay == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "overlay manager is not configured")
	}
	if req == nil || strings.TrimSpace(req.GetOverlayId()) == "" {
		return nil, kerrors.BadRequest("ERR_MISSING_OVERLAY_ID", "overlay_id is required")
	}
	if strings.TrimSpace(req.GetEdgeId()) == "" {
		return nil, kerrors.BadRequest("ERR_MISSING_EDGE_ID", "edge_id is required")
	}
	if err := s.overlay.DeleteEdgeDelta(ctx, req.GetOverlayId(), req.GetEdgeId()); err != nil {
		return nil, err
	}
	return &pb.DeleteEdgeFromOverlayReply{
		OverlayId: req.GetOverlayId(),
		EdgeId:    req.GetEdgeId(),
	}, nil
}

func (s *GraphService) ListVersions(ctx context.Context, req *pb.ListVersionsRequest) (*pb.ListVersionsReply, error) {
	_ = req
	if s.version == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "version manager is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	items, err := s.version.ListVersions(ctx, namespace)
	if err != nil {
		return nil, err
	}
	reply := &pb.ListVersionsReply{
		Versions: make([]*pb.VersionInfo, 0, len(items)),
	}
	for _, item := range items {
		reply.Versions = append(reply.Versions, &pb.VersionInfo{
			VersionId:     item.ID,
			ParentId:      item.ParentID,
			CommitMessage: item.CommitMessage,
			CreatedAtUnix: item.CreatedAt.Unix(),
		})
	}
	return reply, nil
}

func (s *GraphService) DiffVersions(ctx context.Context, req *pb.DiffVersionsRequest) (*pb.DiffVersionsReply, error) {
	if s.version == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "version manager is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	diff, err := s.version.DiffVersions(ctx, namespace, req.FromVersionId, req.ToVersionId)
	if err != nil {
		return nil, err
	}
	return &pb.DiffVersionsReply{
		FromVersionId:    diff.FromVersionID,
		ToVersionId:      diff.ToVersionID,
		EntitiesAdded:    int32(diff.EntitiesAdded),
		EntitiesModified: int32(diff.EntitiesModified),
		EntitiesDeleted:  int32(diff.EntitiesDeleted),
		EdgesAdded:       int32(diff.EdgesAdded),
		EdgesModified:    int32(diff.EdgesModified),
		EdgesDeleted:     int32(diff.EdgesDeleted),
	}, nil
}

func (s *GraphService) RollbackVersion(ctx context.Context, req *pb.RollbackVersionRequest) (*pb.RollbackVersionReply, error) {
	if s.version == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "version manager is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	versionID, err := s.version.Rollback(ctx, namespace, req.VersionId, req.Reason)
	if err != nil {
		return nil, err
	}
	return &pb.RollbackVersionReply{RollbackVersionId: versionID}, nil
}

func (s *GraphService) GetCoverageReport(ctx context.Context, req *pb.GetCoverageReportRequest) (*pb.GetCoverageReportReply, error) {
	if s.analytics == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "analytics engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	report, err := s.analytics.CoverageReport(ctx, namespace, req.Domain)
	if err != nil {
		return nil, err
	}
	reply := &pb.GetCoverageReportReply{
		Domain:          report.Domain,
		TotalEntities:   int32(report.TotalEntities),
		CoveredEntities: int32(report.CoveredEntities),
		CoveragePercent: report.CoveragePercent,
		UncoveredTypes:  report.UncoveredTypes,
		GeneratedAtUnix: report.GeneratedAt.Unix(),
		ByType:          make([]*pb.CoverageByType, 0, len(report.ByType)),
	}
	for _, item := range report.ByType {
		reply.ByType = append(reply.ByType, &pb.CoverageByType{
			EntityType:      item.EntityType,
			TotalEntities:   int32(item.TotalEntities),
			CoveredEntities: int32(item.CoveredEntities),
			CoveragePercent: item.CoveragePercent,
		})
	}
	return reply, nil
}

func (s *GraphService) GetTraceabilityMatrix(ctx context.Context, req *pb.GetTraceabilityMatrixRequest) (*pb.GetTraceabilityMatrixReply, error) {
	if s.analytics == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "analytics engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	report, err := s.analytics.TraceabilityMatrix(ctx, namespace, req.SourceTypes, req.TargetTypes, int(req.MaxHops))
	if err != nil {
		return nil, err
	}

	reply := &pb.GetTraceabilityMatrixReply{
		Matrix:            make([]*pb.TraceabilitySourceRow, 0, len(report.Matrix)),
		TotalSources:      int32(report.TotalSources),
		TotalTargets:      int32(report.TotalTargets),
		ComputeDurationMs: report.ComputeDurationMs,
	}
	for _, row := range report.Matrix {
		item := &pb.TraceabilitySourceRow{
			EntityId: row.SourceID,
			Name:     row.SourceName,
			Type:     row.SourceType,
			Targets:  make([]*pb.TraceabilityTarget, 0, len(row.Targets)),
		}
		for _, target := range row.Targets {
			item.Targets = append(item.Targets, &pb.TraceabilityTarget{
				EntityId: target.EntityID,
				Name:     target.Name,
				Type:     target.Type,
				Hops:     int32(target.Hops),
				Path:     target.Path,
			})
		}
		reply.Matrix = append(reply.Matrix, item)
	}
	return reply, nil
}

func (s *GraphService) CreateViewDefinition(ctx context.Context, req *pb.CreateViewDefinitionRequest) (*pb.CreateViewDefinitionReply, error) {
	if s.projection == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "projection engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	view, err := s.projection.CreateViewDefinition(ctx, namespace, projection.ViewDefinition{
		RoleName:           req.RoleName,
		AllowedEntityTypes: append([]string(nil), req.AllowedEntityTypes...),
		AllowedFields:      append([]string(nil), req.AllowedFields...),
		PIIMaskFields:      append([]string(nil), req.PiiMaskFields...),
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateViewDefinitionReply{View: toPBViewDefinition(view)}, nil
}

func (s *GraphService) GetViewDefinition(ctx context.Context, req *pb.GetViewDefinitionRequest) (*pb.GetViewDefinitionReply, error) {
	if s.projection == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "projection engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	view, err := s.projection.GetViewDefinition(ctx, namespace, req.ViewId)
	if err != nil {
		return nil, err
	}
	return &pb.GetViewDefinitionReply{View: toPBViewDefinition(view)}, nil
}

func (s *GraphService) ListViewDefinitions(ctx context.Context, req *pb.ListViewDefinitionsRequest) (*pb.ListViewDefinitionsReply, error) {
	_ = req
	if s.projection == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "projection engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	items, err := s.projection.ListViewDefinitions(ctx, namespace)
	if err != nil {
		return nil, err
	}
	reply := &pb.ListViewDefinitionsReply{
		Views: make([]*pb.ViewDefinition, 0, len(items)),
	}
	for i := range items {
		view := items[i]
		reply.Views = append(reply.Views, toPBViewDefinition(&view))
	}
	return reply, nil
}

func (s *GraphService) DeleteViewDefinition(ctx context.Context, req *pb.DeleteViewDefinitionRequest) (*pb.DeleteViewDefinitionReply, error) {
	if s.projection == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "projection engine is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	namespace := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID)
	if err := s.projection.DeleteViewDefinition(ctx, namespace, req.ViewId); err != nil {
		return nil, err
	}
	return &pb.DeleteViewDefinitionReply{ViewId: req.ViewId}, nil
}

func (s *GraphService) applyProjectionToSingleNode(ctx context.Context, appCtx middleware.AppContext, raw map[string]any) (map[string]any, error) {
	if s.viewResolver == nil {
		return raw, nil
	}
	role := projectionRole(ctx, appCtx)
	if role == "" {
		return raw, nil
	}
	label := mapString(raw, "label")
	id := mapString(raw, "id")
	nodeRaw := map[string]any{
		"id":         id,
		"label":      label,
		"properties": raw,
	}
	projected, err := s.viewResolver.Resolve(ctx, biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID), role, map[string]any{
		"nodes": []map[string]any{nodeRaw},
		"edges": []map[string]any{},
	})
	if err != nil {
		return nil, err
	}
	nodes := projectionNodeMaps(projected["nodes"])
	if len(nodes) == 0 {
		return map[string]any{"id": id, "label": label}, nil
	}
	node := nodes[0]
	props := projectionMap(node["properties"])
	props["id"] = mapString(props, "id")
	if props["id"] == "" {
		props["id"] = node["id"]
	}
	props["label"] = mapString(node, "label")
	return props, nil
}

func (s *GraphService) applyProjectionToGraphReply(ctx context.Context, appCtx middleware.AppContext, reply *pb.GraphReply) (*pb.GraphReply, error) {
	if s.viewResolver == nil || reply == nil {
		return reply, nil
	}
	role := projectionRole(ctx, appCtx)
	if role == "" {
		return reply, nil
	}

	nodes := make([]map[string]any, 0, len(reply.Nodes))
	for _, node := range reply.Nodes {
		properties, err := parseJSON(node.PropertiesJson)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, map[string]any{
			"id":         node.Id,
			"label":      node.Label,
			"properties": properties,
		})
	}
	edges := make([]map[string]any, 0, len(reply.Edges))
	for _, edge := range reply.Edges {
		properties, err := parseJSON(edge.PropertiesJson)
		if err != nil {
			return nil, err
		}
		edges = append(edges, map[string]any{
			"id":         edge.Id,
			"source":     edge.Source,
			"target":     edge.Target,
			"type":       edge.Type,
			"properties": properties,
		})
	}

	projected, err := s.viewResolver.Resolve(ctx, biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID), role, map[string]any{
		"nodes": nodes,
		"edges": edges,
	})
	if err != nil {
		return nil, err
	}

	out := &pb.GraphReply{
		Nodes: make([]*pb.GraphNode, 0),
		Edges: make([]*pb.GraphEdge, 0),
	}
	for _, node := range projectionNodeMaps(projected["nodes"]) {
		out.Nodes = append(out.Nodes, &pb.GraphNode{
			Id:             projectionString(node, "id"),
			Label:          projectionString(node, "label"),
			PropertiesJson: mustJSON(projectionMap(node["properties"])),
		})
	}
	for _, edge := range projectionNodeMaps(projected["edges"]) {
		out.Edges = append(out.Edges, &pb.GraphEdge{
			Id:             projectionString(edge, "id"),
			Source:         projectionString(edge, "source"),
			Target:         projectionString(edge, "target"),
			Type:           projectionString(edge, "type"),
			PropertiesJson: mustJSON(projectionMap(edge["properties"])),
		})
	}
	return out, nil
}

func (s *GraphService) enrichGraphReply(ctx context.Context, appCtx middleware.AppContext, reply *pb.GraphReply) (*pb.GraphReply, error) {
	if s.entityReader == nil || reply == nil || len(reply.Nodes) == 0 {
		return reply, nil
	}

	nodeMaps := make([]map[string]any, 0, len(reply.Nodes))
	for _, node := range reply.Nodes {
		props, err := parseJSON(node.GetPropertiesJson())
		if err != nil {
			return nil, err
		}
		if node.GetId() != "" {
			props["id"] = node.GetId()
		}
		if node.GetLabel() != "" {
			props["label"] = node.GetLabel()
		}
		nodeMaps = append(nodeMaps, props)
	}

	fresh, err := s.entityReader.EnrichWithFreshVersions(ctx, appCtx.AppID, appCtx.TenantID, nodeMaps)
	if err != nil {
		return nil, err
	}
	freshByID := make(map[string]map[string]any, len(fresh))
	for _, item := range fresh {
		id := mapString(item, "id")
		if id != "" {
			freshByID[id] = item
		}
	}

	for i := range reply.Nodes {
		id := reply.Nodes[i].GetId()
		freshItem, ok := freshByID[id]
		if !ok {
			continue
		}
		reply.Nodes[i].PropertiesJson = mustJSON(freshItem)
		reply.Nodes[i].Properties = stringifyMap(freshItem)
		if label := strings.TrimSpace(mapString(freshItem, "label")); label != "" {
			reply.Nodes[i].Label = label
		}
	}
	return reply, nil
}

func (s *GraphService) ensureReplyContainsNodes(ctx context.Context, appCtx middleware.AppContext, reply *pb.GraphReply, requiredIDs []string) (*pb.GraphReply, error) {
	if len(requiredIDs) == 0 {
		return reply, nil
	}
	if reply == nil {
		reply = &pb.GraphReply{
			Nodes: make([]*pb.GraphNode, 0),
			Edges: make([]*pb.GraphEdge, 0),
		}
	}
	existing := make(map[string]struct{}, len(reply.GetNodes()))
	for _, node := range reply.GetNodes() {
		id := strings.TrimSpace(node.GetId())
		if id != "" {
			existing[id] = struct{}{}
		}
	}
	for _, requestedID := range requiredIDs {
		nodeID := strings.TrimSpace(requestedID)
		if nodeID == "" {
			continue
		}
		if _, ok := existing[nodeID]; ok {
			continue
		}
		node, _ := s.fetchNodeForReply(ctx, appCtx, nodeID)
		if node == nil {
			continue
		}
		id := strings.TrimSpace(mapString(node, "id"))
		if id == "" {
			id = nodeID
			node["id"] = id
		}
		label := strings.TrimSpace(mapString(node, "label"))
		if label == "" {
			label = strings.TrimSpace(mapString(node, "entity_type"))
		}
		if label != "" {
			node["label"] = label
		}
		reply.Nodes = append(reply.Nodes, &pb.GraphNode{
			Id:             id,
			Label:          label,
			PropertiesJson: mustJSON(node),
			Properties:     stringifyMap(node),
		})
		existing[id] = struct{}{}
	}
	return reply, nil
}

func (s *GraphService) fetchNodeForReply(ctx context.Context, appCtx middleware.AppContext, nodeID string) (map[string]any, error) {
	if s.entityReader != nil {
		if out, err := s.entityReader.GetEntity(ctx, appCtx.AppID, appCtx.TenantID, nodeID); err == nil && out != nil {
			return out, nil
		}
	}
	if s.uc != nil {
		if out, err := s.uc.GetNode(ctx, appCtx.AppID, appCtx.TenantID, nodeID); err == nil && out != nil {
			return out, nil
		}
	}
	return nil, nil
}

func projectionRole(ctx context.Context, appCtx middleware.AppContext) string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		role := strings.TrimSpace(tr.RequestHeader().Get("X-KG-Role"))
		if role != "" {
			return role
		}
	}
	parts := strings.Split(appCtx.Scopes, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func projectionNodeMaps(raw any) []map[string]any {
	if raw == nil {
		return []map[string]any{}
	}
	if nodes, ok := raw.([]map[string]any); ok {
		return nodes
	}
	items, ok := raw.([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if node, ok := item.(map[string]any); ok {
			out = append(out, node)
		}
	}
	return out
}

func projectionMap(raw any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	if out, ok := raw.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func projectionString(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	return fmt.Sprint(raw[key])
}

func toPBViewDefinition(view *projection.ViewDefinition) *pb.ViewDefinition {
	if view == nil {
		return nil
	}
	return &pb.ViewDefinition{
		ViewId:             view.ID,
		RoleName:           view.RoleName,
		AllowedEntityTypes: append([]string(nil), view.AllowedEntityTypes...),
		AllowedFields:      append([]string(nil), view.AllowedFields...),
		PiiMaskFields:      append([]string(nil), view.PIIMaskFields...),
		CreatedAtUnix:      view.CreatedAt.Unix(),
	}
}

func getAppContext(ctx context.Context) (middleware.AppContext, error) {
	appCtx, ok := middleware.AppContextFromContext(ctx)
	if !ok || appCtx.AppID == "" {
		return middleware.AppContext{}, fmt.Errorf("missing app context")
	}
	if appCtx.TenantID == "" {
		appCtx.TenantID = "default"
	}
	return appCtx, nil
}

func parseJSON(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid properties_json: %w", err)
	}
	return out, nil
}

func mapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func mustJSON(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func stringifyMap(m map[string]any) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func primaryNodeLabel(labels []string) string {
	for _, label := range labels {
		trimmed := strings.TrimSpace(label)
		if trimmed == "" || trimmed == "Entity" {
			continue
		}
		return trimmed
	}
	if len(labels) == 0 {
		return ""
	}
	return strings.TrimSpace(labels[0])
}

func toGraphReply(result map[string]any) *pb.GraphReply {
	reply := &pb.GraphReply{
		Nodes: []*pb.GraphNode{},
		Edges: []*pb.GraphEdge{},
	}

	nodeByID := make(map[string]*pb.GraphNode)
	edgeByID := make(map[string]*pb.GraphEdge)
	internalNodeIDToID := make(map[int64]string)

	rows, _ := result["data"].([]map[string]any)
	if rows == nil {
		if genericRows, ok := result["data"].([]any); ok {
			for _, row := range genericRows {
				if m, ok := row.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		}
	}

	for _, row := range rows {
		collectNode(nodeByID, internalNodeIDToID, row["n"])
		collectNode(nodeByID, internalNodeIDToID, row["m"])
		collectEdge(edgeByID, internalNodeIDToID, row["r"])
		collectNodesFromPath(nodeByID, internalNodeIDToID, row["nodes"])
		collectEdgesFromPath(edgeByID, internalNodeIDToID, row["rels"])
	}

	for _, n := range nodeByID {
		reply.Nodes = append(reply.Nodes, n)
	}
	for _, e := range edgeByID {
		reply.Edges = append(reply.Edges, e)
	}
	return reply
}

func applyPagination(ctx context.Context, reply *pb.GraphReply, pageSize int32, pageToken string) (*pb.GraphReply, error) {
	if pageSize <= 0 {
		return reply, nil
	}
	offset, err := biz.DecodePageToken(pageToken)
	if err != nil {
		return nil, fmt.Errorf("invalid page token: %w", err)
	}
	if offset < 0 {
		offset = 0
	}

	if offset >= len(reply.Nodes) {
		if tr, ok := transport.FromServerContext(ctx); ok {
			tr.ReplyHeader().Set("X-Next-Page-Token", "")
		}
		return &pb.GraphReply{Nodes: []*pb.GraphNode{}, Edges: []*pb.GraphEdge{}}, nil
	}

	end := offset + int(pageSize)
	if end > len(reply.Nodes) {
		end = len(reply.Nodes)
	}
	pagedNodes := reply.Nodes[offset:end]

	allowed := make(map[string]struct{}, len(pagedNodes))
	for _, n := range pagedNodes {
		allowed[n.Id] = struct{}{}
	}
	pagedEdges := make([]*pb.GraphEdge, 0)
	for _, e := range reply.Edges {
		_, okSource := allowed[e.Source]
		_, okTarget := allowed[e.Target]
		if okSource || okTarget {
			pagedEdges = append(pagedEdges, e)
		}
	}

	next := ""
	if end < len(reply.Nodes) {
		next = biz.EncodePageToken(end)
	}
	if tr, ok := transport.FromServerContext(ctx); ok {
		tr.ReplyHeader().Set("X-Next-Page-Token", next)
	}
	return &pb.GraphReply{
		Nodes: pagedNodes,
		Edges: pagedEdges,
	}, nil
}

func summarizeBatchLabels(entities []*pb.BatchEntity) string {
	if len(entities) == 0 {
		return "{}"
	}
	counts := make(map[string]int)
	for _, entity := range entities {
		label := strings.TrimSpace(entity.GetLabel())
		if label == "" {
			label = "<empty>"
		}
		counts[label]++
	}
	return formatSummaryCounts(counts)
}

func summarizeSearchLabels(results []*pb.HybridSearchResult) string {
	if len(results) == 0 {
		return "{}"
	}
	counts := make(map[string]int)
	for _, result := range results {
		label := strings.TrimSpace(result.GetLabel())
		if label == "" {
			label = "<empty>"
		}
		counts[label]++
	}
	return formatSummaryCounts(counts)
}

func formatSummaryCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func collectNodesFromPath(nodeByID map[string]*pb.GraphNode, internalNodeIDToID map[int64]string, raw any) {
	items, ok := raw.([]any)
	if !ok {
		return
	}
	for _, item := range items {
		collectNode(nodeByID, internalNodeIDToID, item)
	}
}

func collectEdgesFromPath(edgeByID map[string]*pb.GraphEdge, internalNodeIDToID map[int64]string, raw any) {
	items, ok := raw.([]any)
	if !ok {
		return
	}
	for _, item := range items {
		collectEdge(edgeByID, internalNodeIDToID, item)
	}
}

func collectNode(nodeByID map[string]*pb.GraphNode, internalNodeIDToID map[int64]string, raw any) {
	node, ok := raw.(neo4j.Node)
	if !ok {
		return
	}
	id := fmt.Sprint(node.Props["id"])
	if id == "" || id == "<nil>" {
		id = fmt.Sprint(node.Id)
	}
	internalNodeIDToID[node.Id] = id
	if _, exists := nodeByID[id]; exists {
		return
	}
	label := ""
	if len(node.Labels) > 0 {
		label = node.Labels[0]
	}
	nodeByID[id] = &pb.GraphNode{
		Id:             id,
		Label:          label,
		PropertiesJson: mustJSON(node.Props),
	}
}

func collectEdge(edgeByID map[string]*pb.GraphEdge, internalNodeIDToID map[int64]string, raw any) {
	rel, ok := raw.(neo4j.Relationship)
	if !ok {
		return
	}
	id := fmt.Sprint(rel.Props["id"])
	if id == "" || id == "<nil>" {
		id = fmt.Sprint(rel.Id)
	}
	if _, exists := edgeByID[id]; exists {
		return
	}
	source := internalNodeIDToID[rel.StartId]
	target := internalNodeIDToID[rel.EndId]
	edgeByID[id] = &pb.GraphEdge{
		Id:             id,
		Source:         source,
		Target:         target,
		Type:           rel.Type,
		PropertiesJson: mustJSON(rel.Props),
	}
}
