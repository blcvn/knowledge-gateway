package projection

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// ProviderSet wires projection engine dependencies.
var ProviderSet = wire.NewSet(NewEngine)

type ProjectionEngine interface {
	Apply(ctx context.Context, namespace, role string, rawData map[string]any) (map[string]any, error)
	CreateViewDefinition(ctx context.Context, namespace string, view ViewDefinition) (*ViewDefinition, error)
	GetViewDefinition(ctx context.Context, namespace, viewID string) (*ViewDefinition, error)
	ListViewDefinitions(ctx context.Context, namespace string) ([]ViewDefinition, error)
	DeleteViewDefinition(ctx context.Context, namespace, viewID string) error
}

type Engine struct {
	db  *gorm.DB
	log *log.Helper
}

func NewEngine(db *gorm.DB, logger log.Logger) *Engine {
	return &Engine{db: db, log: log.NewHelper(logger)}
}

func (e *Engine) Apply(ctx context.Context, namespace, role string, rawData map[string]any) (map[string]any, error) {
	if rawData == nil || role == "" || e == nil {
		return rawData, nil
	}
	view, err := e.lookupRoleView(ctx, namespace, role)
	if err != nil {
		return nil, err
	}
	if view == nil {
		return rawData, nil
	}

	nodes := toNodeMaps(rawData["nodes"])
	edges := toEdgeMaps(rawData["edges"])

	allowedTypes := toSet(view.AllowedEntityTypes)
	allowedFields := toSet(view.AllowedFields)
	piiMaskFields := toSet(view.PIIMaskFields)

	projectedNodes := make([]map[string]any, 0, len(nodes))
	allowedNodeIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		label := asString(node["label"])
		if len(allowedTypes) > 0 {
			if _, ok := allowedTypes[label]; !ok {
				continue
			}
		}
		properties := toMap(node["properties"])
		filteredProperties := make(map[string]any, len(properties))
		for key, val := range properties {
			if len(allowedFields) > 0 {
				if _, ok := allowedFields[key]; !ok {
					continue
				}
			}
			if _, ok := piiMaskFields[key]; ok {
				filteredProperties[key] = MaskPIIValue(key, val)
				continue
			}
			filteredProperties[key] = val
		}
		id := asString(node["id"])
		allowedNodeIDs[id] = struct{}{}
		projectedNodes = append(projectedNodes, map[string]any{
			"id":         id,
			"label":      label,
			"properties": filteredProperties,
		})
	}

	projectedEdges := make([]map[string]any, 0, len(edges))
	for _, edge := range edges {
		source := asString(edge["source"])
		target := asString(edge["target"])
		if _, ok := allowedNodeIDs[source]; !ok {
			continue
		}
		if _, ok := allowedNodeIDs[target]; !ok {
			continue
		}
		projectedEdges = append(projectedEdges, map[string]any{
			"id":         asString(edge["id"]),
			"source":     source,
			"target":     target,
			"type":       asString(edge["type"]),
			"properties": toMap(edge["properties"]),
		})
	}

	return map[string]any{
		"nodes": projectedNodes,
		"edges": projectedEdges,
	}, nil
}

func (e *Engine) CreateViewDefinition(ctx context.Context, namespace string, view ViewDefinition) (*ViewDefinition, error) {
	if e == nil || e.db == nil {
		return nil, errors.New("projection engine is not configured")
	}
	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(view.RoleName) == "" {
		return nil, errors.New("role_name is required")
	}
	if view.ID == "" {
		view.ID = uuid.NewString()
	}
	view.AppID = appID
	view.TenantID = tenantID

	record := toRecord(view)
	if err := e.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	out := record.toDomain()
	return &out, nil
}

func (e *Engine) GetViewDefinition(ctx context.Context, namespace, viewID string) (*ViewDefinition, error) {
	if e == nil || e.db == nil {
		return nil, errors.New("projection engine is not configured")
	}
	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}
	var record ViewDefinitionRecord
	err = e.db.WithContext(ctx).
		Where("id = ? AND app_id = ? AND tenant_id = ?", viewID, appID, tenantID).
		Take(&record).Error
	if err != nil {
		return nil, err
	}
	out := record.toDomain()
	return &out, nil
}

func (e *Engine) ListViewDefinitions(ctx context.Context, namespace string) ([]ViewDefinition, error) {
	if e == nil || e.db == nil {
		return nil, errors.New("projection engine is not configured")
	}
	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}
	var records []ViewDefinitionRecord
	if err := e.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ?", appID, tenantID).
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]ViewDefinition, 0, len(records))
	for _, record := range records {
		out = append(out, record.toDomain())
	}
	return out, nil
}

func (e *Engine) DeleteViewDefinition(ctx context.Context, namespace, viewID string) error {
	if e == nil || e.db == nil {
		return errors.New("projection engine is not configured")
	}
	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return err
	}
	return e.db.WithContext(ctx).
		Where("id = ? AND app_id = ? AND tenant_id = ?", viewID, appID, tenantID).
		Delete(&ViewDefinitionRecord{}).Error
}

func (e *Engine) lookupRoleView(ctx context.Context, namespace, role string) (*ViewDefinition, error) {
	if e == nil || e.db == nil {
		return nil, nil
	}
	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}
	var record ViewDefinitionRecord
	err = e.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ? AND role_name = ?", appID, tenantID, role).
		Order("created_at DESC").
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	view := record.toDomain()
	return &view, nil
}

func splitNamespace(namespace string) (appID, tenantID string, err error) {
	parts := strings.Split(strings.TrimSpace(namespace), "/")
	if len(parts) < 3 || parts[0] != "graph" {
		return "", "", fmt.Errorf("invalid namespace %q", namespace)
	}
	if parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("invalid namespace %q", namespace)
	}
	return parts[1], parts[2], nil
}

func toSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func toMap(raw any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	if m, ok := raw.(map[string]any); ok {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return map[string]any{}
}

func toNodeMaps(raw any) []map[string]any {
	if raw == nil {
		return []map[string]any{}
	}
	if items, ok := raw.([]map[string]any); ok {
		return append([]map[string]any(nil), items...)
	}
	arr, ok := raw.([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func toEdgeMaps(raw any) []map[string]any {
	return toNodeMaps(raw)
}

func asString(raw any) string {
	if raw == nil {
		return ""
	}
	return fmt.Sprint(raw)
}

func sortViewDefinitions(items []ViewDefinition) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}
