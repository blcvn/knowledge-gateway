package workers

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"

	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/write"
)

type WriteStore interface {
	ListOutboxEvents() []write.OutboxEvent
	GetNodeByID(id string) (write.NodeRecord, bool)
	GetRelationshipByID(id string) (write.RelationshipRecord, bool)
	ListNodes() []write.NodeRecord
	ListRelationships() []write.RelationshipRecord
}

type OntologyResolver interface {
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
}

type Runtime struct {
	store      WriteStore
	ontology   OntologyResolver
	cache      *rediscache.Client
	graph      *GraphStore
	vector     *VectorStore
	mu         sync.Mutex
	outbox     map[string]EventEnvelope
	maxRetries int
}

func NewRuntime(store WriteStore, ontology OntologyResolver, cache *rediscache.Client) *Runtime {
	return &Runtime{
		store:      store,
		ontology:   ontology,
		cache:      cache,
		graph:      &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:     &VectorStore{Documents: map[string]VectorDocument{}},
		outbox:     map[string]EventEnvelope{},
		maxRetries: 3,
	}
}

func (r *Runtime) Graph() *GraphStore {
	return r.graph
}

func (r *Runtime) Vector() *VectorStore {
	return r.vector
}

func (r *Runtime) PollOnce() WorkerReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	report := WorkerReport{}
	for _, event := range r.store.ListOutboxEvents() {
		env, ok := r.outbox[event.ID]
		if !ok {
			env = EventEnvelope{Event: event, Status: EventPending, RetryCount: event.RetryCount}
		}
		if env.Status == EventDone || env.Status == EventDeadLetter {
			continue
		}

		env.Status = EventProcessing
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
			r.outbox[event.ID] = env
			continue
		}
		env.Status = EventDone
		env.Error = ""
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

	for id, source := range sourceNodes {
		graphNode, ok := r.graph.Nodes[id]
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
		if doc, ok := r.vector.Documents[id]; !ok || doc.IsDeleted != source.IsDeleted || doc.StatusValue != source.StatusValue || doc.NodeType != source.NodeType || doc.DomainID != source.DomainID {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_mismatch", Details: "vector projection does not match source node"})
		}
	}
	for id := range r.graph.Nodes {
		if _, ok := sourceNodes[id]; !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_graph_node", Details: "node exists in graph but not source"})
		}
	}
	for id := range r.vector.Documents {
		if _, ok := sourceNodes[id]; !ok {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_vector_doc", Details: "node exists in vector but not source"})
		}
	}
	for id, sourceRel := range sourceRelationships {
		graphRel, ok := r.graph.Rels[id]
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
	for id := range r.graph.Rels {
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
		return r.projectNode(node)
	case "RELATIONSHIP_UPSERTED", "RELATIONSHIP_DELETED":
		relID, _ := event.Payload["relationship_id"].(string)
		rel, ok := r.store.GetRelationshipByID(relID)
		if !ok {
			return errors.New("relationship not found")
		}
		r.graph.Rels[rel.ID] = GraphRelationship{
			ID:         rel.ID,
			RelType:    rel.RelType,
			FromNodeID: rel.FromNodeID,
			ToNodeID:   rel.ToNodeID,
			DomainID:   rel.DomainID,
			Properties: cloneMap(rel.Properties),
		}
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
	cfg, err := r.ontology.GetStatusFieldConfig(node.DomainID)
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
		Embedding:     embedText(buildEmbeddingText(node)),
	}
	if cfg != nil && cfg.AuthorityFieldName != "" && len(cfg.AuthorityValuesMap) > 0 {
		if raw, ok := node.Properties[cfg.AuthorityFieldName]; ok {
			if score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", raw)]; ok {
				doc.AuthorityScore = &score
			}
		}
	}
	r.vector.Documents[node.ID] = doc
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

func embedText(text string) []float64 {
	vec := make([]float64, 8)
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	sum := h.Sum64()
	for i := range vec {
		vec[i] = float64((sum>>(uint(i)*8))&0xff) / 255.0
	}
	return vec
}

func buildEmbeddingText(node write.NodeRecord) string {
	parts := []string{node.ID, node.NodeType, node.DomainID, node.ExternalRef, node.StatusValue}
	for k, v := range node.Properties {
		parts = append(parts, k, fmt.Sprintf("%v", v))
	}
	return strings.Join(parts, " ")
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
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
