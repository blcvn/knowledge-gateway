package workers

import (
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/write"
)

func TestRuntimeProjectsNodeRelationshipAndCascade(t *testing.T) {
	fixture := newWorkerFixture(t)
	store, ontologySvc, cache := fixture.store, fixture.ontologySvc, fixture.cache
	runtime := NewRuntime(store, ontologySvc, &cache)

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected processed events")
	}
	if _, ok := runtime.Graph().Nodes[fixture.parentID]; !ok {
		t.Fatal("projected graph node missing")
	}
	if _, ok := runtime.Vector().Documents[fixture.childID]; !ok {
		t.Fatal("projected vector doc missing")
	}
	if got := runtime.Graph().Nodes[fixture.childID].StatusValue; got != "con_hieu_luc" {
		t.Fatalf("child status = %q, want con_hieu_luc", got)
	}
}

func TestRuntimePollOnceIsIdempotentForSeenEvents(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)

	first := runtime.PollOnce()
	second := runtime.PollOnce()
	if first.Processed == 0 {
		t.Fatal("first poll expected processed events")
	}
	if second.Processed != 0 || second.Failed != 0 || second.DeadLetter != 0 {
		t.Fatalf("second poll = %+v, want zero work for seen events", second)
	}
}

func TestRuntimeGeneratesNonEmptyEmbeddingsFromSearchableContent(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.PollOnce()

	doc, ok := runtime.Vector().Documents[fixture.parentID]
	if !ok {
		t.Fatal("projected vector doc missing")
	}
	if len(doc.Embedding) == 0 {
		t.Fatal("embedding is empty")
	}
	allZero := true
	for _, value := range doc.Embedding {
		if value != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatalf("embedding = %#v, want non-zero values", doc.Embedding)
	}
}

func TestRuntimeRetriesAndDeadLettersTransientFailures(t *testing.T) {
	runtime := NewRuntime(&failingStore{
		events: []write.OutboxEvent{
			{ID: "evt-1", EventType: "NODE_UPSERTED", Payload: map[string]any{"node_id": "missing"}},
		},
	}, &noopOntology{}, nil)

	for i := 0; i < 3; i++ {
		runtime.PollOnce()
	}
	env, ok := runtime.EventEnvelope("evt-1")
	if !ok {
		t.Fatal("event envelope missing")
	}
	if env.Status != EventDeadLetter {
		t.Fatalf("status = %s, want dead letter", env.Status)
	}
	if env.RetryCount != 3 {
		t.Fatalf("retry_count = %d, want 3", env.RetryCount)
	}
}

func TestRuntimeAccessGrantFanoutInvalidatesACLAndExpandsPayload(t *testing.T) {
	fixture := newWorkerFixture(t)
	store, ontologySvc, cache := fixture.store, fixture.ontologySvc, fixture.cache
	runtime := NewRuntime(store, ontologySvc, &cache)
	runtime.PollOnce()
	if err := cache.SetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", map[string]any{"cached": true}, time.Minute); err != nil {
		t.Fatalf("SetJSON() error = %v", err)
	}

	err := runtime.handleAccessGrantChanged(map[string]any{
		"grantor_tenant_id": "11111111-1111-1111-1111-111111111111",
		"grantee_tenant_id": "22222222-2222-2222-2222-222222222222",
		"grantee_app_id":    "22222222-aaaa-2222-aaaa-222222222222",
		"scope_type":        "domain",
		"scope_value":       "test-domain",
		"permission":        "read",
	})
	if err != nil {
		t.Fatalf("handleAccessGrantChanged() error = %v", err)
	}

	node := runtime.Graph().Nodes[fixture.parentID]
	if !containsString(node.ACLVisibleTo, "22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222") {
		t.Fatalf("ACLVisibleTo = %#v", node.ACLVisibleTo)
	}
	var cached any
	if ok, err := cache.GetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", &cached); err != nil || ok {
		t.Fatalf("expected cache invalidation, ok=%v err=%v", ok, err)
	}

	if err := cache.SetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", map[string]any{"cached": true}, time.Minute); err != nil {
		t.Fatalf("SetJSON() revoke setup error = %v", err)
	}
	if err := runtime.handleAccessGrantChanged(map[string]any{
		"grantor_tenant_id": "11111111-1111-1111-1111-111111111111",
		"grantee_tenant_id": "22222222-2222-2222-2222-222222222222",
		"grantee_app_id":    "22222222-aaaa-2222-aaaa-222222222222",
		"scope_type":        "domain",
		"scope_value":       "test-domain",
		"permission":        "read",
		"status":            "revoked",
	}); err != nil {
		t.Fatalf("handleAccessGrantChanged(revoked) error = %v", err)
	}
	if containsString(runtime.Graph().Nodes[fixture.parentID].ACLVisibleTo, "22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222") {
		t.Fatalf("parent ACLVisibleTo still contains revoked token: %#v", runtime.Graph().Nodes[fixture.parentID].ACLVisibleTo)
	}
	if containsString(runtime.Vector().Documents[fixture.childID].ACLVisibleTo, "22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222") {
		t.Fatalf("child vector ACLVisibleTo still contains revoked token: %#v", runtime.Vector().Documents[fixture.childID].ACLVisibleTo)
	}
	if ok, _ := cache.GetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", &cached); ok {
		t.Fatal("expected cache invalidation after revoke")
	}
}

func TestRuntimeReconcileReportsHealthyState(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.PollOnce()

	if _, err := fixture.writeSvc.UpdateNode(fixture.actor, fixture.childID, write.NodeUpdateRequest{Properties: map[string]any{"tinh_trang": "con_hieu_luc"}}); err != nil {
		t.Fatalf("UpdateNode(child) error = %v", err)
	}
	runtime.PollOnce()

	report := runtime.Reconcile()
	if report.Overall != "pass" {
		t.Fatalf("overall = %q, want pass", report.Overall)
	}
	if report.GraphDriftCount != 0 {
		t.Fatalf("graph_drift_count = %d, want 0", report.GraphDriftCount)
	}
	if report.VectorDriftCount != 0 {
		t.Fatalf("vector_drift_count = %d, want 0", report.VectorDriftCount)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v, want none", report.Issues)
	}
}

func TestRuntimeReconcileReportsReplicaDrift(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.PollOnce()

	parent := runtime.Graph().Nodes[fixture.parentID]
	parent.NodeType = "Mismatch"
	runtime.Graph().Nodes[fixture.parentID] = parent
	delete(runtime.Vector().Documents, fixture.childID)
	runtime.Graph().Nodes["orphan-graph-node"] = GraphNode{ID: "orphan-graph-node", NodeType: "Orphan", DomainID: "test-domain"}
	runtime.Vector().Documents["orphan-vector-node"] = VectorDocument{NodeID: "orphan-vector-node", NodeType: "Orphan", DomainID: "test-domain"}
	for id := range runtime.Graph().Rels {
		delete(runtime.Graph().Rels, id)
		break
	}

	report := runtime.Reconcile()
	if report.Overall != "fail" {
		t.Fatalf("overall = %q, want fail", report.Overall)
	}
	if report.GraphDriftCount == 0 {
		t.Fatal("graph_drift_count = 0, want drift")
	}
	if report.VectorDriftCount == 0 {
		t.Fatal("vector_drift_count = 0, want drift")
	}
	for _, kind := range []string{"graph_mismatch", "vector_mismatch", "orphan_graph_node", "orphan_vector_doc", "missing_relationship"} {
		if !containsIssueKind(report.Issues, kind) {
			t.Fatalf("missing issue kind %q in %#v", kind, report.Issues)
		}
	}
}

type noopOntology struct{}

func (noopOntology) GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error) {
	return nil, nil
}

type failingStore struct {
	events []write.OutboxEvent
}

func (f *failingStore) ListOutboxEvents() []write.OutboxEvent { return f.events }
func (f *failingStore) GetNodeByID(id string) (write.NodeRecord, bool) {
	return write.NodeRecord{}, false
}
func (f *failingStore) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	return write.RelationshipRecord{}, false
}
func (f *failingStore) ListNodes() []write.NodeRecord                 { return nil }
func (f *failingStore) ListRelationships() []write.RelationshipRecord { return nil }

type workerFixture struct {
	store       *write.MemoryStore
	ontologySvc *ontology.Service
	writeSvc    *write.Service
	cache       rediscache.Client
	actor       access.Identity
	parentID    string
	childID     string
}

func newWorkerFixture(t *testing.T) workerFixture {
	t.Helper()
	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}
	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologySvc := ontology.NewService(ontologyStore, accessResolver)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	if _, err := ontologySvc.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{ID: "test-domain", Name: "Test Domain"}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "test-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Parent",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(parent) error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "test-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Child",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(child) error = %v", err)
	}
	if _, err := ontologySvc.CreateRelType(actor, actor.TenantID, "test-domain", ontology.RelTypeCreateRequest{
		RelTypeName:  "PARENT_OF",
		FromNodeType: "Parent",
		ToNodeType:   "Child",
		SameDomain:   true,
	}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateRelType() error = %v", err)
	}
	if _, err := ontologySvc.UpsertStatusFieldConfig(actor, actor.TenantID, "test-domain", ontology.StatusFieldConfigRequest{
		StatusFieldName:   "tinh_trang",
		ValidStatusValues: []string{"con_hieu_luc"},
		CascadeRules: []ontology.CascadeRule{
			{FromNodeType: "Parent", ViaRel: "PARENT_OF", ToNodeType: "Child"},
		},
	}); err != nil {
		t.Fatalf("UpsertStatusFieldConfig() error = %v", err)
	}

	store := write.NewMemoryStore()
	writeSvc := write.NewService(store, ontologySvc, accessResolver, &postgres.SessionManager{}, nil)
	parent, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "test-domain",
		NodeType:   "Parent",
		Properties: map[string]any{"ten": "Cha", "tinh_trang": "con_hieu_luc"},
	})
	if err != nil {
		t.Fatalf("CreateNode(parent) error = %v", err)
	}
	child, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "test-domain",
		NodeType:   "Child",
		Properties: map[string]any{"ten": "Con", "tinh_trang": "khac"},
	})
	if err != nil {
		t.Fatalf("CreateNode(child) error = %v", err)
	}
	if _, err := writeSvc.CreateRelationship(actor, write.RelationshipCreateRequest{
		RelType:    "PARENT_OF",
		FromNodeID: parent.NodeID,
		ToNodeID:   child.NodeID,
		DomainID:   "test-domain",
		Properties: map[string]any{},
	}); err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}

	return workerFixture{
		store:       store,
		ontologySvc: ontologySvc,
		writeSvc:    writeSvc,
		cache:       cache,
		actor:       actor,
		parentID:    parent.NodeID,
		childID:     child.NodeID,
	}
}

func containsText(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsIssueKind(issues []ReconciliationIssue, want string) bool {
	for _, issue := range issues {
		if issue.Kind == want {
			return true
		}
	}
	return false
}
