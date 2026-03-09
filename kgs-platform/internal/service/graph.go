package service

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"sort"
	"strings"
	"time"

	pb "kgs-platform/api/graph/v1"
	"kgs-platform/internal/analytics"
	"kgs-platform/internal/batch"
	"kgs-platform/internal/biz"
	"kgs-platform/internal/overlay"
	"kgs-platform/internal/projection"
	"kgs-platform/internal/search"
	"kgs-platform/internal/server/middleware"
	"kgs-platform/internal/version"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphService struct {
	pb.UnimplementedGraphServer
	uc         GraphUsecase
	batchUC    *batch.Usecase
	searchUC   search.SearchEngine
	overlay    overlay.OverlayManager
	version    version.VersionManager
	analytics  analytics.AnalyticsEngine
	projection projection.ProjectionEngine
}

type GraphUsecase interface {
	CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error)
	GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
	CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error)
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
	projectionEngine projection.ProjectionEngine,
) *GraphService {
	return &GraphService{
		uc:         uc,
		batchUC:    batchUC,
		searchUC:   searchUC,
		overlay:    overlayMgr,
		version:    versionMgr,
		analytics:  analyticsEngine,
		projection: projectionEngine,
	}
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
	out, err := s.uc.GetNode(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId)
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

func (s *GraphService) GetContext(ctx context.Context, req *pb.GetContextRequest) (*pb.GraphReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.uc.GetContext(ctx, appCtx.AppID, appCtx.TenantID, req.NodeId, int(req.Depth), req.Direction)
	if err != nil {
		return nil, err
	}
	projectedReply, err := s.applyProjectionToGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		return nil, err
	}
	reply, err := applyPagination(ctx, projectedReply, req.PageSize, req.PageToken)
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
	projectedReply, err := s.applyProjectionToGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		return nil, err
	}
	reply, err := applyPagination(ctx, projectedReply, req.PageSize, req.PageToken)
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
	projectedReply, err := s.applyProjectionToGraphReply(ctx, appCtx, toGraphReply(result))
	if err != nil {
		return nil, err
	}
	reply, err := applyPagination(ctx, projectedReply, req.PageSize, req.PageToken)
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
	reply, err := s.applyProjectionToGraphReply(ctx, appCtx, toGraphReply(result))
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
	if s.projection == nil {
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
	projected, err := s.projection.Apply(ctx, biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID), role, map[string]any{
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
	if s.projection == nil || reply == nil {
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

	projected, err := s.projection.Apply(ctx, biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID), role, map[string]any{
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
