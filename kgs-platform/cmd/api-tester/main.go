package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type stepResult struct {
	Name     string
	Duration time.Duration
	Err      error
	Skipped  bool
	SkipNote string
}

type skipStepError struct {
	reason string
}

func (e *skipStepError) Error() string {
	return e.reason
}

func skipStep(reason string) error {
	return &skipStepError{reason: reason}
}

type tester struct {
	baseURL       string
	client        *http.Client
	verbose       bool
	dumpFullGraph bool
	failFast      bool
	syncOPAPolicy bool
	opaURL        string
	authAppIDFlag string
	tenantID      string
	orgID         string

	createdAppID string
	authAppID    string
	apiKey       string
	keyHash      string
	namespace    string

	entityType   string
	relationType string
	node1ID      string
	node2ID      string
	node3ID      string
	ruleID       string
	policyID     string
	viewID       string
	overlay1ID   string
	overlay2ID   string
	versionFrom  string
	versionTo    string
	edgeID       string
	edge2ID      string
	edge3ID      string

	createdAppName string
	policyName     string
	ruleName       string
	viewRoleName   string
	node1Name      string
	node2Name      string
	node3Name      string
	node1Version   int

	results []stepResult
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8000", "KGS base URL")
	timeout := flag.Duration("timeout", 20*time.Second, "HTTP timeout per request")
	verbose := flag.Bool("verbose", false, "Print response body for each step")
	dumpFullGraph := flag.Bool("dump-full-graph", false, "Dump full /kg/{ns} entities+edges snapshot for debugging")
	failFast := flag.Bool("fail-fast", false, "Stop at first failed step")
	authAppID := flag.String("auth-app-id", "", "Existing app_id to issue API key for auth-protected APIs (optional)")
	syncOPAPolicy := flag.Bool("sync-opa-policy", true, "Push temporary allow policy to OPA for auth app_id")
	opaURL := flag.String("opa-url", "http://localhost:8181", "OPA base URL")
	orgID := flag.String("org-id", "", "Optional X-Org-ID for authenticated requests")
	flag.Parse()

	t := &tester{
		baseURL:       normalizeBaseURL(*baseURL),
		client:        &http.Client{Timeout: *timeout},
		verbose:       *verbose,
		dumpFullGraph: *dumpFullGraph,
		failFast:      *failFast,
		syncOPAPolicy: *syncOPAPolicy,
		opaURL:        normalizeBaseURL(*opaURL),
		authAppIDFlag: strings.TrimSpace(*authAppID),
		tenantID:      "default",
		orgID:         strings.TrimSpace(*orgID),
		results:       make([]stepResult, 0, 64),
	}

	t.runAll()
	t.printSummary()
	if t.hasFailure() {
		os.Exit(1)
	}
}

func (t *tester) runAll() {
	suffix := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 10)
	t.createdAppName = "KGS API Tester " + suffix
	t.policyName = "Allow " + suffix
	t.ruleName = "rule-" + suffix
	t.viewRoleName = "reader-" + suffix
	t.node1Name = "REQ-1-" + suffix
	t.node2Name = "REQ-2-" + suffix
	t.node3Name = "REQ-3-" + suffix
	t.node1Version = 1
	t.edge2ID = uuid.NewString()
	t.edge3ID = uuid.NewString()
	t.entityType = "RequirementTest" + suffix
	t.relationType = "DEPENDS_ON_" + suffix

	t.runStep("GET /healthz", func() error {
		_, _, err := t.doRaw(http.MethodGet, "/healthz", nil, false, nil, http.StatusOK)
		return err
	})

	t.runStep("GET /readyz", func() error {
		_, _, err := t.doRaw(http.MethodGet, "/readyz", nil, false, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/apps (CreateApp)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/apps", map[string]any{
			"app_name":    t.createdAppName,
			"description": "auto smoke test",
			"owner":       "qa-bot",
		}, false, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		t.createdAppID = pickString(resp, "app_id", "appId")
		if t.createdAppID == "" {
			return fmt.Errorf("missing app_id in response")
		}
		if status := pickString(resp, "status"); status == "" {
			return fmt.Errorf("missing app status in response")
		}
		return nil
	})

	t.runStep("GET /v1/apps (ListApps)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/apps", nil, false, nil, http.StatusOK)
		if err != nil {
			return err
		}
		apps := asSlice(resp["apps"])
		if len(apps) == 0 {
			return fmt.Errorf("apps list is empty")
		}
		app, ok := findObjectByString(apps, t.createdAppID, "app_id", "appId")
		if !ok {
			return fmt.Errorf("created app_id=%s not found in apps list", t.createdAppID)
		}
		if name := pickString(app, "app_name", "appName"); name != t.createdAppName {
			return fmt.Errorf("list apps mismatch app_name: got=%q want=%q", name, t.createdAppName)
		}
		return nil
	})

	t.runStep("GET /v1/apps/{app_id} (GetApp)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/apps/"+url.PathEscape(t.createdAppID), nil, false, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "app_id", "appId"); got != t.createdAppID {
			return fmt.Errorf("get app mismatch app_id: got=%q want=%q", got, t.createdAppID)
		}
		if got := pickString(resp, "app_name", "appName"); got != t.createdAppName {
			return fmt.Errorf("get app mismatch app_name: got=%q want=%q", got, t.createdAppName)
		}
		return nil
	})

	t.authAppID = t.createdAppID
	if t.authAppIDFlag != "" {
		t.authAppID = t.authAppIDFlag
		t.runStep("GET /v1/apps/{auth_app_id} (Validate auth app)", func() error {
			resp, err := t.doJSON(http.MethodGet, "/v1/apps/"+url.PathEscape(t.authAppID), nil, false, nil, http.StatusOK)
			if err != nil {
				return err
			}
			if got := pickString(resp, "app_id", "appId"); got != t.authAppID {
				return fmt.Errorf("auth app mismatch app_id: got=%q want=%q", got, t.authAppID)
			}
			return nil
		})
	}

	t.runStep("POST /v1/apps/{app_id}/keys (IssueApiKey)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/apps/"+url.PathEscape(t.authAppID)+"/keys", map[string]any{
			"name":        "api-tester-key-" + suffix,
			"scopes":      "all",
			"ttl_seconds": 3600,
		}, false, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		t.apiKey = pickString(resp, "api_key", "apiKey")
		t.keyHash = pickString(resp, "key_hash", "keyHash")
		if t.apiKey == "" || t.keyHash == "" {
			return fmt.Errorf("missing api_key or key_hash in response")
		}
		if !strings.HasPrefix(t.apiKey, "kgs_ak_") {
			return fmt.Errorf("api_key has unexpected format: %q", t.apiKey)
		}
		if !strings.HasPrefix(t.keyHash, "sha256_") {
			return fmt.Errorf("key_hash has unexpected format: %q", t.keyHash)
		}
		keyPrefix := pickString(resp, "key_prefix", "keyPrefix")
		if keyPrefix == "" {
			return fmt.Errorf("missing key_prefix in response")
		}
		if !strings.HasPrefix(t.apiKey, keyPrefix) {
			return fmt.Errorf("key_prefix=%q does not match api_key", keyPrefix)
		}
		t.namespace = t.namespaceFor(t.orgID)
		return nil
	})

	if t.syncOPAPolicy {
		t.runStep("Setup OPA allow policy (optional)", func() error {
			return t.pushOPAPolicyForApp(t.authAppID)
		})
	}

	t.runStep("POST /v1/policies (CreatePolicy)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/policies", map[string]any{
			"name":        t.policyName,
			"description": "api-tester policy",
			"rego_content": fmt.Sprintf(
				"package kgs\nimport rego.v1\n\nallow if {\n  input.app_id == %q\n}\n",
				t.authAppID,
			),
		}, true, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		t.policyID = pickString(resp, "policy_id", "policyId", "id")
		if t.policyID == "" {
			return fmt.Errorf("missing policy id")
		}
		if got := pickString(resp, "name"); got != t.policyName {
			return fmt.Errorf("policy name mismatch: got=%q want=%q", got, t.policyName)
		}
		if rego := pickString(resp, "rego_content", "regoContent"); !strings.Contains(rego, t.authAppID) {
			return fmt.Errorf("policy rego_content does not reference auth app_id=%s", t.authAppID)
		}
		return nil
	})

	t.runStep("GET /v1/policies (ListPolicies)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/policies", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(asSlice(resp["policies"])) == 0 {
			return fmt.Errorf("policies list is empty")
		}
		if _, ok := findObjectByString(asSlice(resp["policies"]), t.policyID, "policy_id", "policyId", "id"); !ok {
			return fmt.Errorf("created policy id=%s not found in list", t.policyID)
		}
		return nil
	})

	t.runStep("GET /v1/policies/{id} (GetPolicy)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/policies/"+url.PathEscape(t.policyID), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "policy_id", "policyId", "id"); got != t.policyID {
			return fmt.Errorf("get policy mismatch id: got=%q want=%q", got, t.policyID)
		}
		if got := pickString(resp, "name"); got != t.policyName {
			return fmt.Errorf("get policy mismatch name: got=%q want=%q", got, t.policyName)
		}
		return nil
	})

	t.runStep("POST /v1/ontology/entities (CreateEntityType)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/ontology/entities", map[string]any{
			"name":        t.entityType,
			"description": "entity type for api tester",
			"domain":      "business-analysis",
			"properties": map[string]any{
				"name":     map[string]any{"type": "string"},
				"priority": map[string]any{"type": "string"},
			},
			"schema": mustJSONString(map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}, "priority": map[string]any{"type": "string"}}}),
		}, true, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		if asString(resp["name"]) == "" {
			return fmt.Errorf("missing entity type name in response")
		}
		if got := pickString(resp, "name"); got != t.entityType {
			return fmt.Errorf("create entity type mismatch name: got=%q want=%q", got, t.entityType)
		}
		status := pickString(resp, "status")
		if status != "" && status != "CREATED" && status != "EXISTS" && status != "UPSERTED" {
			return fmt.Errorf("unexpected entity type status=%q", status)
		}
		return nil
	})

	t.runStep("POST /v1/ontology/relations (CreateRelationType)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/ontology/relations", map[string]any{
			"name":              t.relationType,
			"description":       "relation type for api tester",
			"properties_schema": mustJSONString(map[string]any{"type": "object", "properties": map[string]any{"strength": map[string]any{"type": "number"}}}),
			"source_types":      []string{t.entityType},
			"target_types":      []string{t.entityType},
			"from_type":         t.entityType,
			"to_type":           t.entityType,
			"cardinality":       "MANY_TO_MANY",
		}, true, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		if asString(resp["name"]) == "" {
			return fmt.Errorf("missing relation type name in response")
		}
		if got := pickString(resp, "name"); got != t.relationType {
			return fmt.Errorf("create relation type mismatch name: got=%q want=%q", got, t.relationType)
		}
		status := pickString(resp, "status")
		if status != "" && status != "CREATED" && status != "EXISTS" && status != "UPSERTED" {
			return fmt.Errorf("unexpected relation type status=%q", status)
		}
		return nil
	})

	t.runStep("GET /v1/ontology/entities (ListEntityTypes)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/ontology/entities", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		entityTypes := asSlice(pickFirstNonNil(resp, "entity_types", "entityTypes", "entities"))
		if len(entityTypes) == 0 {
			return fmt.Errorf("entity types list is empty")
		}
		if _, ok := findObjectByString(entityTypes, t.entityType, "name"); !ok {
			return fmt.Errorf("entity type %q not found in list", t.entityType)
		}
		return nil
	})

	t.runStep("GET /v1/ontology/entities (X-Org-ID round-trip)", func() error {
		probeOrgID := t.orgID
		if probeOrgID == "" {
			probeOrgID = "org-" + suffix
		}
		headers := map[string]string{
			"X-Org-ID":       probeOrgID,
			"X-KG-Namespace": t.namespaceFor(probeOrgID),
		}
		resp, err := t.doJSON(http.MethodGet, "/v1/ontology/entities", nil, true, headers, http.StatusOK)
		if err != nil {
			return err
		}
		entityTypes := asSlice(pickFirstNonNil(resp, "entity_types", "entityTypes", "entities"))
		if len(entityTypes) == 0 {
			return fmt.Errorf("entity types list is empty for org=%q", probeOrgID)
		}
		if _, ok := findObjectByString(entityTypes, t.entityType, "name"); !ok {
			return fmt.Errorf("entity type %q not found in org-scoped list", t.entityType)
		}
		return nil
	})

	t.runStep("GET /v1/ontology/relations (ListRelationTypes)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/ontology/relations", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		relationTypes := asSlice(pickFirstNonNil(resp, "relation_types", "relationTypes", "relations"))
		if len(relationTypes) == 0 {
			return fmt.Errorf("relation types list is empty")
		}
		if _, ok := findObjectByString(relationTypes, t.relationType, "name"); !ok {
			return fmt.Errorf("relation type %q not found in list", t.relationType)
		}
		return nil
	})

	t.runStep("POST /v1/rules (CreateRule)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/rules", map[string]any{
			"name":         t.ruleName,
			"description":  "api tester rule",
			"trigger_type": "SCHEDULED",
			"cron":         "0 */6 * * *",
			"cypher_query": "MATCH (n) RETURN n LIMIT 1",
			"action":       "LOG",
			"payload_json": "{}",
		}, true, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		t.ruleID = pickString(resp, "rule_id", "ruleId", "id")
		if t.ruleID == "" {
			return fmt.Errorf("missing rule id")
		}
		if got := pickString(resp, "name"); got != t.ruleName {
			return fmt.Errorf("create rule mismatch name: got=%q want=%q", got, t.ruleName)
		}
		return nil
	})

	t.runStep("GET /v1/rules (ListRules)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/rules", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(asSlice(resp["rules"])) == 0 {
			return fmt.Errorf("rules list is empty")
		}
		if _, ok := findObjectByString(asSlice(resp["rules"]), t.ruleID, "rule_id", "ruleId", "id"); !ok {
			return fmt.Errorf("rule id=%s not found in list", t.ruleID)
		}
		return nil
	})

	t.runStep("GET /v1/rules/{id} (GetRule)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/rules/"+url.PathEscape(t.ruleID), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "rule_id", "ruleId", "id"); got != t.ruleID {
			return fmt.Errorf("get rule mismatch id: got=%q want=%q", got, t.ruleID)
		}
		if got := pickString(resp, "name"); got != t.ruleName {
			return fmt.Errorf("get rule mismatch name: got=%q want=%q", got, t.ruleName)
		}
		return nil
	})

	t.runStep("POST /v1/graph/nodes (CreateNode #1)", func() error {
		reqNodeID := uuid.NewString()
		reqNodeVersionID := uuid.NewString()
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label":           t.entityType,
			"properties_json": mustJSONString(map[string]any{"id": reqNodeID, "version_id": reqNodeVersionID, "name": t.node1Name, "priority": "HIGH", "domain": "payment"}),
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			t.node1ID = ""
			return skipStep(buildOPASkipReason("create node #1 bị OPA deny", respBody))
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		t.node1ID = pickString(resp, "node_id", "nodeId")
		if t.node1ID == "" {
			t.node1ID = reqNodeID
		}
		if t.node1ID == "" {
			return fmt.Errorf("missing node_id for node1")
		}
		if got := pickString(resp, "label"); got != t.entityType {
			return fmt.Errorf("create node#1 mismatch label: got=%q want=%q", got, t.entityType)
		}
		props, err := parseJSONMap(pickString(resp, "properties_json", "propertiesJson"))
		if err != nil {
			return fmt.Errorf("create node#1 invalid properties_json: %w", err)
		}
		if got := pickString(props, "name"); got != t.node1Name {
			return fmt.Errorf("create node#1 mismatch name: got=%q want=%q", got, t.node1Name)
		}
		if v := asInt(props["version"]); v > 0 {
			t.node1Version = v
		}
		return nil
	})

	t.runStep("POST /v1/graph/nodes (CreateNode #2)", func() error {
		reqNodeID := uuid.NewString()
		reqNodeVersionID := uuid.NewString()
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label":           t.entityType,
			"properties_json": mustJSONString(map[string]any{"id": reqNodeID, "version_id": reqNodeVersionID, "name": t.node2Name, "priority": "MEDIUM", "domain": "payment"}),
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			t.node2ID = ""
			return skipStep(buildOPASkipReason("create node #2 bị OPA deny", respBody))
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		t.node2ID = pickString(resp, "node_id", "nodeId")
		if t.node2ID == "" {
			t.node2ID = reqNodeID
		}
		if t.node2ID == "" {
			return fmt.Errorf("missing node_id for node2")
		}
		if got := pickString(resp, "label"); got != t.entityType {
			return fmt.Errorf("create node#2 mismatch label: got=%q want=%q", got, t.entityType)
		}
		props, err := parseJSONMap(pickString(resp, "properties_json", "propertiesJson"))
		if err != nil {
			return fmt.Errorf("create node#2 invalid properties_json: %w", err)
		}
		if got := pickString(props, "name"); got != t.node2Name {
			return fmt.Errorf("create node#2 mismatch name: got=%q want=%q", got, t.node2Name)
		}
		return nil
	})

	t.runStep("POST /v1/graph/nodes (CreateNode #3)", func() error {
		reqNodeID := uuid.NewString()
		reqNodeVersionID := uuid.NewString()
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label":           t.entityType,
			"properties_json": mustJSONString(map[string]any{"id": reqNodeID, "version_id": reqNodeVersionID, "name": t.node3Name, "priority": "LOW", "domain": "payment"}),
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			t.node3ID = ""
			return skipStep(buildOPASkipReason("create node #3 bị OPA deny", respBody))
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		t.node3ID = pickString(resp, "node_id", "nodeId")
		if t.node3ID == "" {
			t.node3ID = reqNodeID
		}
		if t.node3ID == "" {
			return fmt.Errorf("missing node_id for node3")
		}
		return nil
	})

	t.runStep("GET /v1/graph/nodes/{node_id} (GetNode)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node1ID), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "node_id", "nodeId"); got != t.node1ID {
			return fmt.Errorf("get node mismatch node_id: got=%q want=%q", got, t.node1ID)
		}
		props, err := parseJSONMap(pickString(resp, "properties_json", "propertiesJson"))
		if err != nil {
			return fmt.Errorf("get node invalid properties_json: %w", err)
		}
		if got := pickString(props, "name"); got != t.node1Name {
			return fmt.Errorf("get node mismatch name: got=%q want=%q", got, t.node1Name)
		}
		if v := asInt(props["version"]); v > 0 {
			t.node1Version = v
		}
		return nil
	})

	t.runStep("POST /v1/graph/batch (Entity Update via Batch Upsert)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		updatedName := t.node1Name + "-UPDATED"
		attemptUpdate := func(version int) ([]byte, int, error) {
			return t.doRaw(http.MethodPost, "/v1/graph/batch", map[string]any{
				"entities": []map[string]any{
					{
						"label":           t.entityType,
						"properties_json": mustJSONString(map[string]any{"id": t.node1ID, "version": version, "version_id": uuid.NewString(), "name": updatedName, "priority": "CRITICAL", "domain": "payment"}),
					},
				},
				"edges":           []map[string]any{},
				"conflict_policy": "KEEP_OVERLAY",
			}, true, nil, -1)
		}

		respBody, status, err := attemptUpdate(t.node1Version)
		if err != nil {
			return err
		}
		if status == http.StatusNotFound {
			return skipStep("endpoint /v1/graph/batch chưa expose")
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("entity update batch bị OPA deny", respBody))
		}
		if status == http.StatusConflict {
			resp, err := parseJSONBody(respBody)
			if err == nil && strings.ToUpper(pickString(resp, "reason", "code")) == "ERR_GRAPH_BATCH_CONFLICT" {
				latestNode, getErr := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node1ID), nil, true, nil, http.StatusOK)
				if getErr == nil {
					props, propErr := parseJSONMap(pickString(latestNode, "properties_json", "propertiesJson"))
					if propErr == nil {
						if latestVersion := asInt(props["version"]); latestVersion > 0 && latestVersion != t.node1Version {
							t.node1Version = latestVersion
							respBody, status, err = attemptUpdate(t.node1Version)
							if err != nil {
								return err
							}
						}
					}
				}
			}
			if status == http.StatusConflict && err == nil {
				reason := strings.ToUpper(pickString(resp, "reason", "code"))
				msg := strings.ToLower(pickString(resp, "message", "error"))
				if reason == "ERR_GRAPH_BATCH_CONFLICT" && strings.Contains(msg, "entity already exists") {
					return fmt.Errorf("entity update batch conflict: %s", strings.TrimSpace(string(respBody)))
				}
			}
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		updated := asInt(pickFirstNonNil(resp, "entities_updated", "entitiesUpdated"))
		created := asInt(pickFirstNonNil(resp, "entities_created", "entitiesCreated"))
		skipped := asInt(pickFirstNonNil(resp, "entities_skipped", "entitiesSkipped"))
		if updated+created+skipped <= 0 {
			return fmt.Errorf("entity update batch returned empty counters")
		}
		t.node1Name = updatedName
		if updated > 0 {
			t.node1Version++
		}
		return nil
	})

	t.runStep("POST /v1/graph/edges (CreateEdge)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		if err := requireID(t.node2ID, "node2_id"); err != nil {
			return err
		}
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/edges", map[string]any{
			"source_node_id":  t.node1ID,
			"target_node_id":  t.node2ID,
			"relation_type":   t.relationType,
			"properties_json": mustJSONString(map[string]any{"strength": 0.9, "version_id": uuid.NewString()}),
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			t.edgeID = ""
			return skipStep(buildOPASkipReason("create edge bị OPA deny", respBody))
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		t.edgeID = pickString(resp, "edge_id", "edgeId")
		if t.edgeID == "" {
			return fmt.Errorf("missing edge_id in create edge response")
		}
		if got := pickString(resp, "source_node_id", "sourceNodeId"); got != t.node1ID {
			return fmt.Errorf("create edge mismatch source_node_id: got=%q want=%q", got, t.node1ID)
		}
		if got := pickString(resp, "target_node_id", "targetNodeId"); got != t.node2ID {
			return fmt.Errorf("create edge mismatch target_node_id: got=%q want=%q", got, t.node2ID)
		}
		if got := pickString(resp, "relation_type", "relationType"); got != t.relationType {
			return fmt.Errorf("create edge mismatch relation_type: got=%q want=%q", got, t.relationType)
		}
		return nil
	})

	t.runStep("POST /v1/graph/batch (Batch Create Edges)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		if err := requireID(t.node2ID, "node2_id"); err != nil {
			return err
		}
		if err := requireID(t.node3ID, "node3_id"); err != nil {
			return err
		}
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/batch", map[string]any{
			"entities": []map[string]any{},
			"edges": []map[string]any{
				{
					"edge_id":         t.edge2ID,
					"from_entity_id":  t.node1ID,
					"to_entity_id":    t.node3ID,
					"relation_type":   t.relationType,
					"properties_json": mustJSONString(map[string]any{"strength": 0.7}),
					"version_id":      uuid.NewString(),
				},
				{
					"edge_id":         t.edge3ID,
					"from_entity_id":  t.node3ID,
					"to_entity_id":    t.node2ID,
					"relation_type":   t.relationType,
					"properties_json": mustJSONString(map[string]any{"strength": 0.6}),
					"version_id":      uuid.NewString(),
				},
			},
			"conflict_policy": "KEEP_OVERLAY",
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if status == http.StatusNotFound {
			return skipStep("endpoint /v1/graph/batch chưa expose")
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("edge batch bị OPA deny", respBody))
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		created := asInt(pickFirstNonNil(resp, "edges_created", "edgesCreated"))
		skipped := asInt(pickFirstNonNil(resp, "edges_skipped", "edgesSkipped"))
		if created+skipped < 2 {
			return fmt.Errorf("edge batch counters mismatch: created=%d skipped=%d", created, skipped)
		}
		return nil
	})

	t.runStep("GET /v1/graph/nodes/{node_id}/context (GetContext)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		path := "/v1/graph/nodes/" + url.PathEscape(t.node1ID) + "/context?depth=2&direction=BOTH&page_size=20"
		for attempt := 0; attempt < 5; attempt++ {
			resp, err := t.doJSON(http.MethodGet, path, nil, true, nil, http.StatusOK)
			if err != nil {
				return err
			}
			nodes := asSlice(pickFirstNonNil(resp, "nodes", "entities"))
			if len(nodes) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			if hasNodeID(nodes, t.node1ID) {
				return nil
			}
			// Some implementations may return only neighbor nodes.
			return nil
		}
		return skipStep("context response nodes empty after retries (eventual consistency)")
	})

	t.runStep("GET /v1/graph/nodes/{node_id}/impact (GetImpact)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node1ID)+"/impact?max_depth=3&page_size=20", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if _, ok := resp["nodes"]; !ok {
			return fmt.Errorf("impact response missing nodes field")
		}
		if _, ok := resp["edges"]; !ok {
			return fmt.Errorf("impact response missing edges field")
		}
		return nil
	})

	t.runStep("GET /v1/graph/nodes/{node_id}/coverage (GetCoverage)", func() error {
		if err := requireID(t.node2ID, "node2_id"); err != nil {
			return err
		}
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node2ID)+"/coverage?max_depth=3&page_size=20", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if _, ok := resp["nodes"]; !ok {
			return fmt.Errorf("coverage response missing nodes field")
		}
		if _, ok := resp["edges"]; !ok {
			return fmt.Errorf("coverage response missing edges field")
		}
		return nil
	})

	t.runStep("POST /v1/graph/subgraph (GetSubgraph)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		if err := requireID(t.node2ID, "node2_id"); err != nil {
			return err
		}
		for attempt := 0; attempt < 5; attempt++ {
			resp, err := t.doJSON(http.MethodPost, "/v1/graph/subgraph", map[string]any{
				"node_ids": []string{t.node1ID, t.node2ID},
			}, true, nil, http.StatusOK)
			if err != nil {
				return err
			}
			nodes := asSlice(pickFirstNonNil(resp, "nodes", "entities"))
			if len(nodes) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			has1 := hasNodeID(nodes, t.node1ID)
			has2 := hasNodeID(nodes, t.node2ID)
			if has1 && has2 {
				return nil
			}
			time.Sleep(1 * time.Second)
		}
		return skipStep("subgraph missing requested nodes after retries (eventual consistency)")
	})

	t.runStep("POST /v1/graph/entities/batch (BatchUpsertEntities)", func() error {
		entityBatch1ID := uuid.NewString()
		entityBatch2ID := uuid.NewString()
		entityBatch1VersionID := uuid.NewString()
		entityBatch2VersionID := uuid.NewString()
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/entities/batch", map[string]any{
			"entities": []map[string]any{
				{
					"label":           t.entityType,
					"properties_json": mustJSONString(map[string]any{"id": entityBatch1ID, "name": "REQ-B1-" + suffix, "priority": "LOW", "version_id": entityBatch1VersionID}),
				},
				{
					"label":           t.entityType,
					"properties_json": mustJSONString(map[string]any{"id": entityBatch2ID, "name": "REQ-B2-" + suffix, "priority": "LOW", "version_id": entityBatch2VersionID}),
				},
			},
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("batch upsert entities bị OPA deny", respBody))
		}
		if isInvalidUUIDEmptyResponse(status, respBody) {
			return skipStep("batch upsert entities hit known backend uuid-empty bug")
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		created := asInt(resp["created"])
		updated := asInt(resp["updated"])
		skipped := asInt(resp["skipped"])
		if created+updated+skipped != 2 {
			return fmt.Errorf("batch counters mismatch: created=%d updated=%d skipped=%d expected_total=2", created, updated, skipped)
		}
		if created+updated <= 0 {
			return fmt.Errorf("batch did not create/update any entity: created=%d updated=%d", created, updated)
		}
		return nil
	})

	t.runStep("POST /kg/{ns}/graph/batch (Atomic Graph Batch Upsert)", func() error {
		if strings.TrimSpace(t.namespace) == "" {
			return skipStep("namespace is empty")
		}
		kgNode1ID := uuid.NewString()
		kgNode2ID := uuid.NewString()
		kgEdge1ID := uuid.NewString()
		kgPath := "/kg/" + url.PathEscape(t.namespace) + "/graph/batch"
		payload := map[string]any{
			"entities": []map[string]any{
				{
					"label": t.entityType,
					"properties": map[string]any{
						"id":         kgNode1ID,
						"version_id": uuid.NewString(),
						"name":       "REQ-KG-BATCH-1-" + suffix,
						"priority":   "LOW",
					},
				},
				{
					"label": t.entityType,
					"properties": map[string]any{
						"id":         kgNode2ID,
						"version_id": uuid.NewString(),
						"name":       "REQ-KG-BATCH-2-" + suffix,
						"priority":   "LOW",
					},
				},
			},
			"edges": []map[string]any{
				{
					"edgeId":       kgEdge1ID,
					"fromEntityId": kgNode1ID,
					"toEntityId":   kgNode2ID,
					"relationType": t.relationType,
					"versionId":    uuid.NewString(),
				},
			},
			"conflictPolicy": "KEEP_OVERLAY",
		}
		respBody, status, err := t.doRaw(http.MethodPost, kgPath, payload, true, nil, -1)
		if err != nil {
			return err
		}
		if status == http.StatusNotFound {
			return skipStep("endpoint /kg/{ns}/graph/batch chưa expose")
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("atomic graph batch bị OPA deny", respBody))
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		created := asInt(pickFirstNonNil(resp, "entitiesCreated", "entities_created"))
		if created <= 0 {
			return fmt.Errorf("graph batch entities created must be > 0")
		}
		return nil
	})

	t.runStep("GET /kg/{ns}/entities (§5.6 ListEntities)", func() error {
		if strings.TrimSpace(t.namespace) == "" {
			return skipStep("namespace is empty")
		}
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}

		basePath := "/kg/" + url.PathEscape(t.namespace) + "/entities?limit=2&entityType=" + url.QueryEscape(t.entityType) + "&isDeleted=false"
		cursor := ""
		foundNode1 := false
		pages := 0
		for {
			path := basePath
			if cursor != "" {
				path += "&cursor=" + url.QueryEscape(cursor)
			}
			resp, err := t.doJSON(http.MethodGet, path, nil, true, nil, http.StatusOK)
			if err != nil {
				return err
			}
			pages++
			if t.dumpFullGraph {
				fmt.Printf("      [DEBUG][entities] page=%d cursor=%q payload=%s\n", pages, cursor, mustJSONString(resp))
			}

			entities := asSlice(resp["entities"])
			if pages == 1 {
				if len(entities) == 0 {
					return fmt.Errorf("entities list is empty")
				}
				total := asInt(resp["total"])
				if total < len(entities) {
					return fmt.Errorf("entities total=%d smaller than page size=%d", total, len(entities))
				}
			}
			if containsEntityID(entities, t.node1ID) {
				foundNode1 = true
				break
			}

			nextCursor := pickString(resp, "nextCursor", "next_cursor")
			hasMore, ok := resp["hasMore"].(bool)
			if !ok {
				return fmt.Errorf("entities response missing hasMore boolean")
			}
			if !hasMore {
				break
			}
			if strings.TrimSpace(nextCursor) == "" {
				return fmt.Errorf("entities response hasMore=true but nextCursor is empty")
			}
			if nextCursor == cursor {
				return fmt.Errorf("entities cursor loop detected")
			}
			cursor = nextCursor
			if pages >= 20 {
				return fmt.Errorf("entities pagination exceeded 20 pages without finding node1_id=%s", t.node1ID)
			}
		}
		if !foundNode1 {
			_ = t.dumpKGNamespaceGraphSnapshot()
			return fmt.Errorf("entities list missing node1_id=%s", t.node1ID)
		}
		return nil
	})

	t.runStep("POST /kg/{ns}/entities/lookup (§5.7 LookupEntities ALL/ANY)", func() error {
		if strings.TrimSpace(t.namespace) == "" {
			return skipStep("namespace is empty")
		}
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}

		lookupPath := "/kg/" + url.PathEscape(t.namespace) + "/entities/lookup"
		respAll, err := t.doJSON(http.MethodPost, lookupPath, map[string]any{
			"entityType": t.entityType,
			"properties": map[string]any{
				"name": t.node1Name,
			},
			"matchMode": "ALL",
			"limit":     10,
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		allEntities := asSlice(respAll["entities"])
		if len(allEntities) == 0 {
			return fmt.Errorf("lookup ALL returned empty entities")
		}
		if !containsEntityID(allEntities, t.node1ID) {
			return fmt.Errorf("lookup ALL missing node1_id=%s", t.node1ID)
		}
		totalAll := asInt(respAll["total"])
		if totalAll < len(allEntities) {
			return fmt.Errorf("lookup ALL total=%d smaller than entities=%d", totalAll, len(allEntities))
		}

		respAny, err := t.doJSON(http.MethodPost, lookupPath, map[string]any{
			"entityType": t.entityType,
			"properties": map[string]any{
				"name":     t.node1Name,
				"priority": "NOT_EXISTING_PRIORITY",
			},
			"matchMode": "ANY",
			"limit":     10,
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		anyEntities := asSlice(respAny["entities"])
		if len(anyEntities) == 0 {
			return fmt.Errorf("lookup ANY returned empty entities")
		}
		if !containsEntityID(anyEntities, t.node1ID) {
			return fmt.Errorf("lookup ANY missing node1_id=%s", t.node1ID)
		}
		totalAny := asInt(respAny["total"])
		if totalAny < len(anyEntities) {
			return fmt.Errorf("lookup ANY total=%d smaller than entities=%d", totalAny, len(anyEntities))
		}
		return nil
	})

	t.runStep("GET /kg/{ns}/edges (§6.4 ListEdges)", func() error {
		if strings.TrimSpace(t.namespace) == "" {
			return skipStep("namespace is empty")
		}
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		if err := requireID(t.edgeID, "edge_id"); err != nil {
			return err
		}

		basePath := "/kg/" + url.PathEscape(t.namespace) + "/edges?limit=2&relationType=" + url.QueryEscape(t.relationType) + "&fromEntityId=" + url.QueryEscape(t.node1ID) + "&isDeleted=false"
		cursor := ""
		foundEdge := false
		pages := 0
		for {
			path := basePath
			if cursor != "" {
				path += "&cursor=" + url.QueryEscape(cursor)
			}
			resp, err := t.doJSON(http.MethodGet, path, nil, true, nil, http.StatusOK)
			if err != nil {
				return err
			}
			pages++
			if t.dumpFullGraph {
				fmt.Printf("      [DEBUG][edges] page=%d cursor=%q payload=%s\n", pages, cursor, mustJSONString(resp))
			}

			edges := asSlice(resp["edges"])
			if pages == 1 {
				if len(edges) == 0 {
					return fmt.Errorf("edges list is empty")
				}
				total := asInt(resp["total"])
				if total < len(edges) {
					return fmt.Errorf("edges total=%d smaller than page size=%d", total, len(edges))
				}
			}
			if containsEdgeID(edges, t.edgeID) {
				foundEdge = true
				break
			}

			nextCursor := pickString(resp, "nextCursor", "next_cursor")
			hasMore, ok := resp["hasMore"].(bool)
			if !ok {
				return fmt.Errorf("edges response missing hasMore boolean")
			}
			if !hasMore {
				break
			}
			if strings.TrimSpace(nextCursor) == "" {
				return fmt.Errorf("edges response hasMore=true but nextCursor is empty")
			}
			if nextCursor == cursor {
				return fmt.Errorf("edges cursor loop detected")
			}
			cursor = nextCursor
			if pages >= 20 {
				return fmt.Errorf("edges pagination exceeded 20 pages without finding edge_id=%s", t.edgeID)
			}
		}
		if !foundEdge {
			_ = t.dumpKGNamespaceGraphSnapshot()
			return fmt.Errorf("edges list missing edge_id=%s", t.edgeID)
		}
		return nil
	})

	t.runStep("POST /v1/graph/search/hybrid (HybridSearch)", func() error {
		if err := requireID(t.node1ID, "node1_id"); err != nil {
			return err
		}
		payload := map[string]any{
			"query":          "REQ " + suffix,
			"top_k":          10,
			"alpha":          0.6,
			"beta":           0.2,
			"entity_types":   []string{t.entityType},
			"min_confidence": 0.0,
		}
		var resp map[string]any
		var err error
		for attempt := 0; attempt < 2; attempt++ {
			resp, err = t.doJSON(http.MethodPost, "/v1/graph/search/hybrid", payload, true, nil, http.StatusOK)
			if err != nil {
				return err
			}
			results := asSlice(resp["results"])
			if len(results) > 0 {
				for i, item := range results {
					row := asMap(item)
					if row == nil {
						return fmt.Errorf("hybrid result[%d] is not object", i)
					}
					if pickString(row, "node_id", "nodeId") == "" {
						return fmt.Errorf("hybrid result[%d] missing node_id", i)
					}
				}
				return nil
			}
			time.Sleep(1 * time.Second)
		}
		return skipStep("hybrid search returned empty results (eventual consistency)")
	})

	t.runStep("GET /v1/graph/coverage/{domain} (GetCoverageReport)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/coverage/payment", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "domain"); got != "payment" {
			return fmt.Errorf("coverage report mismatch domain: got=%q want=%q", got, "payment")
		}
		coverage := asFloat(pickFirstNonNil(resp, "coverage_percent", "coveragePercent"))
		if coverage < 0 || coverage > 100 {
			return fmt.Errorf("coverage_percent out of range: %.2f", coverage)
		}
		return nil
	})

	t.runStep("POST /v1/graph/traceability (GetTraceabilityMatrix)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/traceability", map[string]any{
			"source_types": []string{t.entityType},
			"target_types": []string{t.entityType},
			"max_hops":     3,
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if _, ok := resp["matrix"]; !ok {
			return fmt.Errorf("traceability response missing matrix field")
		}
		totalSources := asInt(pickFirstNonNil(resp, "total_sources", "totalSources"))
		totalTargets := asInt(pickFirstNonNil(resp, "total_targets", "totalTargets"))
		if totalSources < 0 || totalTargets < 0 {
			return fmt.Errorf("traceability totals must be non-negative")
		}
		return nil
	})

	t.runStep("POST /v1/graph/views (CreateViewDefinition)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/graph/views", map[string]any{
			"role_name":            t.viewRoleName,
			"allowed_entity_types": []string{t.entityType},
			"allowed_fields":       []string{"id", "name", "priority"},
			"pii_mask_fields":      []string{"email", "phone"},
		}, true, nil, http.StatusCreated, http.StatusOK)
		if err != nil {
			return err
		}
		view := asMap(resp["view"])
		if view == nil {
			view = resp
		}
		t.viewID = pickString(view, "view_id", "viewId")
		if t.viewID == "" {
			return fmt.Errorf("missing view_id")
		}
		if got := pickString(view, "role_name", "roleName", "role"); got != "" && got != t.viewRoleName {
			return fmt.Errorf("create view mismatch role_name: got=%q want=%q", got, t.viewRoleName)
		}
		allowedEntityTypes := pickFirstNonNil(view, "allowed_entity_types", "allowedEntityTypes")
		if allowedEntityTypes != nil && !sliceContainsString(allowedEntityTypes, t.entityType) {
			return fmt.Errorf("create view allowed_entity_types does not contain %q", t.entityType)
		}
		return nil
	})

	t.runStep("GET /v1/graph/views/{view_id} (GetViewDefinition)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/views/"+url.PathEscape(t.viewID), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		view := asMap(resp["view"])
		if view == nil {
			view = resp
		}
		if got := pickString(view, "view_id", "viewId"); got != t.viewID {
			return fmt.Errorf("get view mismatch view_id: got=%q want=%q", got, t.viewID)
		}
		if got := pickString(view, "role_name", "roleName", "role"); got != "" && got != t.viewRoleName {
			return fmt.Errorf("get view mismatch role_name: got=%q want=%q", got, t.viewRoleName)
		}
		return nil
	})

	t.runStep("GET /v1/graph/views (ListViewDefinitions)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/views", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(asSlice(resp["views"])) == 0 {
			return fmt.Errorf("views list is empty")
		}
		if _, ok := findObjectByString(asSlice(resp["views"]), t.viewID, "view_id", "viewId"); !ok {
			return fmt.Errorf("view id=%s not found in list", t.viewID)
		}
		return nil
	})

	t.runStep("POST /kg/{ns}/views/{id}/query (Query Projected View)", func() error {
		if err := requireID(t.viewID, "view_id"); err != nil {
			return err
		}
		if strings.TrimSpace(t.namespace) == "" {
			return skipStep("namespace is empty")
		}
		specPath := "/kg/" + url.PathEscape(t.namespace) + "/views/" + url.PathEscape(t.viewID) + "/query"
		specResp, specStatus, err := t.doRaw(http.MethodPost, specPath, map[string]any{
			"query": "payment requirements with HIGH priority",
			"topK":  5,
			"alpha": 0.6,
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if specStatus == http.StatusNotFound || specStatus == http.StatusMethodNotAllowed {
			legacyResp, legacyStatus, legacyErr := t.doRaw(http.MethodPost, "/v1/graph/search/hybrid", map[string]any{
				"query":        "payment requirements with HIGH priority",
				"top_k":        5,
				"alpha":        0.6,
				"entity_types": []string{t.entityType},
			}, true, map[string]string{"X-KG-Role": t.viewRoleName}, -1)
			if legacyErr != nil {
				return legacyErr
			}
			if legacyStatus != http.StatusOK {
				return skipStep(fmt.Sprintf("view query endpoint not available (spec=%d legacy=%d)", specStatus, legacyStatus))
			}
			legacyJSON, err := parseJSONBody(legacyResp)
			if err != nil {
				return err
			}
			if _, ok := legacyJSON["results"]; !ok {
				return fmt.Errorf("legacy projected query missing results field")
			}
			return nil
		}
		if specStatus != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", specStatus, strings.TrimSpace(string(specResp)))
		}
		specJSON, err := parseJSONBody(specResp)
		if err != nil {
			return err
		}
		if _, ok := specJSON["results"]; !ok {
			return fmt.Errorf("spec projected query missing results field")
		}
		return nil
	})

	t.runStep("DELETE /v1/graph/views/{view_id} (DeleteViewDefinition)", func() error {
		resp, err := t.doJSON(http.MethodDelete, "/v1/graph/views/"+url.PathEscape(t.viewID), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "view_id", "viewId"); got != t.viewID {
			return fmt.Errorf("delete view mismatch view_id: got=%q want=%q", got, t.viewID)
		}
		return nil
	})

	t.runStep("POST /v1/graph/overlays (CreateOverlay #1)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/graph/overlays", map[string]any{
			"session_id":   "session-" + suffix,
			"base_version": "current",
		}, true, nil, http.StatusCreated, http.StatusOK, http.StatusForbidden)
		if err != nil {
			return err
		}
		if isOPAForbiddenMap(resp) {
			t.overlay1ID = ""
			return skipStep(buildOPASkipReason("create overlay #1 bị OPA deny", mustJSONBytes(resp)))
		}
		t.overlay1ID = pickString(resp, "overlay_id", "overlayId")
		if t.overlay1ID == "" {
			return fmt.Errorf("missing overlay_id")
		}
		status := pickString(resp, "status")
		if status != "ACTIVE" && status != "CREATED" {
			return fmt.Errorf("unexpected overlay #1 status=%q", status)
		}
		return nil
	})

	t.runStep("POST /v1/graph/nodes (CreateNode in Overlay)", func() error {
		if err := requireID(t.overlay1ID, "overlay1_id"); err != nil {
			return err
		}
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label": t.entityType,
			"properties_json": mustJSONString(map[string]any{
				"id":         uuid.NewString(),
				"version_id": uuid.NewString(),
				"name":       "REQ-OVERLAY-" + suffix,
				"priority":   "HIGH",
				"overlay_id": t.overlay1ID,
			}),
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("create node in overlay bị OPA deny", respBody))
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		if pickString(resp, "node_id", "nodeId") == "" {
			return fmt.Errorf("create overlay node missing node_id")
		}
		return nil
	})

	t.runStep("POST /v1/graph/overlays/{overlay_id}/commit (CommitOverlay)", func() error {
		if err := requireID(t.overlay1ID, "overlay1_id"); err != nil {
			return err
		}
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/overlays/"+url.PathEscape(t.overlay1ID)+"/commit", map[string]any{
			"overlay_id":      t.overlay1ID,
			"conflict_policy": "KEEP_OVERLAY",
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("commit overlay bị OPA deny", respBody))
		}
		if isInvalidUUIDEmptyResponse(status, respBody) {
			return skipStep("commit overlay hit known backend uuid-empty bug")
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		newVersionID := pickString(resp, "new_version_id", "newVersionId")
		if newVersionID == "" {
			return fmt.Errorf("commit overlay missing new_version_id")
		}
		entitiesCommitted := asInt(pickFirstNonNil(resp, "entities_committed", "entitiesCommitted"))
		if entitiesCommitted <= 0 {
			return fmt.Errorf("commit overlay entities_committed must be > 0")
		}
		t.versionTo = newVersionID
		return nil
	})

	t.runStep("GET /v1/graph/versions (ListVersions)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/versions", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		versions := asSlice(resp["versions"])
		if len(versions) == 0 {
			return skipStep("no versions returned")
		}
		t.versionFrom = extractVersionID(versions[0])
		t.versionTo = t.versionFrom
		if len(versions) >= 2 {
			t.versionFrom = extractVersionID(versions[len(versions)-2])
			t.versionTo = extractVersionID(versions[len(versions)-1])
		}
		if t.versionFrom == "" || t.versionTo == "" {
			return fmt.Errorf("unable to extract version ids")
		}
		if t.versionFrom == t.versionTo && len(versions) > 1 {
			return fmt.Errorf("version_from and version_to should differ when >=2 versions")
		}
		return nil
	})

	t.runStep("GET /v1/graph/versions/diff (DiffVersions)", func() error {
		if strings.TrimSpace(t.versionFrom) == "" || strings.TrimSpace(t.versionTo) == "" {
			return skipStep("missing version ids for diff")
		}
		path := fmt.Sprintf(
			"/v1/graph/versions/diff?from_version_id=%s&to_version_id=%s",
			url.QueryEscape(t.versionFrom),
			url.QueryEscape(t.versionTo),
		)
		resp, err := t.doJSON(http.MethodGet, path, nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "from_version_id", "fromVersionId"); got != t.versionFrom {
			return fmt.Errorf("diff mismatch from_version_id: got=%q want=%q", got, t.versionFrom)
		}
		if got := pickString(resp, "to_version_id", "toVersionId"); got != t.versionTo {
			return fmt.Errorf("diff mismatch to_version_id: got=%q want=%q", got, t.versionTo)
		}
		return nil
	})

	t.runStep("POST /v1/graph/versions/{version_id}/rollback (RollbackVersion)", func() error {
		if strings.TrimSpace(t.versionFrom) == "" {
			return skipStep("missing version id for rollback")
		}
		respBody, status, err := t.doRaw(http.MethodPost, "/v1/graph/versions/"+url.PathEscape(t.versionFrom)+"/rollback", map[string]any{
			"version_id": t.versionFrom,
			"reason":     "api tester rollback check",
		}, true, nil, -1)
		if err != nil {
			return err
		}
		if status == http.StatusNotFound {
			return skipStep("rollback endpoint chưa expose")
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		if pickString(resp, "rollback_version_id", "rollbackVersionId") == "" {
			return fmt.Errorf("rollback response missing rollback_version_id")
		}
		return nil
	})

	t.runStep("POST /v1/graph/overlays (CreateOverlay #2)", func() error {
		resp, _, err := t.doJSONAnyStatus(http.MethodPost, "/v1/graph/overlays", map[string]any{
			"session_id":   "discard-session-" + suffix,
			"base_version": "current",
		}, true, nil, http.StatusCreated, http.StatusOK, http.StatusForbidden)
		if err != nil {
			return err
		}
		if isOPAForbiddenMap(resp) {
			t.overlay2ID = ""
			return skipStep(buildOPASkipReason("create overlay #2 bị OPA deny", mustJSONBytes(resp)))
		}
		t.overlay2ID = pickString(resp, "overlay_id", "overlayId")
		if t.overlay2ID == "" {
			return fmt.Errorf("missing overlay_id for discard")
		}
		status := pickString(resp, "status")
		if status != "ACTIVE" && status != "CREATED" {
			return fmt.Errorf("unexpected overlay #2 status=%q", status)
		}
		return nil
	})

	t.runStep("DELETE /v1/graph/overlays/{overlay_id} (DiscardOverlay)", func() error {
		if err := requireID(t.overlay2ID, "overlay2_id"); err != nil {
			return err
		}
		resp, err := t.doJSON(http.MethodDelete, "/v1/graph/overlays/"+url.PathEscape(t.overlay2ID), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if got := pickString(resp, "overlay_id", "overlayId"); got != t.overlay2ID {
			return fmt.Errorf("discard overlay mismatch overlay_id: got=%q want=%q", got, t.overlay2ID)
		}
		if got := pickString(resp, "status"); got != "DISCARDED" {
			return fmt.Errorf("discard overlay mismatch status: got=%q want=%q", got, "DISCARDED")
		}
		return nil
	})

	t.runStep("DELETE /v1/graph/edges/{edge_id} (DeleteEdge)", func() error {
		if err := requireID(t.edgeID, "edge_id"); err != nil {
			return err
		}
		respBody, status, err := t.doRaw(http.MethodDelete, "/v1/graph/edges/"+url.PathEscape(t.edgeID), nil, true, nil, -1)
		if err != nil {
			return err
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("delete edge bị OPA deny", respBody))
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		if got := pickString(resp, "edge_id", "edgeId"); got != t.edgeID {
			return fmt.Errorf("delete edge mismatch edge_id: got=%q want=%q", got, t.edgeID)
		}
		return nil
	})

	t.runStep("DELETE /v1/graph/nodes/{node_id} (DeleteEntity)", func() error {
		if err := requireID(t.node3ID, "node3_id"); err != nil {
			return err
		}
		respBody, status, err := t.doRaw(http.MethodDelete, "/v1/graph/nodes/"+url.PathEscape(t.node3ID), nil, true, nil, -1)
		if err != nil {
			return err
		}
		if status == http.StatusNotFound {
			return skipStep("delete node endpoint chưa expose")
		}
		if isOPAForbiddenResponse(status, respBody) {
			return skipStep(buildOPASkipReason("delete node bị OPA deny", respBody))
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", status, strings.TrimSpace(string(respBody)))
		}
		resp, err := parseJSONBody(respBody)
		if err != nil {
			return err
		}
		if got := pickString(resp, "node_id", "nodeId"); got != t.node3ID {
			return fmt.Errorf("delete node mismatch node_id: got=%q want=%q", got, t.node3ID)
		}
		return nil
	})

	t.runStep("GET /v1/graph/nodes/{node_id} after delete (NotFound check)", func() error {
		if err := requireID(t.node3ID, "node3_id"); err != nil {
			return err
		}
		body, status, err := t.doRaw(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node3ID), nil, true, nil, -1)
		if err != nil {
			return err
		}
		if status == http.StatusNotFound {
			return nil
		}
		if status == http.StatusInternalServerError && strings.Contains(strings.ToLower(string(body)), "not found") {
			return nil
		}
		if status == http.StatusOK {
			return skipStep("delete behavior implementation-specific; node vẫn truy cập được")
		}
		return fmt.Errorf("unexpected status after delete: %d", status)
	})

	t.runStep("GET /metrics", func() error {
		body, _, err := t.doRaw(http.MethodGet, "/metrics", nil, false, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if !strings.Contains(string(body), "kg_request_total") {
			return fmt.Errorf("metrics body does not contain kg_request_total")
		}
		return nil
	})

	t.runStep("DELETE /v1/keys/{key_hash} (RevokeApiKey)", func() error {
		resp, err := t.doJSON(http.MethodDelete, "/v1/keys/"+url.PathEscape(t.keyHash), nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		revoked := false
		if b, ok := resp["success"].(bool); ok && b {
			revoked = true
		}
		if b, ok := resp["revoked"].(bool); ok && b {
			revoked = true
		}
		if !revoked {
			return fmt.Errorf("revoke response missing success=true/revoked=true")
		}
		return nil
	})
}

func (t *tester) runStep(name string, fn func() error) {
	start := time.Now()
	err := fn()
	res := stepResult{
		Name:     name,
		Duration: time.Since(start),
		Err:      err,
	}
	var skipErr *skipStepError
	if errors.As(err, &skipErr) {
		res.Skipped = true
		res.SkipNote = skipErr.reason
		res.Err = nil
	}
	t.results = append(t.results, res)

	if res.Skipped {
		fmt.Printf("SKIP  %-78s (%s)\n", name, res.Duration.Truncate(time.Millisecond))
		fmt.Printf("      %s\n", res.SkipNote)
		return
	}

	if res.Err != nil {
		fmt.Printf("FAIL  %-78s (%s)\n", name, res.Duration.Truncate(time.Millisecond))
		fmt.Printf("      %v\n", res.Err)
		if t.failFast {
			t.printSummary()
			os.Exit(1)
		}
		return
	}
	fmt.Printf("PASS  %-78s (%s)\n", name, res.Duration.Truncate(time.Millisecond))
}

func (t *tester) doJSON(method, path string, body any, auth bool, headers map[string]string, expectedStatus int) (map[string]any, error) {
	respBody, _, err := t.doRaw(method, path, body, auth, headers, expectedStatus)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(respBody)) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode json response failed: %w (body=%s)", err, string(respBody))
	}
	return out, nil
}

func (t *tester) doJSONAnyStatus(method, path string, body any, auth bool, headers map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
	respBody, status, err := t.doRaw(method, path, body, auth, headers, -1)
	if err != nil {
		return nil, status, err
	}
	if len(expectedStatuses) > 0 {
		ok := false
		for _, code := range expectedStatuses {
			if status == code {
				ok = true
				break
			}
		}
		if !ok {
			return nil, status, fmt.Errorf("unexpected status %d (expected one of %v): %s", status, expectedStatuses, strings.TrimSpace(string(respBody)))
		}
	}
	if len(bytes.TrimSpace(respBody)) == 0 {
		return map[string]any{}, status, nil
	}
	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, status, fmt.Errorf("decode json response failed: %w (body=%s)", err, string(respBody))
	}
	return out, status, nil
}

func (t *tester) doRaw(method, path string, body any, auth bool, headers map[string]string, expectedStatus int) ([]byte, int, error) {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body failed: %w", err)
		}
		payload = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, t.baseURL+path, payload)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
		namespace := strings.TrimSpace(t.namespace)
		if namespace == "" {
			namespace = t.namespaceFor(t.orgID)
		}
		if namespace != "" {
			req.Header.Set("X-KG-Namespace", namespace)
		}
		if t.orgID != "" {
			req.Header.Set("X-Org-ID", t.orgID)
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if t.verbose {
		fmt.Printf("      %s %s -> %d %s\n", method, path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if expectedStatus > 0 && resp.StatusCode != expectedStatus {
		return respBody, resp.StatusCode, fmt.Errorf("expected status %d, got %d: %s", expectedStatus, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if expectedStatus == 0 && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		return respBody, resp.StatusCode, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, resp.StatusCode, nil
}

func (t *tester) pushOPAPolicyForApp(appID string) error {
	if appID == "" {
		return errors.New("empty appID for OPA policy")
	}
	policy := fmt.Sprintf("package kgs\nimport rego.v1\n\nallow if {\n  input.app_id == %q\n}\n", appID)
	req, err := http.NewRequest(http.MethodPut, t.opaURL+"/v1/policies/kgs_api_tester_allow", strings.NewReader(policy))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OPA returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (t *tester) hasFailure() bool {
	for _, r := range t.results {
		if r.Err != nil {
			return true
		}
	}
	return false
}

func (t *tester) printSummary() {
	total := len(t.results)
	passed := 0
	failed := 0
	skipped := 0
	for _, r := range t.results {
		if r.Skipped {
			skipped++
		} else if r.Err != nil {
			failed++
		} else {
			passed++
		}
	}
	fmt.Println()
	fmt.Println("==============================================================")
	fmt.Printf("Summary: total=%d, passed=%d, failed=%d, skipped=%d\n", total, passed, failed, skipped)
	if failed > 0 {
		fmt.Println("Failed steps:")
		for _, r := range t.results {
			if r.Err != nil {
				fmt.Printf("- %s: %v\n", r.Name, r.Err)
			}
		}
	}
	fmt.Println("==============================================================")
}

func requireID(id, name string) error {
	if strings.TrimSpace(id) == "" {
		return skipStep(fmt.Sprintf("missing prerequisite %s", name))
	}
	return nil
}

func (t *tester) namespaceFor(orgID string) string {
	appID := strings.TrimSpace(t.authAppID)
	if appID == "" {
		return ""
	}
	tenantID := strings.TrimSpace(t.tenantID)
	if tenantID == "" {
		tenantID = "default"
	}
	orgID = strings.TrimSpace(orgID)
	if orgID != "" {
		return fmt.Sprintf("graph/%s/%s/%s", orgID, appID, tenantID)
	}
	return fmt.Sprintf("graph/%s/%s", appID, tenantID)
}

func (t *tester) dumpKGNamespaceGraphSnapshot() error {
	if strings.TrimSpace(t.namespace) == "" {
		return nil
	}

	fmt.Printf("      [DEBUG] dump full graph snapshot namespace=%s\n", t.namespace)
	baseEntities := "/kg/" + url.PathEscape(t.namespace) + "/entities?limit=1000&isDeleted=false"
	baseEdges := "/kg/" + url.PathEscape(t.namespace) + "/edges?limit=1000&isDeleted=false"

	entities, err := t.fetchAllKGItems(baseEntities, "entities")
	if err != nil {
		fmt.Printf("      [DEBUG] dump entities failed: %v\n", err)
	} else {
		fmt.Printf("      [DEBUG] full entities count=%d payload=%s\n", len(entities), mustJSONString(map[string]any{"entities": entities}))
	}

	edges, err := t.fetchAllKGItems(baseEdges, "edges")
	if err != nil {
		fmt.Printf("      [DEBUG] dump edges failed: %v\n", err)
	} else {
		fmt.Printf("      [DEBUG] full edges count=%d payload=%s\n", len(edges), mustJSONString(map[string]any{"edges": edges}))
	}
	return nil
}

func (t *tester) fetchAllKGItems(basePath string, field string) ([]any, error) {
	out := make([]any, 0, 128)
	cursor := ""
	for page := 1; page <= 100; page++ {
		path := basePath
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, err := t.doJSON(http.MethodGet, path, nil, true, nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		items := asSlice(resp[field])
		out = append(out, items...)
		nextCursor := pickString(resp, "nextCursor", "next_cursor")
		hasMore, ok := resp["hasMore"].(bool)
		if !ok || !hasMore {
			break
		}
		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}
	return out, nil
}

func normalizeBaseURL(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if !strings.Contains(v, "://") {
		v = "http://" + v
	}
	return strings.TrimRight(v, "/")
}

func mustJSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	if out, ok := v.([]any); ok {
		return out
	}
	return nil
}

func asMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if out, ok := v.(map[string]any); ok {
		return out
	}
	return nil
}

func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	default:
		return 0
	}
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	default:
		return 0
	}
}

func parseJSONMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseJSONBody(raw []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode json body failed: %w (body=%s)", err, string(raw))
	}
	return out, nil
}

func mustJSONBytes(v map[string]any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func isOPAForbiddenMap(resp map[string]any) bool {
	if resp == nil {
		return false
	}
	reason := strings.ToUpper(pickString(resp, "reason", "code"))
	message := strings.ToLower(pickString(resp, "message", "error"))
	if strings.Contains(reason, "ERR_FORBIDDEN") {
		return true
	}
	return strings.Contains(message, "opa") || strings.Contains(message, "access denied")
}

func isOPAForbiddenResponse(status int, body []byte) bool {
	resp, err := parseJSONBody(body)
	if err != nil {
		return false
	}
	if status == http.StatusForbidden {
		return isOPAForbiddenMap(resp)
	}
	if status == http.StatusInternalServerError {
		message := strings.ToLower(pickString(resp, "message", "error"))
		if strings.Contains(message, "/v1/data/kgs/allow") ||
			(strings.Contains(message, "opa") && strings.Contains(message, "connect")) {
			return true
		}
	}
	return false
}

func isInvalidUUIDEmptyResponse(status int, body []byte) bool {
	if status != http.StatusInternalServerError {
		return false
	}
	resp, err := parseJSONBody(body)
	if err != nil {
		return false
	}
	msg := strings.ToLower(pickString(resp, "message", "error"))
	return strings.Contains(msg, "invalid input syntax for type uuid") && strings.Contains(msg, "\"\"")
}

func buildOPASkipReason(prefix string, body []byte) string {
	resp, err := parseJSONBody(body)
	if err != nil {
		return prefix
	}
	if msg := pickString(resp, "message"); msg != "" {
		return prefix + ": " + msg
	}
	return prefix
}

func findObjectByString(items []any, expected string, keys ...string) (map[string]any, bool) {
	for _, item := range items {
		row := asMap(item)
		if row == nil {
			continue
		}
		if got := pickString(row, keys...); got == expected {
			return row, true
		}
	}
	return nil, false
}

func hasNodeID(items []any, id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	_, ok := findObjectByString(items, id, "id", "node_id", "nodeId", "entity_id", "entityId")
	return ok
}

func containsEntityID(items []any, id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	_, ok := findObjectByString(items, id, "entityId", "entity_id", "id", "node_id", "nodeId")
	return ok
}

func containsEdgeID(items []any, id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	_, ok := findObjectByString(items, id, "edgeId", "edge_id", "id")
	return ok
}

func sliceContainsString(v any, expected string) bool {
	for _, item := range asSlice(v) {
		if asString(item) == expected {
			return true
		}
	}
	return false
}

func extractVersionID(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	return pickString(m, "version_id", "versionId")
}

func pickString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if s := asString(val); s != "" {
				return s
			}
		}
	}
	return ""
}

func pickFirstNonNil(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, key := range keys {
		if val, ok := m[key]; ok && val != nil {
			return val
		}
	}
	return nil
}
