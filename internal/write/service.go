package write

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/session"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation")
	ErrNotFound   = errors.New("not found")
)

type AccessResolver interface {
	ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error)
}

type OntologyResolver interface {
	GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error)
	GetCurrentVersion(domainID string) (ontology.OntologyVersion, error)
	GetNodeType(domainID, nodeTypeName string) (ontology.NodeTypeSchema, error)
	GetRelType(domainID, relTypeName, fromNodeType, toNodeType string) (ontology.RelTypeSchema, error)
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
	ListCrossDomainRules(domainID string) []ontology.CrossDomainRelRule
	ResolveCrossDomainRules(domainID, fromNodeType string) []ontology.CrossDomainRelRule
	ValidateCrossDomainTarget(rule ontology.CrossDomainRelRule, targetDomainID, targetNodeType string) error
}

type SessionManager interface {
	Within(ctx context.Context, identity session.WriteIdentity, fn func(session.SessionScope) error) (session.SessionScope, error)
}

type AuditLogger interface {
	RecordWriteAudit(actor access.Identity, ownerTenantID, ownerAppID, action, resourceType, resourceID, outcome, reason string, metadata map[string]any)
}

type Service struct {
	store          Repository
	ontology       OntologyResolver
	accessResolver AccessResolver
	sessionManager SessionManager
	auditLogger    AuditLogger
	ingestMu       sync.RWMutex
	ingestJobs     map[string]IngestJobResponse
	now            func() time.Time
}

func NewService(store Repository, ontology OntologyResolver, accessResolver AccessResolver, sessionManager SessionManager, auditLogger AuditLogger) *Service {
	return &Service{
		store:          store,
		ontology:       ontology,
		accessResolver: accessResolver,
		sessionManager: sessionManager,
		auditLogger:    auditLogger,
		ingestJobs:     map[string]IngestJobResponse{},
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) repositoryForScope(scope session.SessionScope) Repository {
	if scope.Tx == nil {
		return s.store
	}
	if txRepo, ok := s.store.(interface {
		WithTx(*sql.Tx) Repository
	}); ok {
		return txRepo.WithTx(scope.Tx)
	}
	return s.store
}

func (s *Service) IngestDocument(actor access.Identity, req IngestDocumentRequest) (IngestJobResponse, error) {
	if !canRunIngest(actor) {
		return IngestJobResponse{}, ErrForbidden
	}
	if err := validateIngestDocumentRequest(req); err != nil {
		return IngestJobResponse{}, errors.Join(ErrValidation, err)
	}
	if _, err := s.ontology.GetVisibleDomain(actor, req.DomainID); err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return IngestJobResponse{}, ErrForbidden
		}
		return IngestJobResponse{}, err
	}

	job := IngestJobResponse{
		JobID:        newID("job"),
		Status:       "completed",
		NodesCreated: 0,
		Errors:       []string{},
	}
	s.ingestMu.Lock()
	s.ingestJobs[job.JobID] = job
	s.ingestMu.Unlock()

	return IngestJobResponse{
		JobID:  job.JobID,
		Status: "queued",
	}, nil
}

func (s *Service) GetIngestJob(actor access.Identity, jobID string) (IngestJobResponse, error) {
	if !canRunIngest(actor) {
		return IngestJobResponse{}, ErrForbidden
	}
	s.ingestMu.RLock()
	job, ok := s.ingestJobs[jobID]
	s.ingestMu.RUnlock()
	if !ok {
		return IngestJobResponse{}, ErrNotFound
	}
	return job, nil
}

func (s *Service) CreateNode(actor access.Identity, req NodeCreateRequest) (NodeCreateResponse, error) {
	return s.CreateNodeWithContext(context.Background(), actor, req)
}

func (s *Service) CreateNodeWithContext(ctx context.Context, actor access.Identity, req NodeCreateRequest) (NodeCreateResponse, error) {
	if err := validateNodeCreateRequest(req); err != nil {
		return NodeCreateResponse{}, errors.Join(ErrValidation, err)
	}

	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return NodeCreateResponse{}, ErrForbidden
		}
		return NodeCreateResponse{}, err
	}
	if err := s.ensureWritePermission(actor, domain, req.DomainID); err != nil {
		return NodeCreateResponse{}, err
	}

	schema, err := s.ontology.GetNodeType(req.DomainID, req.NodeType)
	if err != nil {
		if errors.Is(err, ontology.ErrNotFound) {
			return NodeCreateResponse{}, errors.Join(ErrValidation, fmt.Errorf("unknown node_type: %s", req.NodeType))
		}
		return NodeCreateResponse{}, err
	}
	if err := validateProperties(schema, req.Properties); err != nil {
		return NodeCreateResponse{}, errors.Join(ErrValidation, err)
	}

	version, err := s.ontology.GetCurrentVersion(req.DomainID)
	if err != nil {
		return NodeCreateResponse{}, err
	}
	statusCfg, err := s.ontology.GetStatusFieldConfig(req.DomainID)
	if err != nil {
		return NodeCreateResponse{}, err
	}

	now := s.now()
	statusValue := ""
	if statusCfg != nil && statusCfg.StatusFieldName != "" {
		if value, ok := req.Properties[statusCfg.StatusFieldName]; ok {
			statusValue = fmt.Sprintf("%v", value)
		}
	}
	nodeID := newID("node")
	node := NodeRecord{
		ID:            nodeID,
		NodeType:      req.NodeType,
		DomainID:      req.DomainID,
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    fallback(req.Visibility, "private"),
		Properties:    req.Properties,
		DomainVersion: version.Version,
		ExternalRef:   strings.TrimSpace(req.ExternalRef),
		StatusValue:   statusValue,
		IsDeleted:     false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	bridgeRelationships, err := s.buildBridgeRelationships(req.DomainID, req.NodeType, actor, nodeID, req.Properties)
	if err != nil {
		return NodeCreateResponse{}, err
	}
	var scope session.SessionScope
	scope, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		event := OutboxEvent{
			ID:            newID("evt"),
			AggregateType: "kg_node",
			AggregateID:   nodeID,
			EventType:     "NODE_UPSERTED",
			Payload: map[string]any{
				"node_id":         nodeID,
				"domain_id":       req.DomainID,
				"owner_tenant_id": actor.TenantID,
				"owner_app_id":    actor.AppID,
				"node_type":       req.NodeType,
				"external_ref":    node.ExternalRef,
				"tx_scope":        scope.Statements,
			},
			Status:     "PENDING",
			RetryCount: 0,
			CreatedAt:  now,
		}
		repo := s.repositoryForScope(scope)
		if err := repo.CreateNodeBundle(ctx, node, bridgeRelationships, event); err != nil {
			if errors.Is(err, ErrDuplicateExternalRef) {
				return errors.Join(ErrValidation, errors.New("external_ref already exists"))
			}
			return err
		}
		return nil
	})
	if err != nil {
		return NodeCreateResponse{}, err
	}
	_ = scope
	s.recordWriteAudit(actor, node.OwnerTenantID, node.OwnerAppID, "kg.node.create", "kg_node", node.ID, "allow", "", map[string]any{
		"domain_id":      node.DomainID,
		"node_type":      node.NodeType,
		"external_ref":   node.ExternalRef,
		"bridge_rel_cnt": len(bridgeRelationships),
	})

	return NodeCreateResponse{
		NodeID:        nodeID,
		DomainVersion: version.Version,
		Status:        "processing",
		SyncETAMs:     800,
	}, nil
}

func (s *Service) UpdateNode(actor access.Identity, nodeID string, req NodeUpdateRequest) (NodeUpdateResponse, error) {
	return s.UpdateNodeWithContext(context.Background(), actor, nodeID, req)
}

func (s *Service) UpdateNodeWithContext(ctx context.Context, actor access.Identity, nodeID string, req NodeUpdateRequest) (NodeUpdateResponse, error) {
	if err := validateNodeUpdateRequest(req); err != nil {
		return NodeUpdateResponse{}, errors.Join(ErrValidation, err)
	}

	node, ok := s.store.GetNodeByID(nodeID)
	if !ok || node.IsDeleted {
		return NodeUpdateResponse{}, ErrNotFound
	}

	domain, err := s.ontology.GetVisibleDomain(actor, node.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return NodeUpdateResponse{}, ErrForbidden
		}
		return NodeUpdateResponse{}, err
	}
	if err := s.ensureNodeMutationPermission(actor, node, domain); err != nil {
		return NodeUpdateResponse{}, err
	}

	schema, err := s.ontology.GetNodeType(node.DomainID, node.NodeType)
	if err != nil {
		if errors.Is(err, ontology.ErrNotFound) {
			return NodeUpdateResponse{}, errors.Join(ErrValidation, fmt.Errorf("unknown node_type: %s", node.NodeType))
		}
		return NodeUpdateResponse{}, err
	}

	mergedProperties := cloneProperties(node.Properties)
	for key, value := range req.Properties {
		mergedProperties[key] = value
	}
	if err := validateProperties(schema, mergedProperties); err != nil {
		return NodeUpdateResponse{}, errors.Join(ErrValidation, err)
	}

	version, err := s.ontology.GetCurrentVersion(node.DomainID)
	if err != nil {
		return NodeUpdateResponse{}, err
	}
	statusCfg, err := s.ontology.GetStatusFieldConfig(node.DomainID)
	if err != nil {
		return NodeUpdateResponse{}, err
	}

	statusValue := node.StatusValue
	if statusCfg != nil && statusCfg.StatusFieldName != "" {
		if value, ok := mergedProperties[statusCfg.StatusFieldName]; ok {
			statusValue = fmt.Sprintf("%v", value)
		} else {
			statusValue = ""
		}
	}

	updated := node
	updated.Properties = mergedProperties
	updated.Visibility = fallback(req.Visibility, node.Visibility)
	if len(updated.ACLVisibleTo) == 0 {
		updated.ACLVisibleTo = []string{node.OwnerTenantID + ":" + node.OwnerAppID}
	}
	if strings.TrimSpace(req.ExternalRef) != "" || node.ExternalRef == "" {
		updated.ExternalRef = strings.TrimSpace(req.ExternalRef)
	}
	updated.DomainVersion = version.Version
	updated.StatusValue = statusValue
	updated.UpdatedAt = s.now()
	var scope session.SessionScope
	scope, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		event := OutboxEvent{
			ID:            newID("evt"),
			AggregateType: "kg_node",
			AggregateID:   nodeID,
			EventType:     "NODE_UPSERTED",
			Payload: map[string]any{
				"node_id":         nodeID,
				"domain_id":       updated.DomainID,
				"owner_tenant_id": updated.OwnerTenantID,
				"owner_app_id":    updated.OwnerAppID,
				"node_type":       updated.NodeType,
				"external_ref":    updated.ExternalRef,
				"tx_scope":        scope.Statements,
			},
			Status:     "PENDING",
			RetryCount: 0,
			CreatedAt:  updated.UpdatedAt,
		}
		repo := s.repositoryForScope(scope)
		if err := repo.UpdateNodeWithOutbox(ctx, updated, event); err != nil {
			switch {
			case errors.Is(err, ErrDuplicateExternalRef):
				return errors.Join(ErrValidation, errors.New("external_ref already exists"))
			case errors.Is(err, ErrNodeNotFound):
				return ErrNotFound
			default:
				return err
			}
		}
		return nil
	})
	if err != nil {
		return NodeUpdateResponse{}, err
	}
	_ = scope
	s.recordWriteAudit(actor, updated.OwnerTenantID, updated.OwnerAppID, "kg.node.update", "kg_node", updated.ID, "allow", "", map[string]any{
		"domain_id":    updated.DomainID,
		"node_type":    updated.NodeType,
		"external_ref": updated.ExternalRef,
	})

	return NodeUpdateResponse{
		NodeID:        nodeID,
		DomainVersion: version.Version,
		Status:        "processing",
	}, nil
}

func (s *Service) DeleteNode(actor access.Identity, nodeID string) (NodeDeleteResponse, error) {
	return s.DeleteNodeWithContext(context.Background(), actor, nodeID)
}

func (s *Service) DeleteNodeWithContext(ctx context.Context, actor access.Identity, nodeID string) (NodeDeleteResponse, error) {
	node, ok := s.store.GetNodeByID(nodeID)
	if !ok || node.IsDeleted {
		return NodeDeleteResponse{}, ErrNotFound
	}

	domain, err := s.ontology.GetVisibleDomain(actor, node.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return NodeDeleteResponse{}, ErrForbidden
		}
		return NodeDeleteResponse{}, err
	}
	if err := s.ensureNodeMutationPermission(actor, node, domain); err != nil {
		return NodeDeleteResponse{}, err
	}

	deleted := node
	deleted.IsDeleted = true
	deleted.UpdatedAt = s.now()
	_, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		event := OutboxEvent{
			ID:            newID("evt"),
			AggregateType: "kg_node",
			AggregateID:   nodeID,
			EventType:     "NODE_DELETED",
			Payload: map[string]any{
				"node_id":         nodeID,
				"domain_id":       deleted.DomainID,
				"owner_tenant_id": deleted.OwnerTenantID,
				"owner_app_id":    deleted.OwnerAppID,
				"tx_scope":        scope.Statements,
			},
			Status:     "PENDING",
			RetryCount: 0,
			CreatedAt:  deleted.UpdatedAt,
		}
		repo := s.repositoryForScope(scope)
		if err := repo.SoftDeleteNodeWithOutbox(ctx, deleted, event); err != nil {
			if errors.Is(err, ErrNodeNotFound) {
				return ErrNotFound
			}
			return err
		}
		return nil
	})
	if err != nil {
		return NodeDeleteResponse{}, err
	}
	s.recordWriteAudit(actor, deleted.OwnerTenantID, deleted.OwnerAppID, "kg.node.delete", "kg_node", deleted.ID, "allow", "", map[string]any{
		"domain_id": deleted.DomainID,
	})

	return NodeDeleteResponse{
		NodeID:    nodeID,
		IsDeleted: true,
	}, nil
}

func (s *Service) CreateRelationship(actor access.Identity, req RelationshipCreateRequest) (RelationshipCreateResponse, error) {
	return s.CreateRelationshipWithContext(context.Background(), actor, req)
}

func (s *Service) CreateRelationshipWithContext(ctx context.Context, actor access.Identity, req RelationshipCreateRequest) (RelationshipCreateResponse, error) {
	if err := validateRelationshipCreateRequest(req); err != nil {
		return RelationshipCreateResponse{}, errors.Join(ErrValidation, err)
	}

	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return RelationshipCreateResponse{}, ErrForbidden
		}
		return RelationshipCreateResponse{}, err
	}
	if err := s.ensureWritePermission(actor, domain, req.DomainID); err != nil {
		return RelationshipCreateResponse{}, err
	}

	fromNode, ok := s.store.GetNodeByID(req.FromNodeID)
	if !ok || fromNode.IsDeleted {
		return RelationshipCreateResponse{}, ErrNotFound
	}
	toNode, ok := s.store.GetNodeByID(req.ToNodeID)
	if !ok || toNode.IsDeleted {
		return RelationshipCreateResponse{}, ErrNotFound
	}
	if fromNode.DomainID != req.DomainID || toNode.DomainID != req.DomainID {
		return RelationshipCreateResponse{}, errors.Join(ErrValidation, errors.New("relationship endpoints must belong to the target domain"))
	}

	schema, err := s.ontology.GetRelType(req.DomainID, req.RelType, fromNode.NodeType, toNode.NodeType)
	if err != nil {
		if errors.Is(err, ontology.ErrNotFound) {
			return RelationshipCreateResponse{}, errors.Join(ErrValidation, fmt.Errorf("unknown relationship type or direction: %s", req.RelType))
		}
		return RelationshipCreateResponse{}, err
	}
	if err := validateRelationshipProperties(schema, req.Properties); err != nil {
		return RelationshipCreateResponse{}, errors.Join(ErrValidation, err)
	}

	version, err := s.ontology.GetCurrentVersion(req.DomainID)
	if err != nil {
		return RelationshipCreateResponse{}, err
	}
	now := s.now()
	relID := newID("rel")
	rel := RelationshipRecord{
		ID:            relID,
		RelType:       req.RelType,
		FromNodeID:    req.FromNodeID,
		ToNodeID:      req.ToNodeID,
		DomainID:      req.DomainID,
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		DomainVersion: version.Version,
		Properties:    req.Properties,
		CreatedAt:     now,
	}
	_, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		event := OutboxEvent{
			ID:            newID("evt"),
			AggregateType: "kg_relationship",
			AggregateID:   relID,
			EventType:     "RELATIONSHIP_UPSERTED",
			Payload: map[string]any{
				"relationship_id": relID,
				"domain_id":       req.DomainID,
				"from_node_id":    req.FromNodeID,
				"to_node_id":      req.ToNodeID,
				"rel_type":        req.RelType,
				"domain_version":  version.Version,
				"tx_scope":        scope.Statements,
			},
			Status:     "PENDING",
			RetryCount: 0,
			CreatedAt:  now,
		}
		repo := s.repositoryForScope(scope)
		if err := repo.CreateRelationshipWithOutbox(ctx, rel, event); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return RelationshipCreateResponse{}, err
	}
	s.recordWriteAudit(actor, rel.OwnerTenantID, rel.OwnerAppID, "kg.relationship.create", "kg_relationship", rel.ID, "allow", "", map[string]any{
		"domain_id":    rel.DomainID,
		"rel_type":     rel.RelType,
		"from_node_id": rel.FromNodeID,
		"to_node_id":   rel.ToNodeID,
	})
	return RelationshipCreateResponse{
		RelationshipID: relID,
		Status:         "processing",
	}, nil
}

func (s *Service) recordWriteAudit(actor access.Identity, ownerTenantID, ownerAppID, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.RecordWriteAudit(actor, ownerTenantID, ownerAppID, action, resourceType, resourceID, outcome, reason, metadata)
}

func canRunIngest(actor access.Identity) bool {
	switch actor.AppType {
	case "ingestion_producer", "admin_tool", "hybrid":
		return true
	default:
		return false
	}
}

func (s *Service) ensureWritePermission(actor access.Identity, domain ontology.Domain, domainID string) error {
	if domain.OwnerTenantID == actor.TenantID {
		return nil
	}

	visibleOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return err
	}
	for _, owner := range visibleOwners {
		if owner.Source != "grant" {
			continue
		}
		if owner.TenantID != domain.OwnerTenantID {
			continue
		}
		if owner.Permission != "write" && owner.Permission != "admin" {
			continue
		}
		if owner.ScopeType == "all" || (owner.ScopeType == "domain" && owner.ScopeValue == domainID) {
			return nil
		}
	}
	return ErrForbidden
}

func (s *Service) ensureNodeMutationPermission(actor access.Identity, node NodeRecord, domain ontology.Domain) error {
	if node.OwnerTenantID == actor.TenantID {
		return nil
	}
	return s.ensureWritePermission(actor, domain, node.DomainID)
}

func (s *Service) buildBridgeRelationships(domainID, nodeType string, actor access.Identity, fromNodeID string, properties map[string]any) ([]RelationshipRecord, error) {
	rules := s.ontology.ResolveCrossDomainRules(domainID, nodeType)
	now := s.now()
	var rels []RelationshipRecord
	for _, rule := range rules {
		rawValue, ok := properties[rule.BridgePropertyKey]
		if !ok {
			if rule.Required {
				return nil, errors.Join(ErrValidation, fmt.Errorf("%s required: must provide %s", rule.RelTypeName, rule.BridgePropertyKey))
			}
			continue
		}
		targetIDs, ok := rawValue.([]any)
		if !ok || len(targetIDs) == 0 {
			if rule.Required {
				return nil, errors.Join(ErrValidation, fmt.Errorf("%s required: must provide %s", rule.RelTypeName, rule.BridgePropertyKey))
			}
			continue
		}
		for _, rawTarget := range targetIDs {
			targetID, ok := rawTarget.(string)
			if !ok || strings.TrimSpace(targetID) == "" {
				return nil, errors.Join(ErrValidation, fmt.Errorf("%s entries must be non-empty strings", rule.BridgePropertyKey))
			}
			targetNode, ok := s.store.GetNodeByID(targetID)
			if !ok || targetNode.IsDeleted {
				return nil, errors.Join(ErrValidation, fmt.Errorf("%s target node not found: %s", rule.BridgePropertyKey, targetID))
			}
			if err := s.ontology.ValidateCrossDomainTarget(rule, targetNode.DomainID, targetNode.NodeType); err != nil {
				return nil, errors.Join(ErrValidation, fmt.Errorf("%s target invalid: %w", rule.BridgePropertyKey, err))
			}
			rels = append(rels, RelationshipRecord{
				ID:            newID("rel"),
				RelType:       rule.RelTypeName,
				FromNodeID:    fromNodeID,
				ToNodeID:      targetID,
				DomainID:      domainID,
				OwnerTenantID: actor.TenantID,
				OwnerAppID:    actor.AppID,
				Properties:    map[string]any{},
				CreatedAt:     now,
			})
		}
	}
	return rels, nil
}

func validateNodeCreateRequest(req NodeCreateRequest) error {
	if strings.TrimSpace(req.DomainID) == "" {
		return errors.New("domain_id is required")
	}
	if strings.TrimSpace(req.NodeType) == "" {
		return errors.New("node_type is required")
	}
	if req.Properties == nil {
		return errors.New("properties is required")
	}
	if req.Visibility != "" && req.Visibility != "public" && req.Visibility != "tenant_shared" && req.Visibility != "private" {
		return fmt.Errorf("invalid visibility: %s", req.Visibility)
	}
	return nil
}

func validateNodeUpdateRequest(req NodeUpdateRequest) error {
	if req.Properties == nil && strings.TrimSpace(req.Visibility) == "" && strings.TrimSpace(req.ExternalRef) == "" {
		return errors.New("at least one mutable field is required")
	}
	if req.Visibility != "" && req.Visibility != "public" && req.Visibility != "tenant_shared" && req.Visibility != "private" {
		return fmt.Errorf("invalid visibility: %s", req.Visibility)
	}
	return nil
}

func validateRelationshipCreateRequest(req RelationshipCreateRequest) error {
	if strings.TrimSpace(req.RelType) == "" {
		return errors.New("rel_type is required")
	}
	if strings.TrimSpace(req.FromNodeID) == "" {
		return errors.New("from_node_id is required")
	}
	if strings.TrimSpace(req.ToNodeID) == "" {
		return errors.New("to_node_id is required")
	}
	if strings.TrimSpace(req.DomainID) == "" {
		return errors.New("domain_id is required")
	}
	if req.Properties == nil {
		req.Properties = map[string]any{}
	}
	return nil
}

func validateIngestDocumentRequest(req IngestDocumentRequest) error {
	if strings.TrimSpace(req.FileURL) == "" {
		return errors.New("file_url is required")
	}
	if strings.TrimSpace(req.DomainID) == "" {
		return errors.New("domain_id is required")
	}
	if strings.TrimSpace(req.LoaiVanBan) == "" {
		return errors.New("loai_van_ban is required")
	}
	return nil
}

func validateRelationshipProperties(schema ontology.RelTypeSchema, properties map[string]any) error {
	for _, prop := range schema.RequiredProps {
		value, ok := properties[prop.Name]
		if !ok {
			return fmt.Errorf("missing required property: %s", prop.Name)
		}
		if err := validatePropertyType(prop.Type, value); err != nil {
			return fmt.Errorf("%s: %w", prop.Name, err)
		}
	}
	for _, prop := range schema.OptionalProps {
		if value, ok := properties[prop.Name]; ok {
			if err := validatePropertyType(prop.Type, value); err != nil {
				return fmt.Errorf("%s: %w", prop.Name, err)
			}
		}
	}
	return nil
}

func validateProperties(schema ontology.NodeTypeSchema, properties map[string]any) error {
	for _, prop := range schema.RequiredProps {
		value, ok := properties[prop.Name]
		if !ok {
			return fmt.Errorf("missing required property: %s", prop.Name)
		}
		if err := validatePropertyType(prop.Type, value); err != nil {
			return fmt.Errorf("%s: %w", prop.Name, err)
		}
	}
	for _, prop := range schema.OptionalProps {
		if value, ok := properties[prop.Name]; ok {
			if err := validatePropertyType(prop.Type, value); err != nil {
				return fmt.Errorf("%s: %w", prop.Name, err)
			}
		}
	}
	return nil
}

func validatePropertyType(expected string, value any) error {
	switch expected {
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("must be a string")
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
		default:
			return errors.New("must be a number")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("must be a boolean")
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return errors.New("must be an object")
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return errors.New("must be an array")
		}
	}
	return nil
}

func cloneProperties(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func newID(prefix string) string {
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000")
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
