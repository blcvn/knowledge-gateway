package service

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/ontology/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OntologyService struct {
	pb.UnimplementedOntologyServer
	db           *gorm.DB
	ontologyRepo *data.OntologyRepo
	projection   *projection.OntologyProjectionSync
}

func NewOntologyService(db *gorm.DB, ontologyRepo *data.OntologyRepo, projectionSync *projection.OntologyProjectionSync) *OntologyService {
	return &OntologyService{
		db:           db,
		ontologyRepo: ontologyRepo,
		projection:   projectionSync,
	}
}

func (s *OntologyService) CreateEntityType(ctx context.Context, req *pb.CreateEntityTypeRequest) (*pb.CreateEntityTypeReply, error) {
	if s.db == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "ontology database is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return &pb.CreateEntityTypeReply{Status: "INVALID"}, nil
	}

	rawSchema := strings.TrimSpace(req.GetSchema())
	if rawSchema == "" {
		rawSchema = "{}"
	}

	entity := biz.EntityType{
		AppID:       appCtx.AppID,
		TenantID:    appCtx.TenantID,
		Name:        name,
		Description: req.GetDescription(),
		Schema:      datatypes.JSON([]byte(rawSchema)),
	}
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "app_id"},
				{Name: "tenant_id"},
				{Name: "name"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"description": entity.Description,
				"schema":      entity.Schema,
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).
		Create(&entity).Error; err != nil {
		return nil, err
	}

	var persisted biz.EntityType
	if err := s.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ? AND name = ?", appCtx.AppID, appCtx.TenantID, name).
		First(&persisted).Error; err != nil {
		return nil, err
	}

	if s.ontologyRepo != nil {
		_ = s.ontologyRepo.InvalidateEntityType(ctx, appCtx.AppID, name)
	}
	if s.projection != nil {
		if err := s.projection.SyncAllRoleViews(ctx, appCtx.AppID, appCtx.TenantID); err != nil {
			return nil, err
		}
	}

	return &pb.CreateEntityTypeReply{
		Id:     uint32(persisted.ID),
		Name:   persisted.Name,
		Status: "UPSERTED",
	}, nil
}

func (s *OntologyService) CreateRelationType(ctx context.Context, req *pb.CreateRelationTypeRequest) (*pb.CreateRelationTypeReply, error) {
	if s.db == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "ontology database is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return &pb.CreateRelationTypeReply{Status: "INVALID"}, nil
	}

	sourceTypes, _ := json.Marshal(req.GetSourceTypes())
	targetTypes, _ := json.Marshal(req.GetTargetTypes())

	rawPropertiesSchema := strings.TrimSpace(req.GetPropertiesSchema())
	if rawPropertiesSchema == "" {
		rawPropertiesSchema = "{}"
	}

	relation := biz.RelationType{
		AppID:       appCtx.AppID,
		TenantID:    appCtx.TenantID,
		Name:        name,
		Description: req.GetDescription(),
		Properties:  datatypes.JSON([]byte(rawPropertiesSchema)),
		SourceTypes: datatypes.JSON(sourceTypes),
		TargetTypes: datatypes.JSON(targetTypes),
	}
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "app_id"},
				{Name: "tenant_id"},
				{Name: "name"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"description":  relation.Description,
				"properties":   relation.Properties,
				"source_types": relation.SourceTypes,
				"target_types": relation.TargetTypes,
				"updated_at":   gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).
		Create(&relation).Error; err != nil {
		return nil, err
	}

	var persisted biz.RelationType
	if err := s.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ? AND name = ?", appCtx.AppID, appCtx.TenantID, name).
		First(&persisted).Error; err != nil {
		return nil, err
	}

	if s.ontologyRepo != nil {
		_ = s.ontologyRepo.InvalidateRelationType(ctx, appCtx.AppID, name)
	}

	return &pb.CreateRelationTypeReply{
		Id:     uint32(persisted.ID),
		Name:   persisted.Name,
		Status: "UPSERTED",
	}, nil
}

func (s *OntologyService) ListEntityTypes(ctx context.Context, req *pb.ListEntityTypesRequest) (*pb.ListEntityTypesReply, error) {
	_ = req
	if s.db == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "ontology database is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}

	var entities []biz.EntityType
	if err := s.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ?", appCtx.AppID, appCtx.TenantID).
		Order("id ASC").
		Find(&entities).Error; err != nil {
		return nil, err
	}

	out := make([]*pb.EntityTypeInfo, 0, len(entities))
	for i := range entities {
		entity := entities[i]
		out = append(out, &pb.EntityTypeInfo{
			Id:     uint32(entity.ID),
			Name:   entity.Name,
			Schema: string(entity.Schema),
		})
	}
	return &pb.ListEntityTypesReply{Entities: out}, nil
}

func (s *OntologyService) ListRelationTypes(ctx context.Context, req *pb.ListRelationTypesRequest) (*pb.ListRelationTypesReply, error) {
	_ = req
	if s.db == nil {
		return nil, kerrors.InternalServer("ERR_NOT_CONFIGURED", "ontology database is not configured")
	}
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}

	var relations []biz.RelationType
	if err := s.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ?", appCtx.AppID, appCtx.TenantID).
		Order("id ASC").
		Find(&relations).Error; err != nil {
		return nil, err
	}

	out := make([]*pb.RelationTypeInfo, 0, len(relations))
	for i := range relations {
		relation := relations[i]
		out = append(out, &pb.RelationTypeInfo{
			Id:               uint32(relation.ID),
			Name:             relation.Name,
			PropertiesSchema: string(relation.Properties),
			SourceTypes:      decodeJSONStringSlice(relation.SourceTypes),
			TargetTypes:      decodeJSONStringSlice(relation.TargetTypes),
		})
	}
	return &pb.ListRelationTypesReply{Relations: out}, nil
}

func decodeJSONStringSlice(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	return out
}
