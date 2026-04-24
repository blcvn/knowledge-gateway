package data

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func (r *EntityReader) ListEntities(
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
) ([]map[string]any, string, bool, int64, error) {
	started := time.Now()
	if r == nil || r.db == nil {
		return nil, "", false, 0, fmt.Errorf("entity reader is not configured")
	}
	if limit <= 0 {
		return nil, "", false, 0, fmt.Errorf("limit must be positive")
	}
	traceID := traceIDFromContext(ctx)
	stdlog.Printf("[KGS][EntityReader] ListEntities start app_id=%s tenant_id=%s limit=%d cursor=%t entity_type=%s source_file=%s domain=%s provenance_type=%s version_id=%s property_key=%s is_deleted=%t trace_id=%s",
		appID, tenantID, limit, strings.TrimSpace(cursorID) != "", strings.TrimSpace(entityType), strings.TrimSpace(sourceFile), strings.TrimSpace(domain), strings.TrimSpace(provenanceType), strings.TrimSpace(versionID), strings.TrimSpace(propertyKey), isDeleted, traceID)

	base := r.db.WithContext(ctx).
		Model(&KGEntity{}).
		Where("app_id = ? AND tenant_id = ?", appID, tenantID).
		Where("is_deleted = ?", isDeleted)
	if entityType = strings.TrimSpace(entityType); entityType != "" {
		base = base.Where("entity_type = ?", entityType)
	}
	if sourceFile = strings.TrimSpace(sourceFile); sourceFile != "" {
		base = base.Where("source_file = ?", sourceFile)
	}
	if domain = strings.TrimSpace(domain); domain != "" {
		base = base.Where("? = ANY(domains)", domain)
	}
	if provenanceType = strings.TrimSpace(provenanceType); provenanceType != "" {
		base = base.Where("provenance_type = ?", provenanceType)
	}
	if versionID = strings.TrimSpace(versionID); versionID != "" {
		base = base.Where("version_id = ?", versionID)
	}
	if propertyKey = strings.TrimSpace(propertyKey); propertyKey != "" {
		base = base.Where("properties ->> ? = ?", propertyKey, propertyValue)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] ListEntities failed stage=count app_id=%s tenant_id=%s trace_id=%s err=%v", appID, tenantID, traceID, err)
		return nil, "", false, 0, err
	}

	rowsQ := base.Session(&gorm.Session{})
	if cursorID = strings.TrimSpace(cursorID); cursorID != "" {
		rowsQ = rowsQ.Where("entity_id > ?", cursorID)
	}

	rows := make([]KGEntity, 0, limit+1)
	if err := rowsQ.Order("entity_id ASC").Limit(limit + 1).Find(&rows).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] ListEntities failed stage=find app_id=%s tenant_id=%s trace_id=%s err=%v", appID, tenantID, traceID, err)
		return nil, "", false, 0, err
	}

	hasMore := false
	nextCursorID := ""
	if len(rows) > limit {
		hasMore = true
		nextCursorID = rows[limit].EntityID
		rows = rows[:limit]
	}

	entities := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entities = append(entities, kgEntityToAPIMap(row))
	}
	stdlog.Printf("[KGS][EntityReader] ListEntities done app_id=%s tenant_id=%s returned=%d total=%d has_more=%t trace_id=%s duration=%s",
		appID, tenantID, len(entities), total, hasMore, traceID, time.Since(started))
	return entities, nextCursorID, hasMore, total, nil
}

func (r *EntityReader) LookupEntities(
	ctx context.Context,
	appID, tenantID string,
	entityType string,
	sourceFile string,
	matchMode string,
	limit int,
	properties map[string]string,
) ([]map[string]any, int64, error) {
	started := time.Now()
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("entity reader is not configured")
	}
	if limit <= 0 {
		return nil, 0, fmt.Errorf("limit must be positive")
	}
	if len(properties) == 0 {
		return nil, 0, fmt.Errorf("properties is required")
	}
	traceID := traceIDFromContext(ctx)
	stdlog.Printf("[KGS][EntityReader] LookupEntities start app_id=%s tenant_id=%s entity_type=%s source_file=%s match_mode=%s limit=%d properties=%d trace_id=%s",
		appID, tenantID, strings.TrimSpace(entityType), strings.TrimSpace(sourceFile), strings.ToUpper(strings.TrimSpace(matchMode)), limit, len(properties), traceID)

	base := r.db.WithContext(ctx).
		Model(&KGEntity{}).
		Where("app_id = ? AND tenant_id = ?", appID, tenantID).
		Where("is_deleted = FALSE")
	if entityType = strings.TrimSpace(entityType); entityType != "" {
		base = base.Where("entity_type = ?", entityType)
	}
	if sourceFile = strings.TrimSpace(sourceFile); sourceFile != "" {
		base = base.Where("source_file = ?", sourceFile)
	}

	matchMode = strings.ToUpper(strings.TrimSpace(matchMode))
	if matchMode == "" {
		matchMode = "ALL"
	}
	if matchMode == "ALL" {
		payload, err := json.Marshal(properties)
		if err != nil {
			return nil, 0, err
		}
		base = base.Where("properties @> ?::jsonb", string(payload))
	} else {
		keys := make([]string, 0, len(properties))
		for key := range properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		clauses := make([]string, 0, len(keys))
		args := make([]any, 0, len(keys)*2)
		for _, key := range keys {
			clauses = append(clauses, "properties ->> ? = ?")
			args = append(args, key, properties[key])
		}
		base = base.Where("("+strings.Join(clauses, " OR ")+")", args...)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] LookupEntities failed stage=count app_id=%s tenant_id=%s trace_id=%s err=%v", appID, tenantID, traceID, err)
		return nil, 0, err
	}

	rows := make([]KGEntity, 0, limit)
	if err := base.Session(&gorm.Session{}).Order("entity_id ASC").Limit(limit).Find(&rows).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] LookupEntities failed stage=find app_id=%s tenant_id=%s trace_id=%s err=%v", appID, tenantID, traceID, err)
		return nil, 0, err
	}

	entities := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entities = append(entities, kgEntityToAPIMap(row))
	}
	stdlog.Printf("[KGS][EntityReader] LookupEntities done app_id=%s tenant_id=%s returned=%d total=%d trace_id=%s duration=%s",
		appID, tenantID, len(entities), total, traceID, time.Since(started))
	return entities, total, nil
}

func (r *EntityReader) ListEdges(
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
) ([]map[string]any, string, bool, int64, error) {
	started := time.Now()
	if r == nil || r.db == nil {
		return nil, "", false, 0, fmt.Errorf("entity reader is not configured")
	}
	if limit <= 0 {
		return nil, "", false, 0, fmt.Errorf("limit must be positive")
	}
	traceID := traceIDFromContext(ctx)
	stdlog.Printf("[KGS][EntityReader] ListEdges start app_id=%s tenant_id=%s limit=%d cursor=%t relation_type=%s source_file=%s from=%s to=%s version_id=%s is_deleted=%t trace_id=%s",
		appID, tenantID, limit, strings.TrimSpace(cursorID) != "", strings.TrimSpace(relationType), strings.TrimSpace(sourceFile), strings.TrimSpace(fromEntityID), strings.TrimSpace(toEntityID), strings.TrimSpace(versionID), isDeleted, traceID)

	base := r.db.WithContext(ctx).
		Table("kg_edges AS e").
		Where("e.app_id = ? AND e.tenant_id = ?", appID, tenantID).
		Where("e.is_deleted = ?", isDeleted)
	if relationType = strings.TrimSpace(relationType); relationType != "" {
		base = base.Where("e.relation_type = ?", relationType)
	}
	if fromEntityID = strings.TrimSpace(fromEntityID); fromEntityID != "" {
		base = base.Where("e.from_entity_id = ?", fromEntityID)
	}
	if toEntityID = strings.TrimSpace(toEntityID); toEntityID != "" {
		base = base.Where("e.to_entity_id = ?", toEntityID)
	}
	if versionID = strings.TrimSpace(versionID); versionID != "" {
		base = base.Where("e.version_id = ?", versionID)
	}
	if sourceFile = strings.TrimSpace(sourceFile); sourceFile != "" {
		base = base.
			Joins("JOIN kg_entities AS fe ON fe.entity_id = e.from_entity_id AND fe.app_id = e.app_id AND fe.tenant_id = e.tenant_id").
			Where("fe.source_file = ?", sourceFile)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] ListEdges failed stage=count app_id=%s tenant_id=%s trace_id=%s err=%v", appID, tenantID, traceID, err)
		return nil, "", false, 0, err
	}

	rowsQ := base.Session(&gorm.Session{}).Select(strings.Join([]string{
		"e.edge_id",
		"e.app_id",
		"e.tenant_id",
		"e.from_entity_id",
		"e.to_entity_id",
		"e.relation_type",
		"e.properties",
		"e.confidence",
		"e.version_id",
		"e.is_deleted",
		"e.created_at",
	}, ","))
	if cursorID = strings.TrimSpace(cursorID); cursorID != "" {
		rowsQ = rowsQ.Where("e.edge_id > ?", cursorID)
	}

	rows := make([]KGEdge, 0, limit+1)
	if err := rowsQ.Order("e.edge_id ASC").Limit(limit + 1).Find(&rows).Error; err != nil {
		stdlog.Printf("[KGS][EntityReader] ListEdges failed stage=find app_id=%s tenant_id=%s trace_id=%s err=%v", appID, tenantID, traceID, err)
		return nil, "", false, 0, err
	}

	hasMore := false
	nextCursorID := ""
	if len(rows) > limit {
		hasMore = true
		nextCursorID = rows[limit].EdgeID
		rows = rows[:limit]
	}

	edges := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		edges = append(edges, kgEdgeToAPIMap(row))
	}
	stdlog.Printf("[KGS][EntityReader] ListEdges done app_id=%s tenant_id=%s returned=%d total=%d has_more=%t trace_id=%s duration=%s",
		appID, tenantID, len(edges), total, hasMore, traceID, time.Since(started))
	return edges, nextCursorID, hasMore, total, nil
}

func kgEntityToAPIMap(entity KGEntity) map[string]any {
	properties := cloneJSONMap(entity.Properties)
	return map[string]any{
		"entityId":       entity.EntityID,
		"appId":          entity.AppID,
		"tenantId":       entity.TenantID,
		"entityType":     entity.EntityType,
		"name":           entity.Name,
		"properties":     properties,
		"confidence":     entity.Confidence,
		"sourceFile":     entity.SourceFile,
		"chunkId":        entity.ChunkID,
		"skillId":        entity.SkillID,
		"versionId":      entity.VersionID,
		"provenanceType": entity.ProvenanceType,
		"domains":        append([]string(nil), entity.Domains...),
		"aliases":        append([]string(nil), entity.Aliases...),
		"version":        entity.Version,
		"isDeleted":      entity.IsDeleted,
		"createdAt":      formatTimeRFC3339(entity.CreatedAt),
		"updatedAt":      formatTimeRFC3339(entity.UpdatedAt),
	}
}

func kgEdgeToAPIMap(edge KGEdge) map[string]any {
	properties := cloneJSONMap(edge.Properties)
	return map[string]any{
		"edgeId":       edge.EdgeID,
		"appId":        edge.AppID,
		"tenantId":     edge.TenantID,
		"fromEntityId": edge.FromEntityID,
		"toEntityId":   edge.ToEntityID,
		"relationType": edge.RelationType,
		"properties":   properties,
		"confidence":   edge.Confidence,
		"versionId":    edge.VersionID,
		"isDeleted":    edge.IsDeleted,
		"createdAt":    formatTimeRFC3339(edge.CreatedAt),
	}
}

func cloneJSONMap(in JSONMap) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func traceIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}
