package workers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
	"kg-service/internal/searchprofile"
	"kg-service/internal/write"
)

type Repository interface {
	ListOutboxEvents() []write.OutboxEvent
	GetNodeByID(id string) (write.NodeRecord, bool)
	GetRelationshipByID(id string) (write.RelationshipRecord, bool)
	write.OutboxReader
	ListNodes() []write.NodeRecord
	ListRelationships() []write.RelationshipRecord
	UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error
	GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool)
	ListProjectionVersions() []write.ProjectionVersionRecord
}

type OntologyResolver interface {
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
}

type Runtime struct {
	store         Repository
	ontology      OntologyResolver
	cache         *rediscache.Client
	embedding     vector.EmbeddingRouter
	profiles      ontology.SearchProfileResolver
	graphAdapter  graphstore.GraphAdapter
	vectorAdapter vectorstore.VectorAdapter
	ftsAdapter    fts.FTSAdapter
	graph         *GraphStore
	vector        *VectorStore
	mu            sync.Mutex
	outbox        map[string]EventEnvelope
	maxRetries    int
}

type projectionVersionWriter interface {
	UpsertProjectionVersion(context.Context, write.ProjectionVersionRecord) error
}

func NewRuntime(store Repository, ontologyResolver OntologyResolver, cache *rediscache.Client) *Runtime {
	deterministic := vector.NewDeterministicProvider(8)
	router := vector.DirectRouter{Provider: deterministic}
	profileResolver, _ := ontologyResolver.(ontology.SearchProfileResolver)
	runtime := &Runtime{
		store:         store,
		ontology:      ontologyResolver,
		cache:         cache,
		embedding:     router,
		profiles:      profileResolver,
		vectorAdapter: vectorstore.NewInMemoryVectorAdapter(),
		ftsAdapter:    fts.NewInMemoryFTSAdapter(),
		graph:         &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:        &VectorStore{Documents: map[string]VectorDocument{}},
		outbox:        map[string]EventEnvelope{},
		maxRetries:    3,
	}
	runtime.graphAdapter = runtimeGraphAdapter{graph: runtime.graph}
	return runtime
}

func (r *Runtime) Graph() *GraphStore {
	return r.graph
}

func (r *Runtime) Vector() *VectorStore {
	return r.vector
}

func (r *Runtime) VectorAdapter() vectorstore.VectorAdapter {
	return r.vectorAdapter
}

func (r *Runtime) FTSAdapter() fts.FTSAdapter {
	return r.ftsAdapter
}

// SetEmbeddingRouter replaces the embedding router used when projecting nodes.
// Call before starting the worker.
func (r *Runtime) SetEmbeddingRouter(router vector.EmbeddingRouter) {
	if router == nil {
		return
	}
	r.embedding = router
}

// SetVectorAdapter replaces the vector adapter. Pass the same instance used by
// the search service so that written nodes are immediately queryable.
func (r *Runtime) SetVectorAdapter(adapter vectorstore.VectorAdapter) {
	if adapter == nil {
		return
	}
	r.vectorAdapter = adapter
}

// SetFTSAdapter replaces the full-text-search adapter. Pass the same instance
// used by the search service.
func (r *Runtime) SetFTSAdapter(adapter fts.FTSAdapter) {
	if adapter == nil {
		return
	}
	r.ftsAdapter = adapter
}

// SetGraphAdapter replaces the graph adapter used when projecting nodes and
// relationships.
func (r *Runtime) SetGraphAdapter(adapter graphstore.GraphAdapter) {
	if adapter == nil {
		return
	}
	r.graphAdapter = adapter
}

// SetSearchProfileResolver replaces the search profile resolver used to build
// domain-specific embedding texts.
func (r *Runtime) SetSearchProfileResolver(resolver ontology.SearchProfileResolver) {
	if resolver == nil {
		return
	}
	r.profiles = resolver
}

func (r *Runtime) PollOnce() WorkerReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	report := WorkerReport{}
	for _, event := range r.store.ListOutboxEvents() {
		env, ok := r.outbox[event.ID]
		if !ok {
			status := EventStatus(event.Status)
			if status == "" {
				status = EventPending
			}
			env = EventEnvelope{Event: event, Status: status, RetryCount: event.RetryCount}
		}
		if env.Status == EventDone || env.Status == EventDeadLetter {
			continue
		}

		env.Status = EventProcessing
		_ = r.store.UpdateOutboxStatus(context.Background(), event.ID, string(EventProcessing), env.RetryCount, nil)
		if err := r.handleEvent(env.Event); err != nil {
			env.RetryCount++
			env.Error = err.Error()
			if env.RetryCount >= r.maxRetries {
				env.Status = EventDeadLetter
				report.DeadLetter++
			} else {
				env.Status = EventFailed
				report.Failed++
			}
			_ = r.store.UpdateOutboxStatus(context.Background(), event.ID, string(env.Status), env.RetryCount, nil)
			r.outbox[event.ID] = env
			continue
		}
		env.Status = EventDone
		env.Error = ""
		now := time.Now().UTC()
		env.Event.ProcessedAt = &now
		_ = r.store.UpdateOutboxStatus(context.Background(), event.ID, string(EventDone), env.RetryCount, &now)
		r.outbox[event.ID] = env
		report.Processed++
	}
	return report
}

func (r *Runtime) EventEnvelope(id string) (EventEnvelope, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	env, ok := r.outbox[id]
	return env, ok
}

func (r *Runtime) Reconcile() ReconciliationReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	report := ReconciliationReport{}
	sourceNodes := map[string]write.NodeRecord{}
	for _, node := range r.store.ListNodes() {
		sourceNodes[node.ID] = node
	}
	sourceRelationships := map[string]write.RelationshipRecord{}
	for _, rel := range r.store.ListRelationships() {
		sourceRelationships[rel.ID] = rel
	}
	graphNodes := map[string]GraphNode{}
	graphRelationships := map[string]GraphRelationship{}
	if r.graphAdapter != nil {
		if nodes, err := r.graphAdapter.ListNodes(context.Background()); err == nil {
			for _, node := range nodes {
				graphNodes[node.ID] = GraphNode{
					ID:            node.ID,
					NodeType:      node.NodeType,
					DomainID:      node.DomainID,
					OwnerTenantID: node.OwnerTenantID,
					OwnerAppID:    node.OwnerAppID,
					ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
					Visibility:    node.Visibility,
					StatusValue:   node.StatusValue,
					IsDeleted:     node.IsDeleted,
					SyncVersion:   node.SyncVersion,
					Properties:    cloneMap(node.Properties),
					CreatedAt:     node.CreatedAt,
					UpdatedAt:     node.UpdatedAt,
				}
			}
		}
		if rels, err := r.graphAdapter.ListRelationships(context.Background()); err == nil {
			for _, rel := range rels {
				graphRelationships[rel.ID] = GraphRelationship{
					ID:          rel.ID,
					RelType:     rel.RelType,
					FromNodeID:  rel.FromNodeID,
					ToNodeID:    rel.ToNodeID,
					DomainID:    rel.DomainID,
					SyncVersion: rel.SyncVersion,
					Properties:  cloneMap(rel.Properties),
				}
			}
		}
	}
	vectorDocs := map[string]VectorDocument{}
	if r.vectorAdapter != nil {
		if docs, err := r.vectorAdapter.Snapshot(context.Background()); err == nil {
			for _, doc := range docs {
				vectorDocs[doc.NodeID] = VectorDocument{
					NodeID:         doc.NodeID,
					NodeType:       doc.NodeType,
					DomainID:       doc.DomainID,
					OwnerTenantID:  doc.OwnerTenantID,
					OwnerAppID:     doc.OwnerAppID,
					ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
					IsDeleted:      doc.IsDeleted,
					StatusValue:    doc.StatusValue,
					AuthorityScore: doc.AuthorityScore,
					SyncVersion:    doc.SyncVersion,
					DomainProps:    cloneMap(doc.DomainProps),
					Embedding:      append([]float64(nil), doc.Embedding...),
				}
			}
		}
	}

	for id, source := range sourceNodes {
		graphNode, ok := graphNodes[id]
		ledger, ledgerOk := r.store.GetProjectionVersion(id, "kg_node")
		if !ok {
			report.GraphDriftCount++
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "missing_projection", Details: "node missing from graph/vector projections"})
			continue
		}
		expectedVersion := int64(source.DomainVersion)
		if ledgerOk && ledger.SourceVersion > 0 {
			expectedVersion = ledger.SourceVersion
		}
		if expectedVersion != 0 && graphNode.SyncVersion != expectedVersion {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "stale_projection_version", Details: "graph node sync version does not match source version"})
		}
		if graphNode.IsDeleted != source.IsDeleted || graphNode.StatusValue != source.StatusValue || graphNode.NodeType != source.NodeType || graphNode.DomainID != source.DomainID {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_mismatch", Details: "graph projection does not match source node"})
		}
		if !reflect.DeepEqual(stripSyncVersionMetadata(graphNode.Properties), source.Properties) || !reflect.DeepEqual(graphNode.ACLVisibleTo, nodeACLVisibleTo(source)) {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_payload_mismatch", Details: "graph node payload does not match source node"})
		}
		if doc, ok := vectorDocs[id]; !ok || doc.IsDeleted != source.IsDeleted || doc.StatusValue != source.StatusValue || doc.NodeType != source.NodeType || doc.DomainID != source.DomainID {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_mismatch", Details: "vector projection does not match source node"})
		} else if !reflect.DeepEqual(stripSyncVersionMetadata(doc.DomainProps), source.Properties) || !reflect.DeepEqual(doc.ACLVisibleTo, nodeACLVisibleTo(source)) {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_payload_mismatch", Details: "vector document payload does not match source node"})
		} else if expectedVersion != 0 && doc.SyncVersion != expectedVersion {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "stale_projection_version", Details: "vector document sync version does not match source version"})
		}
	}
	for id := range graphNodes {
		if _, ok := sourceNodes[id]; !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_graph_node", Details: "node exists in graph but not source"})
		}
	}
	for id := range vectorDocs {
		if _, ok := sourceNodes[id]; !ok {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_vector_doc", Details: "node exists in vector but not source"})
		}
	}
	for id, sourceRel := range sourceRelationships {
		graphRel, ok := graphRelationships[id]
		ledger, ledgerOk := r.store.GetProjectionVersion(id, "kg_relationship")
		if !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "missing_relationship", Details: "relationship missing from graph projection"})
			continue
		}
		expectedVersion := int64(sourceRel.DomainVersion)
		if ledgerOk && ledger.SourceVersion > 0 {
			expectedVersion = ledger.SourceVersion
		}
		if expectedVersion != 0 && graphRel.SyncVersion != expectedVersion {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "stale_projection_version", Details: "graph relationship sync version does not match source version"})
		}
		if graphRel.RelType != sourceRel.RelType || graphRel.FromNodeID != sourceRel.FromNodeID || graphRel.ToNodeID != sourceRel.ToNodeID {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "relationship_mismatch", Details: "graph relationship differs from source"})
		}
		if !reflect.DeepEqual(stripSyncVersionMetadata(graphRel.Properties), sourceRel.Properties) {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "relationship_payload_mismatch", Details: "graph relationship payload does not match source"})
		}
	}
	for id := range graphRelationships {
		if _, ok := sourceRelationships[id]; !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_graph_relationship", Details: "relationship exists in graph but not source"})
		}
	}
	for _, record := range r.store.ListProjectionVersions() {
		switch record.EntityKind {
		case "kg_node":
			if _, ok := sourceNodes[record.EntityID]; !ok {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "orphan_projection_version", Details: "projection ledger contains node entry not present in source"})
				continue
			}
			source := sourceNodes[record.EntityID]
			if record.SourceVersion != int64(source.DomainVersion) {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "stale_projection_version", Details: "projection ledger source version differs from source node"})
			}
		case "kg_relationship":
			if _, ok := sourceRelationships[record.EntityID]; !ok {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "orphan_projection_version", Details: "projection ledger contains relationship entry not present in source"})
				continue
			}
			source := sourceRelationships[record.EntityID]
			if record.SourceVersion != int64(source.DomainVersion) {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "stale_projection_version", Details: "projection ledger source version differs from source relationship"})
			}
		}
	}

	if report.GraphDriftCount == 0 && report.VectorDriftCount == 0 && report.ProjectionVersionDriftCount == 0 {
		report.Overall = "pass"
	} else {
		report.Overall = "fail"
	}
	return report
}

func (r *Runtime) handleEvent(event write.OutboxEvent) error {
	switch event.EventType {
	case "NODE_UPSERTED":
		nodeID, _ := event.Payload["node_id"].(string)
		node, ok := r.store.GetNodeByID(nodeID)
		if !ok {
			return errors.New("node not found")
		}
		if err := r.projectNode(node); err != nil {
			return err
		}
		r.updateProjectionVersion(event, node.DomainVersion, node.ID, node.CreatedAt, int64(node.DomainVersion), int64(node.DomainVersion))
		return r.applyStatusCascade(node.DomainID, node.ID)
	case "NODE_DELETED":
		nodeID, _ := event.Payload["node_id"].(string)
		node, ok := r.store.GetNodeByID(nodeID)
		if !ok {
			return errors.New("node not found")
		}
		node.IsDeleted = true
		if r.graphAdapter != nil {
			_ = r.graphAdapter.DeleteNode(context.Background(), nodeID)
		}
		if r.vectorAdapter != nil {
			_ = r.vectorAdapter.Delete(context.Background(), nodeID)
		}
		if r.ftsAdapter != nil {
			_ = r.ftsAdapter.Delete(context.Background(), nodeID)
		}
		if err := r.projectNode(node); err != nil {
			return err
		}
		r.updateProjectionVersion(event, node.DomainVersion, node.ID, node.UpdatedAt, int64(node.DomainVersion), int64(node.DomainVersion))
		return nil
	case "RELATIONSHIP_UPSERTED", "RELATIONSHIP_DELETED":
		relID, _ := event.Payload["relationship_id"].(string)
		rel, ok := r.store.GetRelationshipByID(relID)
		if !ok {
			return errors.New("relationship not found")
		}
		sourceVersion := int64(rel.DomainVersion)
		relPayload := GraphRelationship{
			ID:          rel.ID,
			RelType:     rel.RelType,
			FromNodeID:  rel.FromNodeID,
			ToNodeID:    rel.ToNodeID,
			DomainID:    rel.DomainID,
			SyncVersion: sourceVersion,
			Properties:  cloneMap(rel.Properties),
		}
		if r.graphAdapter != nil {
			if event.EventType == "RELATIONSHIP_DELETED" {
				_ = r.graphAdapter.DeleteRelationship(context.Background(), relID)
			} else {
				_ = r.graphAdapter.UpsertRelationship(context.Background(), graphstore.GraphRelationship{
					ID:          relPayload.ID,
					RelType:     relPayload.RelType,
					FromNodeID:  relPayload.FromNodeID,
					ToNodeID:    relPayload.ToNodeID,
					DomainID:    relPayload.DomainID,
					SyncVersion: sourceVersion,
					Properties:  cloneMap(relPayload.Properties),
				})
			}
		}
		r.graph.Rels[rel.ID] = relPayload
		r.graph.Rels[rel.ID].Properties["_kg_sync_version"] = sourceVersion
		r.updateProjectionVersion(event, rel.DomainVersion, rel.ID, rel.CreatedAt, int64(rel.DomainVersion), 0)
		return r.applyStatusCascade(rel.DomainID, rel.FromNodeID)
	case "ACCESS_GRANT_CHANGED":
		return r.handleAccessGrantChanged(event.Payload)
	default:
		return nil
	}
}

func (r *Runtime) projectNode(node write.NodeRecord) error {
	acl := nodeACLVisibleTo(node)
	r.graph.Nodes[node.ID] = GraphNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), acl...),
		Visibility:    node.Visibility,
		StatusValue:   node.StatusValue,
		IsDeleted:     node.IsDeleted,
		SyncVersion:   int64(node.DomainVersion),
		Properties:    cloneMap(node.Properties),
	}
	r.graph.Nodes[node.ID].Properties["_kg_sync_version"] = int64(node.DomainVersion)
	if r.graphAdapter != nil {
		_ = r.graphAdapter.UpsertNode(context.Background(), graphstore.GraphNode{
			ID:            node.ID,
			NodeType:      node.NodeType,
			DomainID:      node.DomainID,
			OwnerTenantID: node.OwnerTenantID,
			OwnerAppID:    node.OwnerAppID,
			ACLVisibleTo:  append([]string(nil), acl...),
			Visibility:    node.Visibility,
			StatusValue:   node.StatusValue,
			IsDeleted:     node.IsDeleted,
			SyncVersion:   int64(node.DomainVersion),
			Properties:    cloneMap(node.Properties),
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		})
	}
	cfg, err := r.ontology.GetStatusFieldConfig(node.DomainID)
	if err != nil {
		return err
	}
	resolvedProfile := ontology.ResolvedSearchProfile{}
	if r.profiles != nil {
		if profile, err := r.profiles.Resolve(node.DomainID, node.OwnerTenantID, node.OwnerAppID); err == nil {
			resolvedProfile = profile
		}
	}
	text := searchprofile.BuildEmbeddingText(node, resolvedProfile)
	embeddingProvider := r.embedding.RouteContext(node.OwnerTenantID, node.DomainID)
	if embeddingProvider == nil {
		embeddingProvider = vector.NewDeterministicProvider(8)
	}
	embedding, err := embeddingProvider.Embed(context.Background(), text)
	if err != nil {
		return err
	}
	doc := VectorDocument{
		NodeID:        node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), acl...),
		IsDeleted:     node.IsDeleted,
		StatusValue:   node.StatusValue,
		SyncVersion:   int64(node.DomainVersion),
		DomainProps:   cloneMap(node.Properties),
		Embedding:     embedding,
	}
	if cfg != nil && cfg.AuthorityFieldName != "" && len(cfg.AuthorityValuesMap) > 0 {
		if raw, ok := node.Properties[cfg.AuthorityFieldName]; ok {
			if score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", raw)]; ok {
				doc.AuthorityScore = &score
			}
		}
	}
	if r.vectorAdapter != nil {
		if err := r.vectorAdapter.Upsert(context.Background(), vectorstore.VectorDocument{
			NodeID:         doc.NodeID,
			NodeType:       doc.NodeType,
			DomainID:       doc.DomainID,
			OwnerTenantID:  doc.OwnerTenantID,
			OwnerAppID:     doc.OwnerAppID,
			ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
			IsDeleted:      doc.IsDeleted,
			StatusValue:    doc.StatusValue,
			AuthorityScore: doc.AuthorityScore,
			SyncVersion:    doc.SyncVersion,
			DomainProps:    cloneMap(doc.DomainProps),
			Embedding:      append([]float64(nil), doc.Embedding...),
		}); err != nil {
			return err
		}
	}
	r.vector.Documents[node.ID] = doc
	r.vector.Documents[node.ID].DomainProps["_kg_sync_version"] = int64(node.DomainVersion)
	if !node.IsDeleted && r.ftsAdapter != nil {
		if err := r.ftsAdapter.Index(context.Background(), buildFTSDocument(node, acl, doc.AuthorityScore)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) applyStatusCascade(domainID, fromNodeID string) error {
	cfg, err := r.ontology.GetStatusFieldConfig(domainID)
	if err != nil || cfg == nil || len(cfg.CascadeRules) == 0 {
		return err
	}
	source, ok := r.graph.Nodes[fromNodeID]
	if !ok {
		return nil
	}
	for _, rule := range cfg.CascadeRules {
		for id, node := range r.graph.Nodes {
			if node.DomainID != domainID || node.NodeType != rule.ToNodeType {
				continue
			}
			if nodeHasIncomingRel(id, rule.ViaRel, fromNodeID, r.graph.Rels) {
				node.StatusValue = source.StatusValue
				r.graph.Nodes[id] = node
				if doc, ok := r.vector.Documents[id]; ok {
					doc.StatusValue = source.StatusValue
					r.vector.Documents[id] = doc
				}
			}
		}
	}
	return nil
}

func (r *Runtime) handleAccessGrantChanged(payload map[string]any) error {
	grantorTenantID, _ := payload["grantor_tenant_id"].(string)
	granteeTenantID, _ := payload["grantee_tenant_id"].(string)
	granteeAppID, _ := payload["grantee_app_id"].(string)
	scopeType, _ := payload["scope_type"].(string)
	scopeValue, _ := payload["scope_value"].(string)
	permission, _ := payload["permission"].(string)
	status, _ := payload["status"].(string)
	if grantorTenantID == "" || granteeTenantID == "" || granteeAppID == "" {
		return nil
	}
	token := granteeTenantID + ":" + granteeAppID
	for id, node := range r.graph.Nodes {
		if node.OwnerTenantID != grantorTenantID {
			continue
		}
		if scopeType == "domain" && scopeValue != "" && node.DomainID != scopeValue {
			continue
		}
		if status == "revoked" || status == "deleted" {
			node.ACLVisibleTo = removeString(node.ACLVisibleTo, token)
		} else {
			node.ACLVisibleTo = appendUnique(node.ACLVisibleTo, token)
		}
		r.graph.Nodes[id] = node
		if doc, ok := r.vector.Documents[id]; ok {
			if status == "revoked" || status == "deleted" {
				doc.ACLVisibleTo = removeString(doc.ACLVisibleTo, token)
			} else {
				doc.ACLVisibleTo = appendUnique(doc.ACLVisibleTo, token)
			}
			r.vector.Documents[id] = doc
		}
	}
	_ = permission
	if r.cache != nil {
		r.cache.Delete("acl:" + granteeTenantID + ":" + granteeAppID)
	}
	return nil
}

func (r *Runtime) updateProjectionVersion(event write.OutboxEvent, sourceVersion int, entityID string, sourceUpdatedAt time.Time, graphVersion, vectorVersion int64) {
	writer, ok := any(r.store).(projectionVersionWriter)
	if !ok || writer == nil || sourceVersion == 0 {
		return
	}
	record := write.ProjectionVersionRecord{
		EntityID:        entityID,
		EntityKind:      "kg_node",
		SourceVersion:   int64(sourceVersion),
		SourceEventID:   event.ID,
		SourceUpdatedAt: sourceUpdatedAt,
		GraphBackend:    "graph",
		GraphVersion:    graphVersion,
		VectorBackend:   "vector",
		VectorVersion:   vectorVersion,
	}
	if event.AggregateType == "kg_relationship" {
		record.EntityKind = "kg_relationship"
	}
	_ = writer.UpsertProjectionVersion(context.Background(), record)
}

func nodeACLVisibleTo(node write.NodeRecord) []string {
	if len(node.ACLVisibleTo) > 0 {
		return append([]string(nil), node.ACLVisibleTo...)
	}
	return []string{node.OwnerTenantID + ":" + node.OwnerAppID}
}

func nodeHasIncomingRel(nodeID, relType, fromNodeID string, rels map[string]GraphRelationship) bool {
	for _, rel := range rels {
		if rel.RelType == relType && rel.ToNodeID == nodeID && rel.FromNodeID == fromNodeID {
			return true
		}
	}
	return false
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildFTSDocument(node write.NodeRecord, acl []string, authorityScore *int) fts.FTSDocument {
	fields := map[string]string{
		"id":           node.ID,
		"node_type":    node.NodeType,
		"domain_id":    node.DomainID,
		"external_ref": node.ExternalRef,
		"status_value": node.StatusValue,
	}
	for key, value := range node.Properties {
		fields[key] = fmt.Sprint(value)
	}
	return fts.FTSDocument{
		NodeID:         node.ID,
		NodeType:       node.NodeType,
		DomainID:       node.DomainID,
		OwnerTenantID:  node.OwnerTenantID,
		OwnerAppID:     node.OwnerAppID,
		ACLVisibleTo:   append([]string(nil), acl...),
		IsDeleted:      node.IsDeleted,
		StatusValue:    node.StatusValue,
		AuthorityScore: authorityScore,
		DomainProps:    cloneMap(node.Properties),
		Fields:         fields,
		CreatedAt:      node.CreatedAt,
	}
}

func appendUnique(values []string, item string) []string {
	for _, value := range values {
		if value == item {
			return values
		}
	}
	return append(values, item)
}

func removeString(values []string, item string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == item {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func stripSyncVersionMetadata(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		if key == "_kg_sync_version" {
			continue
		}
		result[key] = value
	}
	return result
}

type runtimeGraphAdapter struct {
	graph *GraphStore
}

func (a runtimeGraphAdapter) UpsertNode(_ context.Context, node graphstore.GraphNode) error {
	if a.graph == nil {
		return nil
	}
	a.graph.Nodes[node.ID] = GraphNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
		Visibility:    node.Visibility,
		StatusValue:   node.StatusValue,
		IsDeleted:     node.IsDeleted,
		SyncVersion:   node.SyncVersion,
		Properties:    cloneMap(node.Properties),
	}
	a.graph.Nodes[node.ID].Properties["_kg_sync_version"] = node.SyncVersion
	return nil
}

func (a runtimeGraphAdapter) DeleteNode(_ context.Context, nodeID string) error {
	if a.graph == nil {
		return nil
	}
	delete(a.graph.Nodes, nodeID)
	for id, rel := range a.graph.Rels {
		if rel.FromNodeID == nodeID || rel.ToNodeID == nodeID {
			delete(a.graph.Rels, id)
		}
	}
	return nil
}

func (a runtimeGraphAdapter) UpsertRelationship(_ context.Context, rel graphstore.GraphRelationship) error {
	if a.graph == nil {
		return nil
	}
	a.graph.Rels[rel.ID] = GraphRelationship{
		ID:          rel.ID,
		RelType:     rel.RelType,
		FromNodeID:  rel.FromNodeID,
		ToNodeID:    rel.ToNodeID,
		DomainID:    rel.DomainID,
		SyncVersion: rel.SyncVersion,
		Properties:  cloneMap(rel.Properties),
	}
	a.graph.Rels[rel.ID].Properties["_kg_sync_version"] = rel.SyncVersion
	return nil
}

func (a runtimeGraphAdapter) DeleteRelationship(_ context.Context, relID string) error {
	if a.graph == nil {
		return nil
	}
	delete(a.graph.Rels, relID)
	return nil
}

func (a runtimeGraphAdapter) ExecuteQuery(_ context.Context, query graphstore.GraphQuery, params map[string]any) ([]map[string]any, error) {
	if a.graph == nil {
		return nil, nil
	}
	return runtimeGraphQuery(a.graph.SnapshotNodes(), a.graph.SnapshotRelationships(), query, params), nil
}

func (a runtimeGraphAdapter) ListNodes(_ context.Context) ([]graphstore.GraphNode, error) {
	if a.graph == nil {
		return nil, nil
	}
	result := make([]graphstore.GraphNode, 0, len(a.graph.Nodes))
	for _, node := range a.graph.Nodes {
		result = append(result, graphstore.GraphNode{
			ID:            node.ID,
			NodeType:      node.NodeType,
			DomainID:      node.DomainID,
			OwnerTenantID: node.OwnerTenantID,
			OwnerAppID:    node.OwnerAppID,
			ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
			Visibility:    node.Visibility,
			StatusValue:   node.StatusValue,
			IsDeleted:     node.IsDeleted,
			SyncVersion:   node.SyncVersion,
			Properties:    cloneMap(node.Properties),
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		})
	}
	return result, nil
}

func (a runtimeGraphAdapter) ListRelationships(_ context.Context) ([]graphstore.GraphRelationship, error) {
	if a.graph == nil {
		return nil, nil
	}
	result := make([]graphstore.GraphRelationship, 0, len(a.graph.Rels))
	for _, rel := range a.graph.Rels {
		result = append(result, graphstore.GraphRelationship{
			ID:          rel.ID,
			RelType:     rel.RelType,
			FromNodeID:  rel.FromNodeID,
			ToNodeID:    rel.ToNodeID,
			DomainID:    rel.DomainID,
			SyncVersion: rel.SyncVersion,
			Properties:  cloneMap(rel.Properties),
		})
	}
	return result, nil
}

func (a runtimeGraphAdapter) ReadSyncVersion(_ context.Context, entityID string) (int64, error) {
	if a.graph == nil {
		return 0, nil
	}
	if node, ok := a.graph.Nodes[entityID]; ok {
		return node.SyncVersion, nil
	}
	if rel, ok := a.graph.Rels[entityID]; ok {
		return rel.SyncVersion, nil
	}
	return 0, nil
}

func runtimeGraphQuery(nodes map[string]GraphNode, rels map[string]GraphRelationship, query graphstore.GraphQuery, params map[string]any) []map[string]any {
	nodeByID := map[string]GraphNode{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	visibility := visibleOwnerTokensFromAny(params[query.ACLTokensParam])
	results := []map[string]any{}
	for _, node := range nodes {
		if node.IsDeleted || node.NodeType != query.StartNodeType {
			continue
		}
		if !runtimeMatchesNode(node.Properties, query.StartMatch, params) {
			continue
		}
		if !runtimeIsVisible(node, visibility) {
			continue
		}
		payload := cloneMap(node.Properties)
		payload["id"] = node.ID
		payload["node_type"] = node.NodeType
		payload["domain_id"] = node.DomainID
		payload["_kg_sync_version"] = node.SyncVersion
		current := node
		ok := true
		for idx, hop := range query.Hops {
			next, found := runtimeNextHop(current, hop, nodeByID, rels, visibility, params)
			if !found {
				ok = false
				break
			}
			current = next
			payload[fmt.Sprintf("hop_%d", idx+1)] = current.ID
		}
		if !ok {
			continue
		}
		selected := map[string]any{}
		for _, field := range query.ReturnFields {
			selected[field] = runtimeResolveFieldValue(node, current, field)
		}
		for k, v := range payload {
			selected[k] = v
		}
		results = append(results, selected)
	}
	return results
}

func runtimeIsVisible(node GraphNode, visibility map[string]struct{}) bool {
	if len(visibility) == 0 {
		return true
	}
	for _, token := range node.ACLVisibleTo {
		if _, ok := visibility[token]; ok {
			return true
		}
	}
	return false
}

func runtimeMatchesNode(props map[string]any, match map[string]any, params map[string]any) bool {
	for key, raw := range match {
		want, ok := raw.(string)
		if ok && strings.HasPrefix(want, "$") {
			if value, exists := params[strings.TrimPrefix(want, "$")]; exists {
				want = fmt.Sprint(value)
			}
		}
		got, exists := props[key]
		if !exists || fmt.Sprint(got) != want {
			return false
		}
	}
	return true
}

func runtimeNextHop(current GraphNode, hop graphstore.GraphQueryHop, nodeByID map[string]GraphNode, relationships map[string]GraphRelationship, visibility map[string]struct{}, params map[string]any) (GraphNode, bool) {
	for _, rel := range relationships {
		if rel.RelType != hop.RelType {
			continue
		}
		var candidateID string
		switch strings.ToLower(hop.Direction) {
		case "in":
			if rel.ToNodeID != current.ID {
				continue
			}
			candidateID = rel.FromNodeID
		default:
			if rel.FromNodeID != current.ID {
				continue
			}
			candidateID = rel.ToNodeID
		}
		candidate, ok := nodeByID[candidateID]
		if !ok || candidate.IsDeleted || candidate.NodeType != hop.ToNodeType {
			continue
		}
		if !runtimeIsVisible(candidate, visibility) {
			continue
		}
		if !runtimeMatchesNode(candidate.Properties, hop.Filter, params) {
			continue
		}
		return candidate, true
	}
	return GraphNode{}, false
}

func runtimeResolveFieldValue(start, current GraphNode, field string) any {
	if strings.Contains(field, ".") {
		parts := strings.SplitN(field, ".", 2)
		switch parts[0] {
		case start.NodeType:
			return start.Properties[parts[1]]
		case current.NodeType:
			return current.Properties[parts[1]]
		}
	}
	return current.Properties[field]
}

func visibleOwnerTokensFromAny(raw any) map[string]struct{} {
	result := map[string]struct{}{}
	switch value := raw.(type) {
	case []string:
		for _, token := range value {
			result[token] = struct{}{}
		}
	case []any:
		for _, item := range value {
			if token, ok := item.(string); ok {
				result[token] = struct{}{}
			}
		}
	}
	return result
}
