package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

type OntologyRepo interface {
	GetEntityType(ctx context.Context, appID, name string) (*EntityType, error)
	GetRelationType(ctx context.Context, appID, name string) (*RelationType, error)
}

type OntologyValidatorConfig struct {
	Enabled             bool
	StrictMode          bool
	SchemaValidation    bool
	EdgeConstraintCheck bool
}

type OntologyValidator struct {
	repo   OntologyRepo
	graph  GraphRepo
	config OntologyValidatorConfig
	log    *log.Helper
}

func NewOntologyValidator(repo OntologyRepo, graph GraphRepo, config OntologyValidatorConfig, logger log.Logger) *OntologyValidator {
	return &OntologyValidator{
		repo:   repo,
		graph:  graph,
		config: config,
		log:    log.NewHelper(logger),
	}
}

func (v *OntologyValidator) ValidateEntity(ctx context.Context, appID, label string, properties map[string]any) error {
	if v == nil || !v.config.Enabled {
		return nil
	}
	if v.repo == nil {
		return nil
	}

	entityType, err := v.repo.GetEntityType(ctx, appID, label)
	if err != nil {
		v.log.Warnf("ontology entity lookup failed app_id=%s label=%s err=%v (bypassing validation)", appID, label, err)
		return nil
	}
	if entityType == nil {
		return v.handleViolation(ctx, "unknown entity type", map[string]string{
			"app_id": appID,
			"label":  label,
		})
	}

	if v.config.SchemaValidation && len(entityType.Schema) > 0 {
		if err := v.validateJSONSchema(json.RawMessage(entityType.Schema), properties); err != nil {
			return v.handleViolation(ctx, "properties validation failed", map[string]string{
				"app_id": appID,
				"label":  label,
				"error":  err.Error(),
			})
		}
	}

	return nil
}

func (v *OntologyValidator) ValidateEdge(ctx context.Context, appID, tenantID, relationType, sourceNodeID, targetNodeID string) error {
	if v == nil || !v.config.Enabled {
		return nil
	}
	if v.repo == nil {
		return nil
	}

	relType, err := v.repo.GetRelationType(ctx, appID, relationType)
	if err != nil {
		v.log.Warnf("ontology relation lookup failed app_id=%s relation=%s err=%v (bypassing validation)", appID, relationType, err)
		return nil
	}
	if relType == nil {
		return v.handleViolation(ctx, "unknown relation type", map[string]string{
			"app_id":        appID,
			"relation_type": relationType,
		})
	}
	if !v.config.EdgeConstraintCheck {
		return nil
	}

	sourceTypes := decodeJSONArray(json.RawMessage(relType.SourceTypes))
	targetTypes := decodeJSONArray(json.RawMessage(relType.TargetTypes))
	if len(sourceTypes) == 0 && len(targetTypes) == 0 {
		return nil
	}
	if v.graph == nil {
		v.log.Warnf("ontology edge validation skipped: graph repo is nil app_id=%s relation=%s", appID, relationType)
		return nil
	}

	if len(sourceTypes) > 0 {
		sourceNode, err := v.graph.GetNode(ctx, appID, tenantID, sourceNodeID)
		if err != nil {
			v.log.Warnf("ontology source node lookup failed app_id=%s node_id=%s err=%v (bypassing validation)", appID, sourceNodeID, err)
			return nil
		}
		sourceLabel := extractLabelFromNode(sourceNode)
		if !containsIgnoreCase(sourceTypes, sourceLabel) {
			return v.handleViolation(ctx, "source type not allowed", map[string]string{
				"relation_type": relationType,
				"source_label":  sourceLabel,
				"allowed":       strings.Join(sourceTypes, ","),
			})
		}
	}

	if len(targetTypes) > 0 {
		targetNode, err := v.graph.GetNode(ctx, appID, tenantID, targetNodeID)
		if err != nil {
			v.log.Warnf("ontology target node lookup failed app_id=%s node_id=%s err=%v (bypassing validation)", appID, targetNodeID, err)
			return nil
		}
		targetLabel := extractLabelFromNode(targetNode)
		if !containsIgnoreCase(targetTypes, targetLabel) {
			return v.handleViolation(ctx, "target type not allowed", map[string]string{
				"relation_type": relationType,
				"target_label":  targetLabel,
				"allowed":       strings.Join(targetTypes, ","),
			})
		}
	}

	return nil
}

func (v *OntologyValidator) handleViolation(ctx context.Context, message string, metadata map[string]string) error {
	_ = ctx
	if v.config.StrictMode {
		return ErrSchemaInvalid(message, metadata)
	}
	v.log.Warnf("[ontology-soft] %s metadata=%v", message, metadata)
	return nil
}

func (v *OntologyValidator) validateJSONSchema(schema json.RawMessage, properties map[string]any) error {
	// Placeholder for Phase 5 (full JSON Schema validation).
	_ = schema
	_ = properties
	return nil
}

func decodeJSONArray(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func containsIgnoreCase(list []string, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return true
		}
	}
	return false
}

func extractLabelFromNode(node map[string]any) string {
	if node == nil {
		return ""
	}

	for _, key := range []string{"label", "entity_type", "type"} {
		if label, ok := node[key].(string); ok {
			label = strings.TrimSpace(label)
			if label != "" {
				return label
			}
		}
	}

	switch labels := node["labels"].(type) {
	case []string:
		for _, label := range labels {
			label = strings.TrimSpace(label)
			if label != "" {
				return label
			}
		}
	case []any:
		for _, item := range labels {
			if label, ok := item.(string); ok {
				label = strings.TrimSpace(label)
				if label != "" {
					return label
				}
			}
		}
	}

	return ""
}
