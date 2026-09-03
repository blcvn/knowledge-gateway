package data

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// QueryNodesFilter holds filters for the QueryNodes method.
type QueryNodesFilter struct {
	Labels         []string          // filter by entity_type(s)
	PropertyEq     map[string]string // exact match: JSONB @> containment
	PropertyExists []string          // property key exists in JSONB
	OrderBy        string            // e.g. "created_at DESC" or "name ASC"
	Limit          int
	Offset         int
}

// QueryNodes returns entities matching a flexible filter with JSONB property containment.
func (r *EntityReader) QueryNodes(
	ctx context.Context,
	appID, tenantID string,
	filter QueryNodesFilter,
) ([]map[string]any, int64, error) {
	started := time.Now()
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("entity reader is not configured")
	}

	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	stdlog.Printf("[KGS][EntityReader] QueryNodes start app_id=%s tenant_id=%s labels=%v property_eq_keys=%d property_exists=%v limit=%d offset=%d",
		appID, tenantID, filter.Labels, len(filter.PropertyEq), filter.PropertyExists, filter.Limit, filter.Offset)

	base := r.db.WithContext(ctx).
		Model(&KGEntity{}).
		Where("app_id = ? AND tenant_id = ?", appID, tenantID).
		Where("is_deleted = FALSE")

	// Filter by label(s)
	if len(filter.Labels) > 0 {
		base = base.Where("entity_type IN ?", filter.Labels)
	}

	// JSONB containment filter: properties @> '{"key": "value"}'
	if len(filter.PropertyEq) > 0 {
		containment, err := json.Marshal(filter.PropertyEq)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid property_eq filter: %w", err)
		}
		base = base.Where("properties @> ?::jsonb", string(containment))
	}

	// Property exists filter: properties ? 'key'
	for _, key := range filter.PropertyExists {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		base = base.Where("properties ? ?", gorm.Expr("?", key))
	}

	// Count total matches
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] QueryNodes failed stage=count app_id=%s tenant_id=%s err=%v", appID, tenantID, err)
		return nil, 0, err
	}

	// Order by
	orderClause := "created_at DESC"
	if filter.OrderBy != "" {
		// Whitelist allowed order fields
		sanitized := sanitizeOrderBy(filter.OrderBy)
		if sanitized != "" {
			orderClause = sanitized
		}
	}

	// Fetch rows
	rows := make([]KGEntity, 0, filter.Limit)
	if err := base.Session(&gorm.Session{}).
		Order(orderClause).
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&rows).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] QueryNodes failed stage=find app_id=%s tenant_id=%s err=%v", appID, tenantID, err)
		return nil, 0, err
	}

	entities := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entities = append(entities, kgEntityToAPIMap(row))
	}

	stdlog.Printf("[KGS][EntityReader] QueryNodes done app_id=%s tenant_id=%s returned=%d total=%d duration=%s",
		appID, tenantID, len(entities), total, time.Since(started))
	return entities, total, nil
}

// sanitizeOrderBy validates and sanitizes the ORDER BY clause.
func sanitizeOrderBy(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	allowedFields := map[string]bool{
		"created_at":  true,
		"updated_at":  true,
		"name":        true,
		"entity_type": true,
		"entity_id":   true,
		"confidence":  true,
		"source_file": true,
		"version":     true,
	}

	parts := strings.Fields(raw)
	if len(parts) == 0 || len(parts) > 2 {
		return ""
	}
	field := strings.ToLower(parts[0])
	if !allowedFields[field] {
		return ""
	}
	direction := "DESC"
	if len(parts) == 2 {
		d := strings.ToUpper(parts[1])
		if d == "ASC" || d == "DESC" {
			direction = d
		}
	}
	return field + " " + direction
}
