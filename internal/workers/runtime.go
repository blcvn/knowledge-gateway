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
					Properties:    cloneMap(node.Properties),
					CreatedAt:     node.CreatedAt,
					UpdatedAt:     node.UpdatedAt,
				}
			}
		}
		if rels, err := r.graphAdapter.ListRelationships(context.Background()); err == nil {
			for _, rel := range rels {
				graphRelationships[rel.ID] = GraphRelationship{
					ID:         rel.ID,
					RelType:    rel.RelType,
					FromNodeID: rel.FromNodeID,
					ToNodeID:   rel.ToNodeID,
					DomainID:   rel.DomainID,
					Properties: cloneMap(rel.Properties),
				}
			}
		}
	}
	vectorDocs := r.vector.SnapshotDocuments()
	if snapshotter, ok := any(r.vectorAdapter).(interface {
		SnapshotDocuments() map[string]vectorstore.VectorDocument
	}); ok && snapshotter != nil {
		vectorDocs = snapshotWorkerVectorDocuments(snapshotter.SnapshotDocuments())
	}

	for id, source := range sourceNodes {
		graphNode, ok := graphNodes[id]
		if !ok {
			report.GraphDriftCount++
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "missing_projection", Details: "node missing from graph/vector projections"})
			continue
		}
		if graphNode.IsDeleted != source.IsDeleted || graphNode.StatusValue != source.StatusValue || graphNode.NodeType != source.NodeType || graphNode.DomainID != source.DomainID {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_mismatch", Details: "graph projection does not match source node"})
		}
		if !reflect.DeepEqual(graphNode.Properties, source.Properties) || !reflect.DeepEqual(graphNode.ACLVisibleTo, nodeACLVisibleTo(source)) {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_payload_mismatch", Details: "graph node payload does not match source node"})
		}
		if doc, ok := vectorDocs[id]; !ok || doc.IsDeleted != source.IsDeleted || doc.StatusValue != source.StatusValue || doc.NodeType != source.NodeType || doc.DomainID != source.DomainID {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_mismatch", Details: "vector projection does not match source node"})
		} else if !reflect.DeepEqual(doc.DomainProps, source.Properties) || !reflect.DeepEqual(doc.ACLVisibleTo, nodeACLVisibleTo(source)) {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_payload_mismatch", Details: "vector document payload does not match source node"})
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
		if !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "missing_relationship", Details: "relationship missing from graph projection"})
			continue
		}
		if graphRel.RelType != sourceRel.RelType || graphRel.FromNodeID != sourceRel.FromNodeID || graphRel.ToNodeID != sourceRel.ToNodeID {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "relationship_mismatch", Details: "graph relationship differs from source"})
		}
	}
	for id := range graphRelationships {
		if _, ok := sourceRelationships[id]; !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_graph_relationship", Details: "relationship exists in graph but not source"})
		}
	}

	if report.GraphDriftCount == 0 && report.VectorDriftCount == 0 {
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
		return r.projectNode(node)
	case "RELATIONSHIP_UPSERTED", "RELATIONSHIP_DELETED":
		relID, _ := event.Payload["relationship_id"].(string)
		rel, ok := r.store.GetRelationshipByID(relID)
		if !ok {
			return errors.New("relationship not found")
		}
		relPayload := GraphRelationship{
			ID:         rel.ID,
			RelType:    rel.RelType,
			FromNodeID: rel.FromNodeID,
			ToNodeID:   rel.ToNodeID,
			DomainID:   rel.DomainID,
			Properties: cloneMap(rel.Properties),
		}
		if r.graphAdapter != nil {
			if event.EventType == "RELATIONSHIP_DELETED" {
				_ = r.graphAdapter.DeleteRelationship(context.Background(), relID)
			} else {
				_ = r.graphAdapter.UpsertRelationship(context.Background(), graphstore.GraphRelationship{
					ID:         relPayload.ID,
					RelType:    relPayload.RelType,
					FromNodeID: relPayload.FromNodeID,
					ToNodeID:   relPayload.ToNodeID,
					DomainID:   relPayload.DomainID,
					Properties: cloneMap(relPayload.Properties),
				})
			}
		}
		r.graph.Rels[rel.ID] = relPayload
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
		StatusValue:   node.StatusValue,
		IsDeleted:     node.IsDeleted,
		Properties:    cloneMap(node.Properties),
	}
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
			DomainProps:    cloneMap(doc.DomainProps),
			Embedding:      append([]float64(nil), doc.Embedding...),
		}); err != nil {
			return err
		}
	}
	r.vector.Documents[node.ID] = doc
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

func snapshotWorkerVectorDocuments(source map[string]vectorstore.VectorDocument) map[string]VectorDocument {
	result := make(map[string]VectorDocument, len(source))
	for id, doc := range source {
		result[id] = VectorDocument{
			NodeID:         doc.NodeID,
			NodeType:       doc.NodeType,
			DomainID:       doc.DomainID,
			OwnerTenantID:  doc.OwnerTenantID,
			OwnerAppID:     doc.OwnerAppID,
			ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
			IsDeleted:      doc.IsDeleted,
			StatusValue:    doc.StatusValue,
			AuthorityScore: doc.AuthorityScore,
			DomainProps:    cloneMap(doc.DomainProps),
			Embedding:      append([]float64(nil), doc.Embedding...),
		}
	}
	return result
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
		Properties:    cloneMap(node.Properties),
	}
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
		ID:         rel.ID,
		RelType:    rel.RelType,
		FromNodeID: rel.FromNodeID,
		ToNodeID:   rel.ToNodeID,
		DomainID:   rel.DomainID,
		Properties: cloneMap(rel.Properties),
	}
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
			ID:         rel.ID,
			RelType:    rel.RelType,
			FromNodeID: rel.FromNodeID,
			ToNodeID:   rel.ToNodeID,
			DomainID:   rel.DomainID,
			Properties: cloneMap(rel.Properties),
		})
	}
	return result, nil
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
