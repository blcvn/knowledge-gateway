package ontology

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

type SearchProfileResolver interface {
	Resolve(domainID, tenantID, appID string) (ResolvedSearchProfile, error)
}

type DefaultSearchProfileResolver struct {
	store DomainStore
}

func NewDefaultSearchProfileResolver(store DomainStore) *DefaultSearchProfileResolver {
	return &DefaultSearchProfileResolver{store: store}
}

func (r *DefaultSearchProfileResolver) Resolve(domainID, tenantID, appID string) (ResolvedSearchProfile, error) {
	if r == nil || r.store == nil {
		return ResolvedSearchProfile{}, errors.New("search profile resolver is not configured")
	}

	domain, ok := r.store.GetDomain(domainID)
	if !ok {
		return ResolvedSearchProfile{}, ErrNotFound
	}

	profile := domain.SearchProfile
	resolved := ResolvedSearchProfile{
		Domain: domain,
	}
	if profile == nil {
		resolved.SemanticFields = defaultSemanticFields()
		resolved.FTSLanguage = "simple"
		resolved.QueryStrategy = defaultQueryStrategy("default")
		return r.attachStrategy(domainID, resolved, "default")
	}

	resolved.SearchProfile = profile
	resolved.SemanticFields = mergeSemanticFields(profile.SemanticFields, profile.TenantOverrides, profile.AppOverrides, tenantID, appID)
	if len(resolved.SemanticFields) == 0 {
		resolved.SemanticFields = defaultSemanticFields()
	}
	resolved.FTSLanguage = resolvedFTSLanguage(profile.FTSLanguage, profile.TenantOverrides, profile.AppOverrides, tenantID, appID)
	strategyRef := resolvedQueryStrategyRef(profile.QueryStrategyRef, profile.TenantOverrides, profile.AppOverrides, tenantID, appID)
	if strategyRef == "" {
		strategyRef = "default"
	}
	return r.attachStrategy(domainID, resolved, strategyRef)
}

func (r *DefaultSearchProfileResolver) attachStrategy(domainID string, resolved ResolvedSearchProfile, strategyKey string) (ResolvedSearchProfile, error) {
	strategy, ok := r.store.GetQueryStrategy(strategyKey)
	if !ok {
		log.Printf("WARNING: missing query strategy %q for domain %s; falling back to default", strategyKey, domainID)
		strategy, ok = r.store.GetQueryStrategy("default")
		if !ok {
			strategy = defaultQueryStrategy("default")
		}
	}
	resolved.QueryStrategy = strategy
	return resolved, nil
}

func defaultSemanticFields() []IndexedField {
	return []IndexedField{
		{FieldName: "id", Weight: 1.0},
		{FieldName: "node_type", Weight: 1.0},
		{FieldName: "domain_id", Weight: 1.0},
		{FieldName: "external_ref", Weight: 1.0},
		{FieldName: "status_value", Weight: 1.0},
		{FieldName: "*properties", Weight: 1.0},
	}
}

func defaultQueryStrategy(key string) QueryStrategy {
	switch key {
	case "deep_traversal":
		return QueryStrategy{
			Key:      "deep_traversal",
			Version:  1,
			MaxDepth: 10,
			Params: map[string]any{
				"direction":     "out",
				"depth_mode":    "variable",
				"acl_predicate": "start_only",
			},
		}
	default:
		return QueryStrategy{
			Key:      "default",
			Version:  1,
			MaxDepth: 5,
			Params: map[string]any{
				"direction":     "out",
				"depth_mode":    "fixed",
				"acl_predicate": "any_hop",
			},
		}
	}
}

func mergeSemanticFields(base []IndexedField, tenantOverrides map[string]SearchProfileOverride, appOverrides map[string]SearchProfileOverride, tenantID, appID string) []IndexedField {
	fields := append([]IndexedField(nil), base...)
	if len(fields) == 0 {
		fields = defaultSemanticFields()
	}
	if tenantOverride, ok := tenantOverrides[tenantID]; ok && len(tenantOverride.SemanticFields) > 0 {
		fields = append([]IndexedField(nil), tenantOverride.SemanticFields...)
	}
	if appOverride, ok := appOverrides[tenantID+":"+appID]; ok && len(appOverride.SemanticFields) > 0 {
		fields = append([]IndexedField(nil), appOverride.SemanticFields...)
	}
	return fields
}

func resolvedFTSLanguage(base string, tenantOverrides map[string]SearchProfileOverride, appOverrides map[string]SearchProfileOverride, tenantID, appID string) string {
	if appOverride, ok := appOverrides[tenantID+":"+appID]; ok && strings.TrimSpace(appOverride.FTSLanguage) != "" {
		return strings.TrimSpace(appOverride.FTSLanguage)
	}
	if tenantOverride, ok := tenantOverrides[tenantID]; ok && strings.TrimSpace(tenantOverride.FTSLanguage) != "" {
		return strings.TrimSpace(tenantOverride.FTSLanguage)
	}
	if strings.TrimSpace(base) != "" {
		return strings.TrimSpace(base)
	}
	return "simple"
}

func resolvedQueryStrategyRef(base string, tenantOverrides map[string]SearchProfileOverride, appOverrides map[string]SearchProfileOverride, tenantID, appID string) string {
	if appOverride, ok := appOverrides[tenantID+":"+appID]; ok && strings.TrimSpace(appOverride.QueryStrategyRef) != "" {
		return strings.TrimSpace(appOverride.QueryStrategyRef)
	}
	if tenantOverride, ok := tenantOverrides[tenantID]; ok && strings.TrimSpace(tenantOverride.QueryStrategyRef) != "" {
		return strings.TrimSpace(tenantOverride.QueryStrategyRef)
	}
	return strings.TrimSpace(base)
}

func validateSearchProfile(profile SearchProfile) error {
	if strings.TrimSpace(profile.FTSLanguage) == "" {
		return errors.New("fts_language must be a non-empty string")
	}
	if len(profile.SemanticFields) == 0 {
		return errors.New("semantic_fields must be nil (use defaults) or a non-empty list")
	}
	for _, field := range profile.SemanticFields {
		if strings.TrimSpace(field.FieldName) == "" {
			return errors.New("semantic_fields.field_name must not be empty")
		}
		if field.Weight < 0.1 || field.Weight > 10.0 {
			return fmt.Errorf("semantic_fields.weight out of range for %s", field.FieldName)
		}
	}
	return nil
}

func validateQueryStrategy(strategy QueryStrategy) error {
	if strings.TrimSpace(strategy.Key) == "" {
		return errors.New("key is required")
	}
	if strategy.MaxDepth <= 0 {
		return errors.New("max_depth must be positive")
	}
	return nil
}
