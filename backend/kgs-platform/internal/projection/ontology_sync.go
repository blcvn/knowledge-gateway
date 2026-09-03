package projection

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type OntologyProjectionSync struct {
	db  *gorm.DB
	log *log.Helper
}

func NewOntologyProjectionSync(db *gorm.DB, logger log.Logger) *OntologyProjectionSync {
	return &OntologyProjectionSync{
		db:  db,
		log: log.NewHelper(logger),
	}
}

func (s *OntologyProjectionSync) SyncRoleView(ctx context.Context, appID, tenantID, roleName string) error {
	if s == nil || s.db == nil {
		return nil
	}
	appID = strings.TrimSpace(appID)
	tenantID = strings.TrimSpace(tenantID)
	roleName = strings.TrimSpace(roleName)
	if appID == "" || tenantID == "" || roleName == "" {
		return nil
	}

	var ontologyTypes []string
	if err := s.db.WithContext(ctx).
		Table("kgs_entity_types").
		Where("app_id = ? AND tenant_id = ?", appID, tenantID).
		Order("name ASC").
		Pluck("name", &ontologyTypes).Error; err != nil {
		return err
	}

	var record ViewDefinitionRecord
	err := s.db.WithContext(ctx).
		Where("app_id = ? AND tenant_id = ? AND role_name = ?", appID, tenantID, roleName).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.log.Infof("projection sync skipped: no view definition app_id=%s tenant_id=%s role=%s", appID, tenantID, roleName)
		return nil
	}
	if err != nil {
		return err
	}

	merged := mergeStringSlices(decodeJSONStringArray(record.AllowedEntityTypesJSON), ontologyTypes)
	record.AllowedEntityTypesJSON = encodeJSONStringArray(merged)
	return s.db.WithContext(ctx).Save(&record).Error
}

func (s *OntologyProjectionSync) SyncAllRoleViews(ctx context.Context, appID, tenantID string) error {
	if s == nil || s.db == nil {
		return nil
	}
	appID = strings.TrimSpace(appID)
	tenantID = strings.TrimSpace(tenantID)
	if appID == "" || tenantID == "" {
		return nil
	}

	var roles []string
	if err := s.db.WithContext(ctx).
		Model(&ViewDefinitionRecord{}).
		Where("app_id = ? AND tenant_id = ?", appID, tenantID).
		Distinct("role_name").
		Pluck("role_name", &roles).Error; err != nil {
		return err
	}
	sort.Strings(roles)

	for _, role := range roles {
		if err := s.SyncRoleView(ctx, appID, tenantID, role); err != nil {
			return err
		}
	}
	return nil
}

func mergeStringSlices(a, b []string) []string {
	merged := make([]string, 0, len(a)+len(b))
	seen := make(map[string]struct{}, len(a)+len(b))

	add := func(values []string) {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, trimmed)
		}
	}

	add(a)
	add(b)
	return merged
}
