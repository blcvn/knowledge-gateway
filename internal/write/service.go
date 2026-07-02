package write

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/identity"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/session"
	"kg-service/internal/telemetry"
)

var (
	ErrForbidden        = errors.New("forbidden")
	ErrValidation       = errors.New("validation")
	ErrNotFound         = errors.New("not found")
	ErrScopeLocked      = errors.New("sync scope locked")
	ErrSessionAbandoned = errors.New("sync session abandoned")
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
	store               Repository
	ontology            OntologyResolver
	accessResolver      AccessResolver
	sessionManager      SessionManager
	auditLogger         AuditLogger
	ftsBackendKind      string
	syncEtaDefaultMs    int
	syncLagToleranceMs  int
	syncLagStuckRetries int
	ingestMu            sync.RWMutex
	ingestJobs          map[string]IngestJobResponse
	now                 func() time.Time
}

func NewService(store Repository, ontology OntologyResolver, accessResolver AccessResolver, sessionManager SessionManager, auditLogger AuditLogger) *Service {
	return &Service{
		store:               store,
		ontology:            ontology,
		accessResolver:      accessResolver,
		sessionManager:      sessionManager,
		auditLogger:         auditLogger,
		syncEtaDefaultMs:    5000,
		syncLagToleranceMs:  30000,
		syncLagStuckRetries: 3,
		ingestJobs:          map[string]IngestJobResponse{},
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) SetSyncETAConfig(defaultMs int) {
	if defaultMs > 0 {
		s.syncEtaDefaultMs = defaultMs
	}
}

func (s *Service) SetSyncLagConfig(toleranceMs, stuckRetries int) {
	if toleranceMs > 0 {
		s.syncLagToleranceMs = toleranceMs
	}
	if stuckRetries > 0 {
		s.syncLagStuckRetries = stuckRetries
	}
}

func (s *Service) SetFTSBackendKind(kind string) {
	if s == nil {
		return
	}
	s.ftsBackendKind = strings.ToLower(strings.TrimSpace(kind))
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

func (s *Service) sealGraphVersion(ctx context.Context, repo Repository, actor access.Identity, graphScope, referenceID, changeSummary string, entities []GraphVersionEntityRecord) (GraphIdentityRecord, GraphVersionRecord, error) {
	return s.sealGraphVersionWithStatus(ctx, repo, actor, graphScope, referenceID, changeSummary, "ONLINE", "SEALED", entities)
}

func (s *Service) sealGraphVersionWithStatus(ctx context.Context, repo Repository, actor access.Identity, graphScope, referenceID, changeSummary, storageClass, versionStatus string, entities []GraphVersionEntityRecord) (GraphIdentityRecord, GraphVersionRecord, error) {
	if strings.TrimSpace(referenceID) == "" {
		referenceID = newID("ref")
	}
	identity, graphVersion, err := repo.SealGraphVersion(ctx, GraphVersionSealRequest{
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		GraphScope:    graphScope,
		ReferenceID:   referenceID,
		StorageClass:  storageClass,
		VersionStatus: versionStatus,
		ChangeSummary: changeSummary,
		Entities:      entities,
	})
	if err != nil {
		return GraphIdentityRecord{}, GraphVersionRecord{}, err
	}
	if strings.EqualFold(s.ftsBackendKind, "postgres") && strings.EqualFold(versionStatus, "SEALED") {
		if err := repo.UpsertGraphProjectionHead(ctx, GraphProjectionHeadRecord{
			IdentifierID:         identity.IdentifierID,
			BackendKind:          "fts",
			BackendName:          "postgres",
			AppliedVersionID:     graphVersion.VersionID,
			AppliedVersionNumber: graphVersion.VersionNumber,
			UpdatedAt:            time.Now().UTC(),
		}); err != nil {
			return GraphIdentityRecord{}, GraphVersionRecord{}, err
		}
	}
	return identity, graphVersion, nil
}

func (s *Service) finalizeGraphVersion(ctx context.Context, repo Repository, versionID string) (int64, error) {
	if repo == nil || strings.TrimSpace(versionID) == "" {
		return 0, nil
	}
	return repo.FinalizeGraphVersion(ctx, versionID)
}

const syncSessionLeaseTTL = 2 * time.Hour

type syncSessionState struct {
	version  GraphVersionRecord
	identity GraphIdentityRecord
	lease    ScopeLeaseRecord
}

func (s *Service) loadSyncSessionForWrite(ctx context.Context, repo Repository, actor access.Identity, versionID string) (syncSessionState, error) {
	state := syncSessionState{}
	version, ok := repo.GetGraphVersionByID(ctx, versionID)
	if !ok {
		return state, ErrNotFound
	}
	if strings.EqualFold(version.VersionStatus, "ABANDONED") {
		return state, ErrSessionAbandoned
	}
	if !strings.EqualFold(version.VersionStatus, "PENDING_ENTITIES") {
		return state, errors.Join(ErrValidation, errors.New("sync session is not open"))
	}
	identity, ok := repo.GetGraphIdentityByID(ctx, version.IdentifierID)
	if !ok {
		return state, ErrNotFound
	}
	if identity.OwnerTenantID != actor.TenantID || identity.OwnerAppID != actor.AppID {
		return state, ErrForbidden
	}
	lease, ok := repo.GetScopeLease(ctx, identity.OwnerTenantID, identity.OwnerAppID, identity.GraphScope)
	if !ok || lease.VersionID != versionID {
		return state, ErrScopeLocked
	}
	return syncSessionState{version: version, identity: identity, lease: lease}, nil
}

func (s *Service) loadSyncSessionForCommit(ctx context.Context, repo Repository, actor access.Identity, versionID string) (syncSessionState, error) {
	state := syncSessionState{}
	version, ok := repo.GetGraphVersionByID(ctx, versionID)
	if !ok {
		return state, ErrNotFound
	}
	identity, ok := repo.GetGraphIdentityByID(ctx, version.IdentifierID)
	if !ok {
		return state, ErrNotFound
	}
	if identity.OwnerTenantID != actor.TenantID || identity.OwnerAppID != actor.AppID {
		return state, ErrForbidden
	}
	state.version = version
	state.identity = identity
	if strings.EqualFold(version.VersionStatus, "SEALED") {
		return state, nil
	}
	if strings.EqualFold(version.VersionStatus, "ABANDONED") {
		return state, ErrSessionAbandoned
	}
	if !strings.EqualFold(version.VersionStatus, "PENDING_ENTITIES") {
		return state, errors.Join(ErrValidation, errors.New("sync session is not open"))
	}
	lease, ok := repo.GetScopeLease(ctx, identity.OwnerTenantID, identity.OwnerAppID, identity.GraphScope)
	if !ok || lease.VersionID != versionID {
		return state, ErrScopeLocked
	}
	state.lease = lease
	return state, nil
}

func (s *Service) OpenSyncSession(ctx context.Context, actor access.Identity, req OpenSyncSessionRequest) (SyncSessionResponse, error) {
	if strings.TrimSpace(req.DomainID) == "" {
		return SyncSessionResponse{}, errors.Join(ErrValidation, errors.New("domain_id is required"))
	}
	if strings.TrimSpace(req.GraphScope) == "" {
		return SyncSessionResponse{}, errors.Join(ErrValidation, errors.New("graph_scope is required"))
	}
	log.Printf("write open_sync_session start tenant=%s app=%s domain=%s graph_scope=%s", actor.TenantID, actor.AppID, req.DomainID, req.GraphScope)
	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return SyncSessionResponse{}, ErrForbidden
		}
		return SyncSessionResponse{}, err
	}
	if err := s.ensureWritePermission(actor, domain, req.DomainID); err != nil {
		return SyncSessionResponse{}, err
	}
	var resp SyncSessionResponse
	_, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		identity, version, err := s.sealGraphVersionWithStatus(ctx, repo, actor, req.GraphScope, "", "sync session open", "ONLINE", "PENDING_ENTITIES", nil)
		if err != nil {
			return err
		}
		if err := repo.AcquireScopeLease(ctx, actor.TenantID, actor.AppID, req.GraphScope, version.VersionID, s.now().Add(syncSessionLeaseTTL)); err != nil {
			return err
		}
		resp = SyncSessionResponse{
			SessionID:          version.VersionID,
			GraphVersionID:     version.VersionID,
			GraphIdentifierID:  identity.IdentifierID,
			GraphVersionNumber: version.VersionNumber,
		}
		log.Printf(
			"write open_sync_session ok tenant=%s app=%s domain=%s graph_scope=%s session_id=%s graph_version_id=%s graph_version_number=%d",
			actor.TenantID,
			actor.AppID,
			req.DomainID,
			req.GraphScope,
			resp.SessionID,
			resp.GraphVersionID,
			resp.GraphVersionNumber,
		)
		return nil
	})
	if err != nil {
		return SyncSessionResponse{}, err
	}
	return resp, nil
}

func (s *Service) CommitSyncSession(ctx context.Context, actor access.Identity, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.Join(ErrValidation, errors.New("session_id is required"))
	}
	log.Printf("write commit_sync_session start tenant=%s app=%s session_id=%s", actor.TenantID, actor.AppID, sessionID)
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		state, err := s.loadSyncSessionForCommit(ctx, repo, actor, sessionID)
		if err != nil {
			return err
		}
		if strings.EqualFold(state.version.VersionStatus, "SEALED") {
			log.Printf(
				"write commit_sync_session ok tenant=%s app=%s session_id=%s graph_identifier_id=%s graph_version_id=%s graph_version_number=%d status=SEALED",
				actor.TenantID,
				actor.AppID,
				sessionID,
				state.identity.IdentifierID,
				state.version.VersionID,
				state.version.VersionNumber,
			)
			return repo.ReleaseScopeLease(ctx, state.identity.OwnerTenantID, state.identity.OwnerAppID, state.identity.GraphScope, sessionID)
		}
		finalized, err := s.finalizeGraphVersion(ctx, repo, sessionID)
		if err != nil {
			return err
		}
		if finalized == 0 {
			current, ok := repo.GetGraphVersionByID(ctx, sessionID)
			if !ok {
				return ErrNotFound
			}
			switch {
			case strings.EqualFold(current.VersionStatus, "SEALED"):
				log.Printf(
					"write commit_sync_session ok tenant=%s app=%s session_id=%s graph_identifier_id=%s graph_version_id=%s graph_version_number=%d status=SEALED",
					actor.TenantID,
					actor.AppID,
					sessionID,
					state.identity.IdentifierID,
					current.VersionID,
					current.VersionNumber,
				)
				return repo.ReleaseScopeLease(ctx, state.identity.OwnerTenantID, state.identity.OwnerAppID, state.identity.GraphScope, sessionID)
			case strings.EqualFold(current.VersionStatus, "ABANDONED"):
				return ErrSessionAbandoned
			default:
				return ErrScopeLocked
			}
		}
		current, ok := repo.GetGraphVersionByID(ctx, sessionID)
		if !ok {
			return ErrNotFound
		}
		if !strings.EqualFold(current.VersionStatus, "SEALED") {
			if strings.EqualFold(current.VersionStatus, "ABANDONED") {
				return ErrSessionAbandoned
			}
			return ErrValidation
		}
		event := OutboxEvent{
			ID:            deterministicUUID("graph-version-sealed:" + sessionID),
			AggregateType: "kg_graph_version",
			AggregateID:   sessionID,
			EventType:     "GRAPH_VERSION_SEALED",
			Payload: map[string]any{
				"graph_identifier_id":  state.identity.IdentifierID,
				"graph_version_id":     sessionID,
				"graph_version_number": current.VersionNumber,
				"graph_scope":          state.identity.GraphScope,
				"owner_tenant_id":      state.identity.OwnerTenantID,
				"owner_app_id":         state.identity.OwnerAppID,
			},
			Status:     "PENDING",
			RetryCount: 0,
			CreatedAt:  s.now(),
		}
		if err := repo.CreateOutboxEvents(ctx, []OutboxEvent{event}); err != nil {
			if strings.Contains(err.Error(), "duplicate key value") {
				return repo.ReleaseScopeLease(ctx, state.identity.OwnerTenantID, state.identity.OwnerAppID, state.identity.GraphScope, sessionID)
			}
			return err
		}
		if err := repo.ReleaseScopeLease(ctx, state.identity.OwnerTenantID, state.identity.OwnerAppID, state.identity.GraphScope, sessionID); err != nil {
			return err
		}
		log.Printf(
			"write commit_sync_session ok tenant=%s app=%s session_id=%s graph_identifier_id=%s graph_version_id=%s graph_version_number=%d",
			actor.TenantID,
			actor.AppID,
			sessionID,
			state.identity.IdentifierID,
			current.VersionID,
			current.VersionNumber,
		)
		return nil
	})
	return err
}

func (s *Service) AbandonSyncSession(ctx context.Context, actor access.Identity, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.Join(ErrValidation, errors.New("session_id is required"))
	}
	log.Printf("write abandon_sync_session start tenant=%s app=%s session_id=%s", actor.TenantID, actor.AppID, sessionID)
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		state, err := s.loadSyncSessionForCommit(ctx, repo, actor, sessionID)
		if err != nil {
			return err
		}
		if strings.EqualFold(state.version.VersionStatus, "SEALED") {
			log.Printf("write abandon_sync_session ok tenant=%s app=%s session_id=%s status=SEALED", actor.TenantID, actor.AppID, sessionID)
			return repo.ReleaseScopeLease(ctx, state.identity.OwnerTenantID, state.identity.OwnerAppID, state.identity.GraphScope, sessionID)
		}
		if err := repo.AbandonGraphVersion(ctx, sessionID); err != nil {
			return err
		}
		log.Printf(
			"write abandon_sync_session ok tenant=%s app=%s session_id=%s graph_identifier_id=%s graph_version_id=%s",
			actor.TenantID,
			actor.AppID,
			sessionID,
			state.identity.IdentifierID,
			state.version.VersionID,
		)
		return repo.ReleaseScopeLease(ctx, state.identity.OwnerTenantID, state.identity.OwnerAppID, state.identity.GraphScope, sessionID)
	})
	return err
}

func graphVersionEntities(entityKind string, changeKind string, ids ...string) []GraphVersionEntityRecord {
	entities := make([]GraphVersionEntityRecord, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		entities = append(entities, GraphVersionEntityRecord{
			EntityKind: entityKind,
			EntityID:   id,
			ChangeKind: changeKind,
		})
	}
	return entities
}

func mergeGraphVersionEntities(sets ...[]GraphVersionEntityRecord) []GraphVersionEntityRecord {
	total := 0
	for _, set := range sets {
		total += len(set)
	}
	entities := make([]GraphVersionEntityRecord, 0, total)
	for _, set := range sets {
		entities = append(entities, set...)
	}
	return entities
}

func relationshipIDs(rels []RelationshipRecord) []string {
	ids := make([]string, 0, len(rels))
	for _, rel := range rels {
		ids = append(ids, rel.ID)
	}
	return ids
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

func (s *Service) EntitySyncStatus(entityID, entityKind string) (map[string]any, error) {
	record, ok := s.store.GetProjectionVersion(entityID, entityKind)
	if !ok {
		return nil, ErrNotFound
	}
	events := s.store.ListOutboxEvents()
	eventByID := make(map[string]OutboxEvent, len(events))
	for _, event := range events {
		eventByID[event.ID] = event
	}
	graphProjectionReady, graphProjectionHeadVersion, graphProjectionHeadVersionID := s.graphProjectionReady(record, eventByID)
	graphVersion := maxInt64(record.GraphVersion, graphProjectionHeadVersion)
	status := map[string]any{
		"entity_id":                        entityID,
		"entity_kind":                      entityKind,
		"source_version":                   record.SourceVersion,
		"graph_version":                    graphVersion,
		"graph_projection_ready":           graphProjectionReady,
		"graph_projection_head_version":    graphProjectionHeadVersion,
		"graph_projection_head_version_id": graphProjectionHeadVersionID,
		"graph_lag_class":                  classifyGraphLag(record.GraphVersion, record.SourceVersion, record.SourceEventID, record.LastGraphSyncedAt, eventByID, s.syncLagStuckRetries, time.Duration(s.syncLagToleranceMs)*time.Millisecond, graphProjectionReady),
		"last_graph_synced_at":             record.LastGraphSyncedAt,
		"vector_version":                   record.VectorVersion,
		"vector_lag_class":                 classifyReplicaLag(record.VectorVersion, record.SourceVersion, record.SourceEventID, record.LastVectorSyncedAt, eventByID, s.syncLagStuckRetries, time.Duration(s.syncLagToleranceMs)*time.Millisecond),
		"last_vector_synced_at":            record.LastVectorSyncedAt,
	}
	if entityKind == "kg_relationship" && record.VectorBackend == "" && record.VectorVersion == 0 {
		status["vector_lag_class"] = "SYNCED"
	}
	return status, nil
}

func (s *Service) graphProjectionReady(record ProjectionVersionRecord, events map[string]OutboxEvent) (bool, int64, string) {
	if record.SourceEventID == "" {
		return false, 0, ""
	}
	event, ok := events[record.SourceEventID]
	if !ok {
		return false, 0, ""
	}
	identifierID, _, versionNumber, ok := graphVersionMetadata(event)
	if !ok {
		return false, 0, ""
	}
	head, ok := s.store.GetGraphProjectionHead(identifierID, "graph", "")
	if !ok {
		return false, 0, ""
	}
	return head.AppliedVersionNumber >= versionNumber, head.AppliedVersionNumber, head.AppliedVersionID
}

func graphVersionMetadata(event OutboxEvent) (string, string, int64, bool) {
	identifierID, _ := event.Payload["graph_identifier_id"].(string)
	versionID, _ := event.Payload["graph_version_id"].(string)
	versionNumber := int64(0)
	switch raw := event.Payload["graph_version_number"].(type) {
	case int64:
		versionNumber = raw
	case int:
		versionNumber = int64(raw)
	case float64:
		versionNumber = int64(raw)
	}
	if identifierID == "" || versionID == "" || versionNumber <= 0 {
		return "", "", 0, false
	}
	return identifierID, versionID, versionNumber, true
}

func classifyGraphLag(replicaVersion, sourceVersion int64, sourceEventID string, lastSyncedAt time.Time, events map[string]OutboxEvent, maxRetries int, lagToleranceWindow time.Duration, projectionReady bool) string {
	effectiveReplicaVersion := replicaVersion
	if !projectionReady && replicaVersion == sourceVersion && sourceVersion > 0 {
		effectiveReplicaVersion = sourceVersion - 1
	}
	return classifyReplicaLag(effectiveReplicaVersion, sourceVersion, sourceEventID, lastSyncedAt, events, maxRetries, lagToleranceWindow)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (s *Service) CreateNode(actor access.Identity, req NodeCreateRequest) (NodeCreateResponse, error) {
	return s.CreateNodeWithContext(context.Background(), actor, req)
}

func (s *Service) CreateNodeWithContext(ctx context.Context, actor access.Identity, req NodeCreateRequest) (NodeCreateResponse, error) {
	var resp NodeCreateResponse
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		created, err := s.createNodeInScope(ctx, scope, actor, req)
		if err != nil {
			return err
		}
		resp = created
		return nil
	})
	if err != nil {
		return NodeCreateResponse{}, err
	}
	return resp, nil
}

func (s *Service) CreateNodesBulkWithContext(ctx context.Context, actor access.Identity, req NodeBulkCreateRequest) (NodeBulkCreateResponse, error) {
	if len(req.Nodes) == 0 {
		return NodeBulkCreateResponse{Succeeded: []NodeCreateResponse{}, Failed: []BulkItemError{}}, nil
	}
	log.Printf("write create_nodes_bulk start tenant=%s app=%s requested=%d graph_version_id=%s", actor.TenantID, actor.AppID, len(req.Nodes), strings.TrimSpace(req.GraphVersionID))

	existing := map[string]NodeRecord{}
	if len(req.Nodes) > 0 {
		refs := make([]string, 0, len(req.Nodes))
		seen := map[string]struct{}{}
		for _, item := range req.Nodes {
			if ref := strings.TrimSpace(item.ExternalRef); ref != "" {
				if _, ok := seen[ref]; !ok {
					seen[ref] = struct{}{}
					refs = append(refs, ref)
				}
			}
		}
		existing = s.store.GetNodesByExternalRefs(refs)
	}

	lookup := bulkNodeLookup{
		base:         s.store,
		externalRefs: existing,
		staged:       map[string]NodeRecord{},
	}
	type bulkNodeDraft struct {
		node               NodeRecord
		bridges            []RelationshipRecord
		referenceID        string
		graphScope         string
		graphIdentifierID  string
		graphVersionID     string
		graphVersionNumber int64
		response           NodeCreateResponse
	}
	drafts := make([]bulkNodeDraft, 0, len(req.Nodes))
	result := NodeBulkCreateResponse{Succeeded: []NodeCreateResponse{}, Failed: []BulkItemError{}}
	for idx, item := range req.Nodes {
		node, bridges, err := s.previewNodeCreateWithBridge(actor, item, lookup)
		if err != nil {
			result.Failed = append(result.Failed, BulkItemError{Index: idx, ExternalRef: strings.TrimSpace(item.ExternalRef), Error: err.Error()})
			continue
		}
		lookup.staged[node.ID] = node
		drafts = append(drafts, bulkNodeDraft{
			node:        node,
			bridges:     bridges,
			referenceID: strings.TrimSpace(item.ReferenceID),
			response: NodeCreateResponse{
				NodeID:        node.ID,
				DomainVersion: node.DomainVersion,
				Status:        "processing",
				SyncETAMs:     estimateSyncETA(s.store, node.DomainID, s.syncEtaDefaultMs),
			},
		})
	}
	if len(drafts) == 0 {
		log.Printf(
			"write create_nodes_bulk rejected tenant=%s app=%s requested=%d failed=%d graph_version_id=%s",
			actor.TenantID,
			actor.AppID,
			len(req.Nodes),
			len(result.Failed),
			strings.TrimSpace(req.GraphVersionID),
		)
		return result, nil
	}
	if strings.TrimSpace(req.GraphVersionID) != "" {
		_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
			TenantID: actor.TenantID,
			AppID:    actor.AppID,
		}, func(scope session.SessionScope) error {
			repo := s.repositoryForScope(scope)
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, req.GraphVersionID)
			if err != nil {
				return err
			}
			nodes := make([]NodeRecord, 0, len(drafts))
			bridgeRels := make([]RelationshipRecord, 0)
			entities := make([]GraphVersionEntityRecord, 0, len(drafts)*2)
			for i, draft := range drafts {
				if graphScope := deriveGraphScopeForNode(draft.node); graphScope != state.identity.GraphScope {
					return errors.Join(ErrValidation, fmt.Errorf("node graph scope mismatch: %s != %s", graphScope, state.identity.GraphScope))
				}
				drafts[i].response = NodeCreateResponse{
					NodeID:             draft.node.ID,
					GraphIdentifierID:  state.identity.IdentifierID,
					GraphVersionID:     state.version.VersionID,
					GraphVersionNumber: state.version.VersionNumber,
					ReferenceID:        draft.referenceID,
					DomainVersion:      draft.node.DomainVersion,
					Status:             "processing",
					SyncETAMs:          estimateSyncETA(s.store, draft.node.DomainID, s.syncEtaDefaultMs),
				}
				nodes = append(nodes, draft.node)
				bridgeRels = append(bridgeRels, draft.bridges...)
				entities = append(entities, graphVersionEntities("node", "UPSERT", draft.node.ID)...)
				bridgeEntityIDs := relationshipIDs(draft.bridges)
				entities = append(entities, graphVersionEntities("embeddable_relationship", "UPSERT", bridgeEntityIDs...)...)
			}
			if err := repo.CreateNodesBulkWithOutbox(ctx, nodes, nil); err != nil {
				if errors.Is(err, ErrDuplicateExternalRef) {
					return errors.Join(ErrValidation, errors.New("external_ref already exists"))
				}
				return err
			}
			if len(bridgeRels) > 0 {
				if err := repo.CreateRelationshipsBulkWithOutbox(ctx, bridgeRels, nil); err != nil {
					return err
				}
			}
			return repo.AddGraphVersionEntities(ctx, state.version.VersionID, entities)
		})
		if err != nil {
			return NodeBulkCreateResponse{}, err
		}
		telemetry.RecordBulkWriteBatchSize(len(req.Nodes))
		telemetry.RecordBulkWritePartialFailure(len(req.Nodes), len(result.Failed))
		for _, draft := range drafts {
			result.Succeeded = append(result.Succeeded, draft.response)
		}
		log.Printf(
			"write create_nodes_bulk ok tenant=%s app=%s requested=%d succeeded=%d failed=%d graph_version_id=%s mode=session",
			actor.TenantID,
			actor.AppID,
			len(req.Nodes),
			len(result.Succeeded),
			len(result.Failed),
			strings.TrimSpace(req.GraphVersionID),
		)
		return result, nil
	}
	bridgeRels := make([]RelationshipRecord, 0)

	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		nodes := make([]NodeRecord, 0, len(drafts))
		for i, draft := range drafts {
			graphScope := deriveGraphScopeForNode(draft.node)
			entities := mergeGraphVersionEntities(
				graphVersionEntities("node", "UPSERT", draft.node.ID),
				graphVersionEntities("embeddable_relationship", "UPSERT", relationshipIDs(draft.bridges)...),
			)
			identity, graphVersion, err := s.sealGraphVersionWithStatus(ctx, repo, actor, graphScope, draft.referenceID, "node create", "ONLINE", "PENDING_ENTITIES", entities)
			if err != nil {
				return err
			}
			drafts[i].referenceID = graphVersion.ReferenceID
			drafts[i].graphScope = graphScope
			drafts[i].graphIdentifierID = identity.IdentifierID
			drafts[i].graphVersionID = graphVersion.VersionID
			drafts[i].graphVersionNumber = graphVersion.VersionNumber
			nodes = append(nodes, draft.node)
			bridgeRels = append(bridgeRels, draft.bridges...)
		}
		if err := repo.CreateNodesBulkWithOutbox(ctx, nodes, nil); err != nil {
			if errors.Is(err, ErrDuplicateExternalRef) {
				return errors.Join(ErrValidation, errors.New("external_ref already exists"))
			}
			return err
		}
		if len(bridgeRels) > 0 {
			if err := repo.CreateRelationshipsBulkWithOutbox(ctx, bridgeRels, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return NodeBulkCreateResponse{}, err
	}
	_, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		events := make([]OutboxEvent, 0, len(drafts)*2)
		for i, draft := range drafts {
			if _, err := s.finalizeGraphVersion(ctx, repo, draft.graphVersionID); err != nil {
				return err
			}
			if strings.EqualFold(s.ftsBackendKind, "postgres") {
				if err := repo.UpsertGraphProjectionHead(ctx, GraphProjectionHeadRecord{
					IdentifierID:         draft.graphIdentifierID,
					BackendKind:          "fts",
					BackendName:          "postgres",
					AppliedVersionID:     draft.graphVersionID,
					AppliedVersionNumber: draft.graphVersionNumber,
					UpdatedAt:            time.Now().UTC(),
				}); err != nil {
					return err
				}
			}
			drafts[i].response = NodeCreateResponse{
				NodeID:             draft.node.ID,
				GraphIdentifierID:  draft.graphIdentifierID,
				GraphVersionID:     draft.graphVersionID,
				GraphVersionNumber: draft.graphVersionNumber,
				ReferenceID:        strings.TrimSpace(draft.referenceID),
				DomainVersion:      draft.node.DomainVersion,
				Status:             "processing",
				SyncETAMs:          estimateSyncETA(s.store, draft.node.DomainID, s.syncEtaDefaultMs),
			}
			events = append(events, OutboxEvent{
				ID:            newID("evt"),
				AggregateType: "kg_node",
				AggregateID:   draft.node.ID,
				EventType:     "NODE_UPSERTED",
				Payload: map[string]any{
					"node_id":              draft.node.ID,
					"domain_id":            draft.node.DomainID,
					"owner_tenant_id":      draft.node.OwnerTenantID,
					"owner_app_id":         draft.node.OwnerAppID,
					"node_type":            draft.node.NodeType,
					"external_ref":         draft.node.ExternalRef,
					"graph_scope":          draft.graphScope,
					"graph_identifier_id":  draft.graphIdentifierID,
					"graph_version_id":     draft.graphVersionID,
					"graph_version_number": draft.graphVersionNumber,
					"reference_id":         strings.TrimSpace(draft.referenceID),
					"entity_ids":           append([]string{draft.node.ID}, relationshipIDs(draft.bridges)...),
					"tx_scope":             scope.Statements,
				},
				Status:     "PENDING",
				RetryCount: 0,
				CreatedAt:  draft.node.UpdatedAt,
			})
		}
		for _, rel := range bridgeRels {
			fromNode, _ := repo.GetNodeByID(rel.FromNodeID)
			graphScope := deriveGraphScopeForNode(fromNode)
			events = append(events, OutboxEvent{
				ID:            newID("evt"),
				AggregateType: "kg_relationship",
				AggregateID:   rel.ID,
				EventType:     "RELATIONSHIP_UPSERTED",
				Payload: map[string]any{
					"relationship_id": rel.ID,
					"domain_id":       rel.DomainID,
					"owner_tenant_id": rel.OwnerTenantID,
					"owner_app_id":    rel.OwnerAppID,
					"from_node_id":    rel.FromNodeID,
					"to_node_id":      rel.ToNodeID,
					"rel_type":        rel.RelType,
					"domain_version":  rel.DomainVersion,
					"graph_scope":     graphScope,
					"tx_scope":        scope.Statements,
				},
				Status:     "PENDING",
				RetryCount: 0,
				CreatedAt:  rel.CreatedAt,
			})
		}
		return repo.CreateOutboxEvents(ctx, events)
	})
	if err != nil {
		return NodeBulkCreateResponse{}, err
	}
	telemetry.RecordBulkWriteBatchSize(len(req.Nodes))
	telemetry.RecordBulkWritePartialFailure(len(req.Nodes), len(result.Failed))
	for _, draft := range drafts {
		result.Succeeded = append(result.Succeeded, draft.response)
	}
	log.Printf(
		"write create_nodes_bulk ok tenant=%s app=%s requested=%d succeeded=%d failed=%d graph_version_id=%s mode=regular",
		actor.TenantID,
		actor.AppID,
		len(req.Nodes),
		len(result.Succeeded),
		len(result.Failed),
		strings.TrimSpace(req.GraphVersionID),
	)
	return result, nil
}

func (s *Service) UpdateNode(actor access.Identity, nodeID string, req NodeUpdateRequest) (NodeUpdateResponse, error) {
	return s.UpdateNodeWithContext(context.Background(), actor, nodeID, req)
}

func (s *Service) UpdateNodeWithContext(ctx context.Context, actor access.Identity, nodeID string, req NodeUpdateRequest) (NodeUpdateResponse, error) {
	if err := validateNodeUpdateRequest(req); err != nil {
		return NodeUpdateResponse{}, errors.Join(ErrValidation, err)
	}
	if strings.TrimSpace(req.GraphVersionID) != "" {
		var resp NodeUpdateResponse
		_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
			TenantID: actor.TenantID,
			AppID:    actor.AppID,
		}, func(scope session.SessionScope) error {
			repo := s.repositoryForScope(scope)
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, req.GraphVersionID)
			if err != nil {
				return err
			}
			node, ok := repo.GetNodeByID(nodeID)
			if !ok || node.IsDeleted {
				return ErrNotFound
			}
			domain, err := s.ontology.GetVisibleDomain(actor, node.DomainID)
			if err != nil {
				if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
					return ErrForbidden
				}
				return err
			}
			if err := s.ensureNodeMutationPermission(actor, node, domain); err != nil {
				return err
			}
			schema, err := s.ontology.GetNodeType(node.DomainID, node.NodeType)
			if err != nil {
				if errors.Is(err, ontology.ErrNotFound) {
					return errors.Join(ErrValidation, fmt.Errorf("unknown node_type: %s", node.NodeType))
				}
				return err
			}
			updated := node
			mergedProperties := cloneProperties(node.Properties)
			for key, value := range req.Properties {
				mergedProperties[key] = value
			}
			if err := validateProperties(schema, mergedProperties); err != nil {
				return errors.Join(ErrValidation, err)
			}
			statusValue := node.StatusValue
			if statusCfg, err := s.ontology.GetStatusFieldConfig(node.DomainID); err == nil && statusCfg != nil && statusCfg.StatusFieldName != "" {
				if value, ok := mergedProperties[statusCfg.StatusFieldName]; ok {
					statusValue = fmt.Sprintf("%v", value)
				} else {
					statusValue = ""
				}
			}
			updated.Properties = mergedProperties
			updated.Visibility = fallback(req.Visibility, node.Visibility)
			if len(updated.ACLVisibleTo) == 0 {
				updated.ACLVisibleTo = []string{node.OwnerTenantID + ":" + node.OwnerAppID}
			}
			if strings.TrimSpace(req.ExternalRef) != "" || node.ExternalRef == "" {
				updated.ExternalRef = strings.TrimSpace(req.ExternalRef)
			}
			version, err := s.ontology.GetCurrentVersion(node.DomainID)
			if err != nil {
				return err
			}
			updated.DomainVersion = version.Version
			updated.StatusValue = statusValue
			updated.UpdatedAt = s.now()
			if graphScope := deriveGraphScopeForNode(updated); graphScope != state.identity.GraphScope {
				return errors.Join(ErrValidation, fmt.Errorf("node graph scope mismatch: %s != %s", graphScope, state.identity.GraphScope))
			}
			if err := repo.UpdateNode(ctx, updated); err != nil {
				switch {
				case errors.Is(err, ErrDuplicateExternalRef):
					return errors.Join(ErrValidation, errors.New("external_ref already exists"))
				case errors.Is(err, ErrNodeNotFound):
					return ErrNotFound
				default:
					return err
				}
			}
			if err := repo.AddGraphVersionEntities(ctx, state.version.VersionID, graphVersionEntities("node", "UPSERT", nodeID)); err != nil {
				return err
			}
			resp = NodeUpdateResponse{
				NodeID:        nodeID,
				DomainVersion: version.Version,
				Status:        "processing",
			}
			return nil
		})
		if err != nil {
			return NodeUpdateResponse{}, err
		}
		return resp, nil
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
		repo := s.repositoryForScope(scope)
		graphScope := deriveGraphScopeForNode(updated)
		identity, graphVersion, err := s.sealGraphVersion(ctx, repo, actor, graphScope, req.ReferenceID, "node update", graphVersionEntities("node", "UPSERT", nodeID))
		if err != nil {
			return err
		}
		event := OutboxEvent{
			ID:            newID("evt"),
			AggregateType: "kg_node",
			AggregateID:   nodeID,
			EventType:     "NODE_UPSERTED",
			Payload: map[string]any{
				"node_id":              nodeID,
				"domain_id":            updated.DomainID,
				"owner_tenant_id":      updated.OwnerTenantID,
				"owner_app_id":         updated.OwnerAppID,
				"node_type":            updated.NodeType,
				"external_ref":         updated.ExternalRef,
				"graph_scope":          graphScope,
				"graph_identifier_id":  identity.IdentifierID,
				"graph_version_id":     graphVersion.VersionID,
				"graph_version_number": graphVersion.VersionNumber,
				"reference_id":         graphVersion.ReferenceID,
				"entity_ids":           []string{nodeID},
				"tx_scope":             scope.Statements,
			},
			Status:     "PENDING",
			RetryCount: 0,
			CreatedAt:  updated.UpdatedAt,
		}
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
	return s.DeleteNodeWithVersion(ctx, actor, nodeID, "")
}

func (s *Service) DeleteNodeWithVersion(ctx context.Context, actor access.Identity, nodeID, graphVersionID string) (NodeDeleteResponse, error) {
	if strings.TrimSpace(graphVersionID) != "" {
		var resp NodeDeleteResponse
		_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
			TenantID: actor.TenantID,
			AppID:    actor.AppID,
		}, func(scope session.SessionScope) error {
			repo := s.repositoryForScope(scope)
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, graphVersionID)
			if err != nil {
				return err
			}
			node, ok := repo.GetNodeByID(nodeID)
			if !ok || node.IsDeleted {
				return ErrNotFound
			}
			domain, err := s.ontology.GetVisibleDomain(actor, node.DomainID)
			if err != nil {
				if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
					return ErrForbidden
				}
				return err
			}
			if err := s.ensureNodeMutationPermission(actor, node, domain); err != nil {
				return err
			}
			deleted := node
			deleted.IsDeleted = true
			deleted.UpdatedAt = s.now()
			if graphScope := deriveGraphScopeForNode(deleted); graphScope != state.identity.GraphScope {
				return errors.Join(ErrValidation, fmt.Errorf("node graph scope mismatch: %s != %s", graphScope, state.identity.GraphScope))
			}
			if err := repo.SoftDeleteNode(ctx, deleted); err != nil {
				if errors.Is(err, ErrNodeNotFound) {
					return ErrNotFound
				}
				return err
			}
			if err := repo.AddGraphVersionEntities(ctx, state.version.VersionID, graphVersionEntities("node", "DELETE", nodeID)); err != nil {
				return err
			}
			resp = NodeDeleteResponse{
				NodeID:    nodeID,
				IsDeleted: true,
			}
			return nil
		})
		if err != nil {
			return NodeDeleteResponse{}, err
		}
		return resp, nil
	}
	var resp NodeDeleteResponse
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		created, err := s.deleteNodeInScope(ctx, scope, actor, nodeID)
		if err != nil {
			return err
		}
		resp = created
		return nil
	})
	if err != nil {
		return NodeDeleteResponse{}, err
	}
	return resp, nil
}

func (s *Service) CreateRelationship(actor access.Identity, req RelationshipCreateRequest) (RelationshipCreateResponse, error) {
	return s.CreateRelationshipWithContext(context.Background(), actor, req)
}

func (s *Service) CreateRelationshipWithContext(ctx context.Context, actor access.Identity, req RelationshipCreateRequest) (RelationshipCreateResponse, error) {
	var resp RelationshipCreateResponse
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		created, err := s.createRelationshipInScope(ctx, scope, actor, req)
		if err != nil {
			return err
		}
		resp = created
		return nil
	})
	if err != nil {
		return RelationshipCreateResponse{}, err
	}
	return resp, nil
}

func (s *Service) CreateRelationshipsBulkWithContext(ctx context.Context, actor access.Identity, req RelationshipBulkCreateRequest) (RelationshipBulkCreateResponse, error) {
	if len(req.Relationships) == 0 {
		return RelationshipBulkCreateResponse{Succeeded: []RelationshipCreateResponse{}, Failed: []BulkItemError{}}, nil
	}
	log.Printf("write create_relationships_bulk start tenant=%s app=%s requested=%d graph_version_id=%s", actor.TenantID, actor.AppID, len(req.Relationships), strings.TrimSpace(req.GraphVersionID))
	type bulkRelDraft struct {
		rel         RelationshipRecord
		referenceID string
		response    RelationshipCreateResponse
	}
	drafts := make([]bulkRelDraft, 0, len(req.Relationships))
	result := RelationshipBulkCreateResponse{Succeeded: []RelationshipCreateResponse{}, Failed: []BulkItemError{}}
	for idx, item := range req.Relationships {
		rel, err := s.previewRelationshipCreate(actor, item, s.store)
		if err != nil {
			result.Failed = append(result.Failed, BulkItemError{Index: idx, Error: err.Error()})
			continue
		}
		drafts = append(drafts, bulkRelDraft{
			rel:         rel,
			referenceID: strings.TrimSpace(item.ReferenceID),
			response: RelationshipCreateResponse{
				RelationshipID: rel.ID,
				Status:         "processing",
			},
		})
	}
	if len(drafts) == 0 {
		log.Printf(
			"write create_relationships_bulk rejected tenant=%s app=%s requested=%d failed=%d graph_version_id=%s",
			actor.TenantID,
			actor.AppID,
			len(req.Relationships),
			len(result.Failed),
			strings.TrimSpace(req.GraphVersionID),
		)
		return result, nil
	}
	if strings.TrimSpace(req.GraphVersionID) != "" {
		_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
			TenantID: actor.TenantID,
			AppID:    actor.AppID,
		}, func(scope session.SessionScope) error {
			repo := s.repositoryForScope(scope)
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, req.GraphVersionID)
			if err != nil {
				return err
			}
			rels := make([]RelationshipRecord, 0, len(drafts))
			entities := make([]GraphVersionEntityRecord, 0, len(drafts))
			for i, draft := range drafts {
				fromNode, _ := repo.GetNodeByID(draft.rel.FromNodeID)
				if graphScope := deriveGraphScopeForNode(fromNode); graphScope != state.identity.GraphScope {
					return errors.Join(ErrValidation, fmt.Errorf("relationship graph scope mismatch: %s != %s", graphScope, state.identity.GraphScope))
				}
				drafts[i].response = RelationshipCreateResponse{
					RelationshipID:     draft.rel.ID,
					GraphIdentifierID:  state.identity.IdentifierID,
					GraphVersionID:     state.version.VersionID,
					GraphVersionNumber: state.version.VersionNumber,
					ReferenceID:        draft.referenceID,
					Status:             "processing",
				}
				rels = append(rels, draft.rel)
				entities = append(entities, graphVersionEntities("relationship", "UPSERT", draft.rel.ID)...)
			}
			if err := repo.CreateRelationshipsBulkWithOutbox(ctx, rels, nil); err != nil {
				return err
			}
			return repo.AddGraphVersionEntities(ctx, state.version.VersionID, entities)
		})
		if err != nil {
			return RelationshipBulkCreateResponse{}, err
		}
		telemetry.RecordBulkWriteBatchSize(len(req.Relationships))
		telemetry.RecordBulkWritePartialFailure(len(req.Relationships), len(result.Failed))
		for _, draft := range drafts {
			result.Succeeded = append(result.Succeeded, draft.response)
		}
		log.Printf(
			"write create_relationships_bulk ok tenant=%s app=%s requested=%d succeeded=%d failed=%d graph_version_id=%s mode=session",
			actor.TenantID,
			actor.AppID,
			len(req.Relationships),
			len(result.Succeeded),
			len(result.Failed),
			strings.TrimSpace(req.GraphVersionID),
		)
		return result, nil
	}
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		rels := make([]RelationshipRecord, 0, len(drafts))
		versions := make([]struct {
			graphIdentifierID  string
			graphVersionID     string
			graphVersionNumber int64
			graphScope         string
			referenceID        string
		}, 0, len(drafts))
		for i, draft := range drafts {
			fromNode, _ := repo.GetNodeByID(draft.rel.FromNodeID)
			graphScope := deriveGraphScopeForNode(fromNode)
			identity, graphVersion, err := s.sealGraphVersionWithStatus(ctx, repo, actor, graphScope, draft.referenceID, "relationship upsert", "ONLINE", "PENDING_ENTITIES", graphVersionEntities("relationship", "UPSERT", draft.rel.ID))
			if err != nil {
				return err
			}
			drafts[i].referenceID = graphVersion.ReferenceID
			drafts[i].response = RelationshipCreateResponse{
				RelationshipID:     draft.rel.ID,
				GraphIdentifierID:  identity.IdentifierID,
				GraphVersionID:     graphVersion.VersionID,
				GraphVersionNumber: graphVersion.VersionNumber,
				ReferenceID:        graphVersion.ReferenceID,
				Status:             "processing",
			}
			versions = append(versions, struct {
				graphIdentifierID  string
				graphVersionID     string
				graphVersionNumber int64
				graphScope         string
				referenceID        string
			}{
				graphIdentifierID:  identity.IdentifierID,
				graphVersionID:     graphVersion.VersionID,
				graphVersionNumber: graphVersion.VersionNumber,
				graphScope:         graphScope,
				referenceID:        graphVersion.ReferenceID,
			})
			rels = append(rels, draft.rel)
		}
		if err := repo.CreateRelationshipsBulkWithOutbox(ctx, rels, nil); err != nil {
			return err
		}
		events := make([]OutboxEvent, 0, len(drafts))
		for i, draft := range drafts {
			version := versions[i]
			if _, err := s.finalizeGraphVersion(ctx, repo, version.graphVersionID); err != nil {
				return err
			}
			if strings.EqualFold(s.ftsBackendKind, "postgres") {
				if err := repo.UpsertGraphProjectionHead(ctx, GraphProjectionHeadRecord{
					IdentifierID:         version.graphIdentifierID,
					BackendKind:          "fts",
					BackendName:          "postgres",
					AppliedVersionID:     version.graphVersionID,
					AppliedVersionNumber: version.graphVersionNumber,
					UpdatedAt:            time.Now().UTC(),
				}); err != nil {
					return err
				}
			}
			drafts[i].response = RelationshipCreateResponse{
				RelationshipID:     draft.rel.ID,
				GraphIdentifierID:  version.graphIdentifierID,
				GraphVersionID:     version.graphVersionID,
				GraphVersionNumber: version.graphVersionNumber,
				ReferenceID:        strings.TrimSpace(version.referenceID),
				Status:             "processing",
			}
			events = append(events, OutboxEvent{
				ID:            newID("evt"),
				AggregateType: "kg_relationship",
				AggregateID:   draft.rel.ID,
				EventType:     "RELATIONSHIP_UPSERTED",
				Payload: map[string]any{
					"relationship_id":      draft.rel.ID,
					"domain_id":            draft.rel.DomainID,
					"owner_tenant_id":      draft.rel.OwnerTenantID,
					"owner_app_id":         draft.rel.OwnerAppID,
					"from_node_id":         draft.rel.FromNodeID,
					"to_node_id":           draft.rel.ToNodeID,
					"rel_type":             draft.rel.RelType,
					"domain_version":       draft.rel.DomainVersion,
					"graph_scope":          version.graphScope,
					"graph_identifier_id":  version.graphIdentifierID,
					"graph_version_id":     version.graphVersionID,
					"graph_version_number": version.graphVersionNumber,
					"reference_id":         strings.TrimSpace(version.referenceID),
					"entity_ids":           []string{draft.rel.ID},
					"tx_scope":             scope.Statements,
				},
				Status:     "PENDING",
				RetryCount: 0,
				CreatedAt:  draft.rel.CreatedAt,
			})
		}
		return repo.CreateOutboxEvents(ctx, events)
	})
	if err != nil {
		return RelationshipBulkCreateResponse{}, err
	}
	telemetry.RecordBulkWriteBatchSize(len(req.Relationships))
	telemetry.RecordBulkWritePartialFailure(len(req.Relationships), len(result.Failed))
	for _, draft := range drafts {
		result.Succeeded = append(result.Succeeded, draft.response)
	}
	log.Printf(
		"write create_relationships_bulk ok tenant=%s app=%s requested=%d succeeded=%d failed=%d graph_version_id=%s mode=regular",
		actor.TenantID,
		actor.AppID,
		len(req.Relationships),
		len(result.Succeeded),
		len(result.Failed),
		strings.TrimSpace(req.GraphVersionID),
	)
	return result, nil
}

func (s *Service) DeleteRelationshipsBulkWithContext(ctx context.Context, actor access.Identity, req RelationshipBulkDeleteRequest) (RelationshipBulkDeleteResponse, error) {
	ids := make([]string, 0, len(req.RelationshipIDs))
	seen := map[string]struct{}{}
	for _, id := range req.RelationshipIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return RelationshipBulkDeleteResponse{RelationshipIDs: []string{}, Count: 0}, nil
	}
	log.Printf("write delete_relationships_bulk start tenant=%s app=%s requested=%d graph_version_id=%s", actor.TenantID, actor.AppID, len(ids), strings.TrimSpace(req.GraphVersionID))
	var deletedIDs []string
	if strings.TrimSpace(req.GraphVersionID) != "" {
		_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
			TenantID: actor.TenantID,
			AppID:    actor.AppID,
		}, func(scope session.SessionScope) error {
			repo := s.repositoryForScope(scope)
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, req.GraphVersionID)
			if err != nil {
				return err
			}
			deletedAt := s.now()
			rels, err := repo.SoftDeleteRelationshipsWithOutbox(ctx, ids, deletedAt)
			if err != nil {
				return err
			}
			if len(rels) == 0 {
				return nil
			}
			entities := make([]GraphVersionEntityRecord, 0, len(rels))
			for _, rel := range rels {
				fromNode, _ := repo.GetNodeByID(rel.FromNodeID)
				if graphScope := deriveGraphScopeForNode(fromNode); graphScope != state.identity.GraphScope {
					return errors.Join(ErrValidation, fmt.Errorf("relationship graph scope mismatch: %s != %s", graphScope, state.identity.GraphScope))
				}
				deletedIDs = append(deletedIDs, rel.ID)
				entities = append(entities, GraphVersionEntityRecord{
					VersionID:  state.version.VersionID,
					EntityKind: "relationship",
					EntityID:   rel.ID,
					ChangeKind: "DELETE",
				})
			}
			return repo.AddGraphVersionEntities(ctx, state.version.VersionID, entities)
		})
		if err != nil {
			return RelationshipBulkDeleteResponse{}, err
		}
		telemetry.RecordBulkWriteBatchSize(len(ids))
		telemetry.RecordBulkWritePartialFailure(len(ids), len(ids)-len(deletedIDs))
		log.Printf(
			"write delete_relationships_bulk ok tenant=%s app=%s requested=%d deleted=%d graph_version_id=%s mode=session",
			actor.TenantID,
			actor.AppID,
			len(ids),
			len(deletedIDs),
			strings.TrimSpace(req.GraphVersionID),
		)
		return RelationshipBulkDeleteResponse{RelationshipIDs: deletedIDs, Count: len(deletedIDs)}, nil
	}
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		deletedAt := s.now()
		rels, err := repo.SoftDeleteRelationshipsWithOutbox(ctx, ids, deletedAt)
		if err != nil {
			return err
		}
		if len(rels) == 0 {
			return nil
		}
		type deletedRelVersion struct {
			graphIdentifierID  string
			graphVersionID     string
			graphVersionNumber int64
			graphScope         string
			referenceID        string
		}
		versions := make([]deletedRelVersion, 0, len(rels))
		for _, rel := range rels {
			deletedIDs = append(deletedIDs, rel.ID)
			fromNode, _ := repo.GetNodeByID(rel.FromNodeID)
			graphScope := deriveGraphScopeForNode(fromNode)
			identity, graphVersion, err := s.sealGraphVersionWithStatus(ctx, repo, actor, graphScope, "", "relationship delete", "ONLINE", "PENDING_ENTITIES", graphVersionEntities("relationship", "DELETE", rel.ID))
			if err != nil {
				return err
			}
			versions = append(versions, deletedRelVersion{
				graphIdentifierID:  identity.IdentifierID,
				graphVersionID:     graphVersion.VersionID,
				graphVersionNumber: graphVersion.VersionNumber,
				graphScope:         graphScope,
				referenceID:        graphVersion.ReferenceID,
			})
		}
		events := make([]OutboxEvent, 0, len(rels))
		for i, rel := range rels {
			version := versions[i]
			if _, err := s.finalizeGraphVersion(ctx, repo, version.graphVersionID); err != nil {
				return err
			}
			if strings.EqualFold(s.ftsBackendKind, "postgres") {
				if err := repo.UpsertGraphProjectionHead(ctx, GraphProjectionHeadRecord{
					IdentifierID:         version.graphIdentifierID,
					BackendKind:          "fts",
					BackendName:          "postgres",
					AppliedVersionID:     version.graphVersionID,
					AppliedVersionNumber: version.graphVersionNumber,
					UpdatedAt:            time.Now().UTC(),
				}); err != nil {
					return err
				}
			}
			events = append(events, OutboxEvent{
				ID:            newID("evt"),
				AggregateType: "kg_relationship",
				AggregateID:   rel.ID,
				EventType:     "RELATIONSHIP_DELETED",
				Payload: map[string]any{
					"relationship_id":      rel.ID,
					"domain_id":            rel.DomainID,
					"owner_tenant_id":      rel.OwnerTenantID,
					"owner_app_id":         rel.OwnerAppID,
					"from_node_id":         rel.FromNodeID,
					"to_node_id":           rel.ToNodeID,
					"rel_type":             rel.RelType,
					"graph_scope":          version.graphScope,
					"graph_identifier_id":  version.graphIdentifierID,
					"graph_version_id":     version.graphVersionID,
					"graph_version_number": version.graphVersionNumber,
					"reference_id":         strings.TrimSpace(version.referenceID),
					"entity_ids":           []string{rel.ID},
					"tx_scope":             scope.Statements,
				},
				Status:     "PENDING",
				RetryCount: 0,
				CreatedAt:  deletedAt,
			})
		}
		return repo.CreateOutboxEvents(ctx, events)
	})
	if err != nil {
		return RelationshipBulkDeleteResponse{}, err
	}
	telemetry.RecordBulkWriteBatchSize(len(ids))
	telemetry.RecordBulkWritePartialFailure(len(ids), len(ids)-len(deletedIDs))
	log.Printf(
		"write delete_relationships_bulk ok tenant=%s app=%s requested=%d deleted=%d graph_version_id=%s mode=regular",
		actor.TenantID,
		actor.AppID,
		len(ids),
		len(deletedIDs),
		strings.TrimSpace(req.GraphVersionID),
	)
	return RelationshipBulkDeleteResponse{RelationshipIDs: deletedIDs, Count: len(deletedIDs)}, nil
}

func (s *Service) DeleteNodesByExternalRefPrefixWithContext(ctx context.Context, actor access.Identity, req NodeDeleteByExternalRefPrefixRequest) (NodeDeleteByExternalRefPrefixResponse, error) {
	return s.DeleteNodesByExternalRefPrefixWithVersion(ctx, actor, req, "")
}

func (s *Service) DeleteNodesByExternalRefPrefixWithVersion(ctx context.Context, actor access.Identity, req NodeDeleteByExternalRefPrefixRequest, graphVersionID string) (NodeDeleteByExternalRefPrefixResponse, error) {
	prefix := strings.TrimSpace(req.ExternalRefPrefix)
	if prefix == "" {
		return NodeDeleteByExternalRefPrefixResponse{}, errors.Join(ErrValidation, errors.New("external_ref_prefix is required"))
	}
	log.Printf("write delete_nodes_by_external_ref_prefix start tenant=%s app=%s prefix=%s graph_version_id=%s", actor.TenantID, actor.AppID, prefix, strings.TrimSpace(graphVersionID))
	if strings.TrimSpace(graphVersionID) != "" {
		var deletedIDs []string
		_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
			TenantID: actor.TenantID,
			AppID:    actor.AppID,
		}, func(scope session.SessionScope) error {
			repo := s.repositoryForScope(scope)
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, graphVersionID)
			if err != nil {
				return err
			}
			deletedAt := s.now()
			nodes, err := repo.SoftDeleteNodesByExternalRefPrefix(ctx, prefix, deletedAt)
			if err != nil {
				return err
			}
			if len(nodes) == 0 {
				return nil
			}
			sort.Slice(nodes, func(i, j int) bool {
				if nodes[i].ExternalRef == nodes[j].ExternalRef {
					return nodes[i].ID < nodes[j].ID
				}
				return nodes[i].ExternalRef < nodes[j].ExternalRef
			})
			entities := make([]GraphVersionEntityRecord, 0, len(nodes))
			for _, node := range nodes {
				if graphScope := deriveGraphScopeForNode(node); graphScope != state.identity.GraphScope {
					return errors.Join(ErrValidation, fmt.Errorf("node graph scope mismatch: %s != %s", graphScope, state.identity.GraphScope))
				}
				deletedIDs = append(deletedIDs, node.ID)
				entities = append(entities, GraphVersionEntityRecord{
					VersionID:  state.version.VersionID,
					EntityKind: "node",
					EntityID:   node.ID,
					ChangeKind: "DELETE",
				})
			}
			return repo.AddGraphVersionEntities(ctx, state.version.VersionID, entities)
		})
		if err != nil {
			return NodeDeleteByExternalRefPrefixResponse{}, err
		}
		log.Printf(
			"write delete_nodes_by_external_ref_prefix ok tenant=%s app=%s prefix=%s deleted=%d graph_version_id=%s mode=session",
			actor.TenantID,
			actor.AppID,
			prefix,
			len(deletedIDs),
			strings.TrimSpace(graphVersionID),
		)
		return NodeDeleteByExternalRefPrefixResponse{NodeIDs: deletedIDs, Count: len(deletedIDs)}, nil
	}
	var deletedIDs []string
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		deletedAt := s.now()
		nodes, err := repo.SoftDeleteNodesByExternalRefPrefixWithOutbox(ctx, prefix, deletedAt)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			return nil
		}
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].ExternalRef == nodes[j].ExternalRef {
				return nodes[i].ID < nodes[j].ID
			}
			return nodes[i].ExternalRef < nodes[j].ExternalRef
		})
		type deletedNodeVersion struct {
			graphIdentifierID  string
			graphVersionID     string
			graphVersionNumber int64
			graphScope         string
			referenceID        string
		}
		versions := make([]deletedNodeVersion, 0, len(nodes))
		for _, node := range nodes {
			deletedIDs = append(deletedIDs, node.ID)
			graphScope := deriveGraphScopeForNode(node)
			identity, graphVersion, err := s.sealGraphVersionWithStatus(ctx, repo, actor, graphScope, "", "node delete", "ONLINE", "PENDING_ENTITIES", graphVersionEntities("node", "DELETE", node.ID))
			if err != nil {
				return err
			}
			versions = append(versions, deletedNodeVersion{
				graphIdentifierID:  identity.IdentifierID,
				graphVersionID:     graphVersion.VersionID,
				graphVersionNumber: graphVersion.VersionNumber,
				graphScope:         graphScope,
				referenceID:        graphVersion.ReferenceID,
			})
		}
		events := make([]OutboxEvent, 0, len(nodes))
		for i, node := range nodes {
			version := versions[i]
			if _, err := s.finalizeGraphVersion(ctx, repo, version.graphVersionID); err != nil {
				return err
			}
			if strings.EqualFold(s.ftsBackendKind, "postgres") {
				if err := repo.UpsertGraphProjectionHead(ctx, GraphProjectionHeadRecord{
					IdentifierID:         version.graphIdentifierID,
					BackendKind:          "fts",
					BackendName:          "postgres",
					AppliedVersionID:     version.graphVersionID,
					AppliedVersionNumber: version.graphVersionNumber,
					UpdatedAt:            time.Now().UTC(),
				}); err != nil {
					return err
				}
			}
			events = append(events, OutboxEvent{
				ID:            newID("evt"),
				AggregateType: "kg_node",
				AggregateID:   node.ID,
				EventType:     "NODE_DELETED",
				Payload: map[string]any{
					"node_id":              node.ID,
					"domain_id":            node.DomainID,
					"owner_tenant_id":      node.OwnerTenantID,
					"owner_app_id":         node.OwnerAppID,
					"graph_scope":          version.graphScope,
					"graph_identifier_id":  version.graphIdentifierID,
					"graph_version_id":     version.graphVersionID,
					"graph_version_number": version.graphVersionNumber,
					"reference_id":         strings.TrimSpace(version.referenceID),
					"entity_ids":           []string{node.ID},
					"tx_scope":             scope.Statements,
				},
				Status:     "PENDING",
				RetryCount: 0,
				CreatedAt:  deletedAt,
			})
		}
		return repo.CreateOutboxEvents(ctx, events)
	})
	if err != nil {
		return NodeDeleteByExternalRefPrefixResponse{}, err
	}
	log.Printf(
		"write delete_nodes_by_external_ref_prefix ok tenant=%s app=%s prefix=%s deleted=%d graph_version_id=%s mode=regular",
		actor.TenantID,
		actor.AppID,
		prefix,
		len(deletedIDs),
		strings.TrimSpace(graphVersionID),
	)
	return NodeDeleteByExternalRefPrefixResponse{NodeIDs: deletedIDs, Count: len(deletedIDs)}, nil
}

type syncETAReader interface {
	ListProjectionVersions() []ProjectionVersionRecord
	GetNodeByID(id string) (NodeRecord, bool)
}

func estimateSyncETA(store syncETAReader, domainID string, defaultMs int) int {
	if store == nil || defaultMs <= 0 {
		return defaultMs
	}
	records := make([]ProjectionVersionRecord, 0, 30)
	for _, record := range store.ListProjectionVersions() {
		if record.EntityKind != "kg_node" || record.SourceVersion == 0 || record.LastGraphSyncedAt.IsZero() || record.SourceUpdatedAt.IsZero() {
			continue
		}
		node, ok := store.GetNodeByID(record.EntityID)
		if !ok || node.DomainID != domainID {
			continue
		}
		records = append(records, record)
	}
	if len(records) < 5 {
		return defaultMs
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].SourceUpdatedAt.After(records[j].SourceUpdatedAt)
	})
	if len(records) > 30 {
		records = records[:30]
	}
	lags := make([]float64, 0, len(records))
	for _, record := range records {
		lag := record.LastGraphSyncedAt.Sub(record.SourceUpdatedAt).Milliseconds()
		if lag > 0 {
			lags = append(lags, float64(lag))
		}
	}
	if len(lags) < 5 {
		return defaultMs
	}
	sort.Float64s(lags)
	mid := len(lags) / 2
	median := lags[mid]
	if len(lags)%2 == 0 {
		median = (lags[mid-1] + lags[mid]) / 2
	}
	return int(math.Ceil(median * 1.5))
}

func classifyReplicaLag(replicaVersion, sourceVersion int64, sourceEventID string, lastSyncedAt time.Time, events map[string]OutboxEvent, maxRetries int, lagToleranceWindow time.Duration) string {
	if replicaVersion == sourceVersion {
		return "SYNCED"
	}
	event, ok := events[sourceEventID]
	if ok {
		if event.RetryCount >= maxRetries {
			return "STUCK"
		}
		if time.Since(event.CreatedAt) > lagToleranceWindow {
			return "LAGGING"
		}
		return "IN_FLIGHT"
	}
	if lastSyncedAt.IsZero() {
		return "STUCK"
	}
	if time.Since(lastSyncedAt) <= lagToleranceWindow {
		return "IN_FLIGHT"
	}
	return "STUCK"
}

func (s *Service) recordWriteAudit(actor access.Identity, ownerTenantID, ownerAppID, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.RecordWriteAudit(actor, ownerTenantID, ownerAppID, action, resourceType, resourceID, outcome, reason, metadata)
}

type nodeReader interface {
	GetNodeByID(id string) (NodeRecord, bool)
	GetNodeByExternalRef(externalRef string) (NodeRecord, bool)
}

type stagedNodeLookup struct {
	base   nodeReader
	staged map[string]NodeRecord
}

func (l stagedNodeLookup) GetNodeByID(id string) (NodeRecord, bool) {
	if node, ok := l.staged[id]; ok {
		return node, true
	}
	return l.base.GetNodeByID(id)
}

func (l stagedNodeLookup) GetNodeByExternalRef(externalRef string) (NodeRecord, bool) {
	if l.base != nil {
		if node, ok := l.base.GetNodeByExternalRef(externalRef); ok {
			return node, true
		}
	}
	for _, node := range l.staged {
		if node.ExternalRef == externalRef {
			return node, true
		}
	}
	return NodeRecord{}, false
}

type bulkNodeLookup struct {
	base         nodeReader
	externalRefs map[string]NodeRecord
	staged       map[string]NodeRecord
}

func (l bulkNodeLookup) GetNodeByID(id string) (NodeRecord, bool) {
	if node, ok := l.staged[id]; ok {
		return node, true
	}
	if l.base != nil {
		return l.base.GetNodeByID(id)
	}
	return NodeRecord{}, false
}

func (l bulkNodeLookup) GetNodeByExternalRef(externalRef string) (NodeRecord, bool) {
	if node, ok := l.stagedByExternalRef(externalRef); ok {
		return node, true
	}
	if node, ok := l.externalRefs[externalRef]; ok {
		return node, true
	}
	if l.base != nil {
		return l.base.GetNodeByExternalRef(externalRef)
	}
	return NodeRecord{}, false
}

func (l bulkNodeLookup) stagedByExternalRef(externalRef string) (NodeRecord, bool) {
	for _, node := range l.staged {
		if node.ExternalRef == externalRef {
			return node, true
		}
	}
	return NodeRecord{}, false
}

func (s *Service) createNodeInScope(ctx context.Context, scope session.SessionScope, actor access.Identity, req NodeCreateRequest) (NodeCreateResponse, error) {
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
	graphScope := deriveGraphScope(req.DomainID, actor.TenantID, actor.AppID, req.Properties)

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
	repo := s.repositoryForScope(scope)
	externalRef := strings.TrimSpace(req.ExternalRef)
	if externalRef != "" {
		if existing, ok := repo.GetNodeByExternalRef(externalRef); ok {
			if err := s.ensureNodeMutationPermission(actor, existing, domain); err != nil {
				return NodeCreateResponse{}, err
			}
			updated := existing
			if existing.DomainID != req.DomainID {
				return NodeCreateResponse{}, errors.Join(ErrValidation, errors.New("external_ref already exists for a different domain"))
			}
			updated.NodeType = req.NodeType
			updated.Visibility = fallback(req.Visibility, existing.Visibility)
			updated.Properties = req.Properties
			updated.DomainVersion = version.Version
			updated.ExternalRef = externalRef
			updated.StatusValue = statusValue
			updated.IsDeleted = false
			updated.UpdatedAt = now
			identity, graphVersion, err := s.sealGraphVersion(ctx, repo, actor, graphScope, req.ReferenceID, "node upsert", mergeGraphVersionEntities(
				graphVersionEntities("node", "UPSERT", updated.ID),
			))
			if err != nil {
				return NodeCreateResponse{}, err
			}
			event := OutboxEvent{
				ID:            newID("evt"),
				AggregateType: "kg_node",
				AggregateID:   updated.ID,
				EventType:     "NODE_UPSERTED",
				Payload: map[string]any{
					"node_id":              updated.ID,
					"domain_id":            req.DomainID,
					"owner_tenant_id":      updated.OwnerTenantID,
					"owner_app_id":         updated.OwnerAppID,
					"node_type":            req.NodeType,
					"external_ref":         updated.ExternalRef,
					"graph_scope":          graphScope,
					"graph_identifier_id":  identity.IdentifierID,
					"graph_version_id":     graphVersion.VersionID,
					"graph_version_number": graphVersion.VersionNumber,
					"reference_id":         graphVersion.ReferenceID,
					"entity_ids":           []string{updated.ID},
					"tx_scope":             scope.Statements,
				},
				Status:     "PENDING",
				RetryCount: 0,
				CreatedAt:  now,
			}
			if err := repo.UpdateNodeWithOutbox(ctx, updated, event); err != nil {
				if errors.Is(err, ErrDuplicateExternalRef) {
					return NodeCreateResponse{}, errors.Join(ErrValidation, errors.New("external_ref already exists"))
				}
				if errors.Is(err, ErrNodeNotFound) {
					return NodeCreateResponse{}, ErrNotFound
				}
				return NodeCreateResponse{}, err
			}
			s.recordWriteAudit(actor, updated.OwnerTenantID, updated.OwnerAppID, "kg.node.create", "kg_node", updated.ID, "allow", "", map[string]any{
				"domain_id":    updated.DomainID,
				"node_type":    updated.NodeType,
				"external_ref": updated.ExternalRef,
				"graph_scope":  graphScope,
			})
			return NodeCreateResponse{
				NodeID:             updated.ID,
				GraphIdentifierID:  identity.IdentifierID,
				GraphVersionID:     graphVersion.VersionID,
				GraphVersionNumber: graphVersion.VersionNumber,
				ReferenceID:        graphVersion.ReferenceID,
				DomainVersion:      version.Version,
				Status:             "processing",
				SyncETAMs:          estimateSyncETA(s.store, req.DomainID, s.syncEtaDefaultMs),
			}, nil
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
	bridgeRelationships, err := s.buildBridgeRelationships(repo, req.DomainID, req.NodeType, actor, nodeID, req.Properties)
	if err != nil {
		return NodeCreateResponse{}, err
	}
	identity, graphVersion, err := s.sealGraphVersion(ctx, repo, actor, graphScope, req.ReferenceID, "node create", mergeGraphVersionEntities(
		graphVersionEntities("node", "UPSERT", nodeID),
		graphVersionEntities("embeddable_relationship", "UPSERT", relationshipIDs(bridgeRelationships)...),
	))
	if err != nil {
		return NodeCreateResponse{}, err
	}
	event := OutboxEvent{
		ID:            newID("evt"),
		AggregateType: "kg_node",
		AggregateID:   nodeID,
		EventType:     "NODE_UPSERTED",
		Payload: map[string]any{
			"node_id":              nodeID,
			"domain_id":            req.DomainID,
			"owner_tenant_id":      actor.TenantID,
			"owner_app_id":         actor.AppID,
			"node_type":            req.NodeType,
			"external_ref":         node.ExternalRef,
			"graph_scope":          graphScope,
			"graph_identifier_id":  identity.IdentifierID,
			"graph_version_id":     graphVersion.VersionID,
			"graph_version_number": graphVersion.VersionNumber,
			"reference_id":         graphVersion.ReferenceID,
			"entity_ids":           append([]string{nodeID}, relationshipIDs(bridgeRelationships)...),
			"tx_scope":             scope.Statements,
		},
		Status:     "PENDING",
		RetryCount: 0,
		CreatedAt:  now,
	}
	if err := repo.CreateNodeBundle(ctx, node, bridgeRelationships, event); err != nil {
		if errors.Is(err, ErrDuplicateExternalRef) {
			return NodeCreateResponse{}, errors.Join(ErrValidation, errors.New("external_ref already exists"))
		}
		return NodeCreateResponse{}, err
	}
	s.recordWriteAudit(actor, node.OwnerTenantID, node.OwnerAppID, "kg.node.create", "kg_node", node.ID, "allow", "", map[string]any{
		"domain_id":      node.DomainID,
		"node_type":      node.NodeType,
		"external_ref":   node.ExternalRef,
		"graph_scope":    graphScope,
		"bridge_rel_cnt": len(bridgeRelationships),
	})
	return NodeCreateResponse{
		NodeID:             nodeID,
		GraphIdentifierID:  identity.IdentifierID,
		GraphVersionID:     graphVersion.VersionID,
		GraphVersionNumber: graphVersion.VersionNumber,
		ReferenceID:        graphVersion.ReferenceID,
		DomainVersion:      version.Version,
		Status:             "processing",
		SyncETAMs:          estimateSyncETA(s.store, req.DomainID, s.syncEtaDefaultMs),
	}, nil
}

func (s *Service) preflightNodeBulkCreate(actor access.Identity, reqs []NodeCreateRequest) error {
	lookup := stagedNodeLookup{base: s.store, staged: map[string]NodeRecord{}}
	for _, req := range reqs {
		node, err := s.previewNodeCreate(actor, req, lookup)
		if err != nil {
			return err
		}
		if node.ExternalRef != "" {
			if existing, ok := lookup.GetNodeByExternalRef(node.ExternalRef); ok && existing.ID != node.ID {
				return errors.Join(ErrValidation, errors.New("external_ref already exists"))
			}
		}
		lookup.staged[node.ID] = node
	}
	return nil
}

func (s *Service) previewNodeCreate(actor access.Identity, req NodeCreateRequest, lookup nodeReader) (NodeRecord, error) {
	node, _, err := s.previewNodeCreateWithBridge(actor, req, lookup)
	return node, err
}

func (s *Service) previewNodeCreateWithBridge(actor access.Identity, req NodeCreateRequest, lookup nodeReader) (NodeRecord, []RelationshipRecord, error) {
	if err := validateNodeCreateRequest(req); err != nil {
		return NodeRecord{}, nil, errors.Join(ErrValidation, err)
	}

	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return NodeRecord{}, nil, ErrForbidden
		}
		return NodeRecord{}, nil, err
	}
	if err := s.ensureWritePermission(actor, domain, req.DomainID); err != nil {
		return NodeRecord{}, nil, err
	}

	schema, err := s.ontology.GetNodeType(req.DomainID, req.NodeType)
	if err != nil {
		if errors.Is(err, ontology.ErrNotFound) {
			return NodeRecord{}, nil, errors.Join(ErrValidation, fmt.Errorf("unknown node_type: %s", req.NodeType))
		}
		return NodeRecord{}, nil, err
	}
	if err := validateProperties(schema, req.Properties); err != nil {
		return NodeRecord{}, nil, errors.Join(ErrValidation, err)
	}

	version, err := s.ontology.GetCurrentVersion(req.DomainID)
	if err != nil {
		return NodeRecord{}, nil, err
	}
	statusCfg, err := s.ontology.GetStatusFieldConfig(req.DomainID)
	if err != nil {
		return NodeRecord{}, nil, err
	}
	_ = deriveGraphScope(req.DomainID, actor.TenantID, actor.AppID, req.Properties)

	now := s.now()
	statusValue := ""
	if statusCfg != nil && statusCfg.StatusFieldName != "" {
		if value, ok := req.Properties[statusCfg.StatusFieldName]; ok {
			statusValue = fmt.Sprintf("%v", value)
		}
	}
	nodeID := newID("node")
	externalRef := strings.TrimSpace(req.ExternalRef)
	if externalRef != "" {
		if existing, ok := lookup.GetNodeByExternalRef(externalRef); ok {
			nodeID = existing.ID
		}
	}
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
		ExternalRef:   externalRef,
		StatusValue:   statusValue,
		IsDeleted:     false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	bridgeRels, err := s.buildBridgeRelationships(lookup, req.DomainID, req.NodeType, actor, nodeID, req.Properties)
	if err != nil {
		return NodeRecord{}, nil, err
	}
	return node, bridgeRels, nil
}

func (s *Service) deleteNodeInScope(ctx context.Context, scope session.SessionScope, actor access.Identity, nodeID string) (NodeDeleteResponse, error) {
	repo := s.repositoryForScope(scope)
	node, ok := repo.GetNodeByID(nodeID)
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
			"graph_scope":     deriveGraphScopeForNode(deleted),
			"tx_scope":        scope.Statements,
		},
		Status:     "PENDING",
		RetryCount: 0,
		CreatedAt:  deleted.UpdatedAt,
	}
	identity, graphVersion, err := s.sealGraphVersion(ctx, repo, actor, deriveGraphScopeForNode(deleted), "", "node delete", graphVersionEntities("node", "DELETE", nodeID))
	if err != nil {
		return NodeDeleteResponse{}, err
	}
	event.Payload["graph_identifier_id"] = identity.IdentifierID
	event.Payload["graph_version_id"] = graphVersion.VersionID
	event.Payload["graph_version_number"] = graphVersion.VersionNumber
	event.Payload["reference_id"] = graphVersion.ReferenceID
	event.Payload["entity_ids"] = []string{nodeID}
	if err := repo.SoftDeleteNodeWithOutbox(ctx, deleted, event); err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			return NodeDeleteResponse{}, ErrNotFound
		}
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

func (s *Service) createRelationshipInScope(ctx context.Context, scope session.SessionScope, actor access.Identity, req RelationshipCreateRequest) (RelationshipCreateResponse, error) {
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

	repo := s.repositoryForScope(scope)
	fromNode, ok := repo.GetNodeByID(req.FromNodeID)
	if !ok || fromNode.IsDeleted {
		return RelationshipCreateResponse{}, ErrNotFound
	}
	toNode, ok := repo.GetNodeByID(req.ToNodeID)
	if !ok || toNode.IsDeleted {
		return RelationshipCreateResponse{}, ErrNotFound
	}
	if fromNode.DomainID != req.DomainID || toNode.DomainID != req.DomainID {
		return RelationshipCreateResponse{}, errors.Join(ErrValidation, errors.New("relationship endpoints must belong to the target domain"))
	}
	if err := ensureSameGraphScope(fromNode, toNode); err != nil {
		return RelationshipCreateResponse{}, errors.Join(ErrValidation, err)
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
	graphScope := deriveGraphScopeForNode(fromNode)
	identity, graphVersion, err := s.sealGraphVersion(ctx, repo, actor, graphScope, req.ReferenceID, "relationship upsert", graphVersionEntities("relationship", "UPSERT", relID))
	if err != nil {
		return RelationshipCreateResponse{}, err
	}
	event := OutboxEvent{
		ID:            newID("evt"),
		AggregateType: "kg_relationship",
		AggregateID:   relID,
		EventType:     "RELATIONSHIP_UPSERTED",
		Payload: map[string]any{
			"relationship_id":      relID,
			"domain_id":            req.DomainID,
			"owner_tenant_id":      actor.TenantID,
			"owner_app_id":         actor.AppID,
			"from_node_id":         req.FromNodeID,
			"to_node_id":           req.ToNodeID,
			"rel_type":             req.RelType,
			"domain_version":       version.Version,
			"graph_scope":          graphScope,
			"graph_identifier_id":  identity.IdentifierID,
			"graph_version_id":     graphVersion.VersionID,
			"graph_version_number": graphVersion.VersionNumber,
			"reference_id":         graphVersion.ReferenceID,
			"entity_ids":           []string{relID},
			"tx_scope":             scope.Statements,
		},
		Status:     "PENDING",
		RetryCount: 0,
		CreatedAt:  now,
	}
	if err := repo.CreateRelationshipWithOutbox(ctx, rel, event); err != nil {
		return RelationshipCreateResponse{}, err
	}
	s.recordWriteAudit(actor, rel.OwnerTenantID, rel.OwnerAppID, "kg.relationship.create", "kg_relationship", rel.ID, "allow", "", map[string]any{
		"domain_id":    rel.DomainID,
		"rel_type":     rel.RelType,
		"from_node_id": rel.FromNodeID,
		"to_node_id":   rel.ToNodeID,
		"graph_scope":  deriveGraphScopeForNode(fromNode),
	})
	return RelationshipCreateResponse{
		RelationshipID:     relID,
		GraphIdentifierID:  identity.IdentifierID,
		GraphVersionID:     graphVersion.VersionID,
		GraphVersionNumber: graphVersion.VersionNumber,
		ReferenceID:        graphVersion.ReferenceID,
		Status:             "processing",
	}, nil
}

func (s *Service) preflightRelationshipBulkCreate(actor access.Identity, reqs []RelationshipCreateRequest) error {
	for _, req := range reqs {
		if _, err := s.previewRelationshipCreate(actor, req, s.store); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) previewRelationshipCreate(actor access.Identity, req RelationshipCreateRequest, lookup nodeReader) (RelationshipRecord, error) {
	if err := validateRelationshipCreateRequest(req); err != nil {
		return RelationshipRecord{}, errors.Join(ErrValidation, err)
	}

	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return RelationshipRecord{}, ErrForbidden
		}
		return RelationshipRecord{}, err
	}
	if err := s.ensureWritePermission(actor, domain, req.DomainID); err != nil {
		return RelationshipRecord{}, err
	}

	fromNode, ok := lookup.GetNodeByID(req.FromNodeID)
	if !ok || fromNode.IsDeleted {
		return RelationshipRecord{}, ErrNotFound
	}
	toNode, ok := lookup.GetNodeByID(req.ToNodeID)
	if !ok || toNode.IsDeleted {
		return RelationshipRecord{}, ErrNotFound
	}
	if fromNode.DomainID != req.DomainID || toNode.DomainID != req.DomainID {
		return RelationshipRecord{}, errors.Join(ErrValidation, errors.New("relationship endpoints must belong to the target domain"))
	}
	if err := ensureSameGraphScope(fromNode, toNode); err != nil {
		return RelationshipRecord{}, errors.Join(ErrValidation, err)
	}

	schema, err := s.ontology.GetRelType(req.DomainID, req.RelType, fromNode.NodeType, toNode.NodeType)
	if err != nil {
		if errors.Is(err, ontology.ErrNotFound) {
			return RelationshipRecord{}, errors.Join(ErrValidation, fmt.Errorf("unknown relationship type or direction: %s", req.RelType))
		}
		return RelationshipRecord{}, err
	}
	if err := validateRelationshipProperties(schema, req.Properties); err != nil {
		return RelationshipRecord{}, errors.Join(ErrValidation, err)
	}

	version, err := s.ontology.GetCurrentVersion(req.DomainID)
	if err != nil {
		return RelationshipRecord{}, err
	}
	now := s.now()
	return RelationshipRecord{
		ID:            newID("rel"),
		RelType:       req.RelType,
		FromNodeID:    req.FromNodeID,
		ToNodeID:      req.ToNodeID,
		DomainID:      req.DomainID,
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		DomainVersion: version.Version,
		Properties:    req.Properties,
		CreatedAt:     now,
	}, nil
}

func canRunIngest(actor access.Identity) bool {
	switch actor.AppType {
	case "ingestion_producer", "admin_tool", "hybrid":
		return true
	default:
		return false
	}
}

func deriveGraphScope(domainID, ownerTenantID, ownerAppID string, properties map[string]any) string {
	if value := firstNonEmptyProperty(properties, "graph_id", "_kg_graph_scope"); value != "" {
		return value
	}
	if value := firstNonEmptyProperty(properties, "repo_id", "repository_id"); value != "" {
		return "repo:" + value
	}
	if value := firstNonEmptyProperty(properties, "project_id"); value != "" {
		return "project:" + value
	}
	return fmt.Sprintf("domain:%s:tenant:%s:app:%s", domainID, ownerTenantID, ownerAppID)
}

func deriveGraphScopeForNode(node NodeRecord) string {
	return deriveGraphScope(node.DomainID, node.OwnerTenantID, node.OwnerAppID, node.Properties)
}

func ensureSameGraphScope(fromNode, toNode NodeRecord) error {
	fromScope := deriveGraphScopeForNode(fromNode)
	toScope := deriveGraphScopeForNode(toNode)
	if fromScope == toScope {
		return nil
	}
	telemetry.RecordGraphScopeConflict(fromScope, toScope)
	return fmt.Errorf("relationship endpoints must belong to the same graph scope: %s != %s", fromScope, toScope)
}

func firstNonEmptyProperty(properties map[string]any, keys ...string) string {
	for _, key := range keys {
		if properties == nil {
			return ""
		}
		if value, ok := properties[key]; ok {
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", value))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
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

func (s *Service) buildBridgeRelationships(reader nodeReader, domainID, nodeType string, actor access.Identity, fromNodeID string, properties map[string]any) ([]RelationshipRecord, error) {
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
			targetNode, ok := reader.GetNodeByID(targetID)
			if !ok {
				targetNode, ok = reader.GetNodeByExternalRef(targetID)
			}
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
				ToNodeID:      targetNode.ID,
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
	return identity.NewUUID()
}

func deterministicUUID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	raw := make([]byte, 16)
	copy(raw, sum[:16])
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
