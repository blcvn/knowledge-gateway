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
)

type stepResult struct {
	Name     string
	Duration time.Duration
	Err      error
}

type tester struct {
	baseURL       string
	client        *http.Client
	verbose       bool
	failFast      bool
	syncOPAPolicy bool
	opaURL        string
	authAppIDFlag string

	createdAppID string
	authAppID    string
	apiKey       string
	keyHash      string
	namespace    string

	entityType   string
	relationType string
	node1ID      string
	node2ID      string
	ruleID       string
	policyID     string
	viewID       string
	overlay1ID   string
	overlay2ID   string
	versionFrom  string
	versionTo    string

	results []stepResult
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8000", "KGS base URL")
	timeout := flag.Duration("timeout", 20*time.Second, "HTTP timeout per request")
	verbose := flag.Bool("verbose", false, "Print response body for each step")
	failFast := flag.Bool("fail-fast", false, "Stop at first failed step")
	authAppID := flag.String("auth-app-id", "", "Existing app_id to issue API key for auth-protected APIs (optional)")
	syncOPAPolicy := flag.Bool("sync-opa-policy", true, "Push temporary allow policy to OPA for auth app_id")
	opaURL := flag.String("opa-url", "http://localhost:8181", "OPA base URL")
	flag.Parse()

	t := &tester{
		baseURL:       strings.TrimRight(*baseURL, "/"),
		client:        &http.Client{Timeout: *timeout},
		verbose:       *verbose,
		failFast:      *failFast,
		syncOPAPolicy: *syncOPAPolicy,
		opaURL:        strings.TrimRight(*opaURL, "/"),
		authAppIDFlag: strings.TrimSpace(*authAppID),
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
		resp, err := t.doJSON(http.MethodPost, "/v1/apps", map[string]any{
			"app_name":    "KGS API Tester " + suffix,
			"description": "auto smoke test",
			"owner":       "qa-bot",
		}, false, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.createdAppID = pickString(resp, "app_id", "appId")
		if t.createdAppID == "" {
			return fmt.Errorf("missing app_id in response")
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
		return nil
	})

	t.runStep("GET /v1/apps/{app_id} (GetApp)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/apps/"+url.PathEscape(t.createdAppID), nil, false, nil, http.StatusOK)
		return err
	})

	t.authAppID = t.createdAppID
	if t.authAppIDFlag != "" {
		t.authAppID = t.authAppIDFlag
		t.runStep("GET /v1/apps/{auth_app_id} (Validate auth app)", func() error {
			_, err := t.doJSON(http.MethodGet, "/v1/apps/"+url.PathEscape(t.authAppID), nil, false, nil, http.StatusOK)
			return err
		})
	}

	t.runStep("POST /v1/apps/{app_id}/keys (IssueApiKey)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/apps/"+url.PathEscape(t.authAppID)+"/keys", map[string]any{
			"name":        "api-tester-key-" + suffix,
			"scopes":      "all",
			"ttl_seconds": 3600,
		}, false, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.apiKey = pickString(resp, "api_key", "apiKey")
		t.keyHash = pickString(resp, "key_hash", "keyHash")
		if t.apiKey == "" || t.keyHash == "" {
			return fmt.Errorf("missing api_key or key_hash in response")
		}
		t.namespace = fmt.Sprintf("graph/%s/default", t.authAppID)
		return nil
	})

	if t.syncOPAPolicy {
		t.runStep("Setup OPA allow policy (optional)", func() error {
			return t.pushOPAPolicyForApp(t.authAppID)
		})
	}

	t.runStep("POST /v1/policies (CreatePolicy)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/policies", map[string]any{
			"name":        "Allow " + suffix,
			"description": "api-tester policy",
			"rego_content": fmt.Sprintf(
				"package kgs\nimport rego.v1\n\ndefault allow := false\n\nallow if {\n  input.app_id == %q\n}\n",
				t.authAppID,
			),
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.policyID = pickString(resp, "id")
		if t.policyID == "" {
			return fmt.Errorf("missing policy id")
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
		return nil
	})

	t.runStep("GET /v1/policies/{id} (GetPolicy)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/policies/"+url.PathEscape(t.policyID), nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/ontology/entities (CreateEntityType)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/ontology/entities", map[string]any{
			"name":        t.entityType,
			"description": "entity type for api tester",
			"schema":      mustJSONString(map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}, "priority": map[string]any{"type": "string"}}}),
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if asString(resp["name"]) == "" {
			return fmt.Errorf("missing entity type name in response")
		}
		return nil
	})

	t.runStep("POST /v1/ontology/relations (CreateRelationType)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/ontology/relations", map[string]any{
			"name":              t.relationType,
			"description":       "relation type for api tester",
			"properties_schema": mustJSONString(map[string]any{"type": "object", "properties": map[string]any{"strength": map[string]any{"type": "number"}}}),
			"source_types":      []string{t.entityType},
			"target_types":      []string{t.entityType},
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if asString(resp["name"]) == "" {
			return fmt.Errorf("missing relation type name in response")
		}
		return nil
	})

	t.runStep("GET /v1/ontology/entities (ListEntityTypes)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/ontology/entities", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(asSlice(resp["entities"])) == 0 {
			return fmt.Errorf("entity types list is empty")
		}
		return nil
	})

	t.runStep("GET /v1/ontology/relations (ListRelationTypes)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/ontology/relations", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(asSlice(resp["relations"])) == 0 {
			return fmt.Errorf("relation types list is empty")
		}
		return nil
	})

	t.runStep("POST /v1/rules (CreateRule)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/rules", map[string]any{
			"name":         "rule-" + suffix,
			"description":  "api tester rule",
			"trigger_type": "SCHEDULED",
			"cron":         "0 */6 * * *",
			"cypher_query": "MATCH (n) RETURN n LIMIT 1",
			"action":       "LOG",
			"payload_json": "{}",
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.ruleID = pickString(resp, "id")
		if t.ruleID == "" {
			return fmt.Errorf("missing rule id")
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
		return nil
	})

	t.runStep("GET /v1/rules/{id} (GetRule)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/rules/"+url.PathEscape(t.ruleID), nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/nodes (CreateNode #1)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label":           t.entityType,
			"properties_json": mustJSONString(map[string]any{"name": "REQ-1-" + suffix, "priority": "HIGH", "domain": "payment"}),
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.node1ID = pickString(resp, "node_id", "nodeId")
		if t.node1ID == "" {
			return fmt.Errorf("missing node_id for node1")
		}
		return nil
	})

	t.runStep("POST /v1/graph/nodes (CreateNode #2)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label":           t.entityType,
			"properties_json": mustJSONString(map[string]any{"name": "REQ-2-" + suffix, "priority": "MEDIUM", "domain": "payment"}),
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.node2ID = pickString(resp, "node_id", "nodeId")
		if t.node2ID == "" {
			return fmt.Errorf("missing node_id for node2")
		}
		return nil
	})

	t.runStep("GET /v1/graph/nodes/{node_id} (GetNode)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node1ID), nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/edges (CreateEdge)", func() error {
		_, err := t.doJSON(http.MethodPost, "/v1/graph/edges", map[string]any{
			"source_node_id":  t.node1ID,
			"target_node_id":  t.node2ID,
			"relation_type":   t.relationType,
			"properties_json": mustJSONString(map[string]any{"strength": 0.9}),
		}, true, nil, http.StatusOK)
		return err
	})

	t.runStep("GET /v1/graph/nodes/{node_id}/context (GetContext)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node1ID)+"/context?depth=2&direction=BOTH&page_size=20", nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("GET /v1/graph/nodes/{node_id}/impact (GetImpact)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node1ID)+"/impact?max_depth=3&page_size=20", nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("GET /v1/graph/nodes/{node_id}/coverage (GetCoverage)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/graph/nodes/"+url.PathEscape(t.node2ID)+"/coverage?max_depth=3&page_size=20", nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/subgraph (GetSubgraph)", func() error {
		_, err := t.doJSON(http.MethodPost, "/v1/graph/subgraph", map[string]any{
			"node_ids": []string{t.node1ID, t.node2ID},
		}, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/entities/batch (BatchUpsertEntities)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/entities/batch", map[string]any{
			"entities": []map[string]any{
				{
					"label":           t.entityType,
					"properties_json": mustJSONString(map[string]any{"name": "REQ-B1-" + suffix, "priority": "LOW"}),
				},
				{
					"label":           t.entityType,
					"properties_json": mustJSONString(map[string]any{"name": "REQ-B2-" + suffix, "priority": "LOW"}),
				},
			},
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if resp["created"] == nil {
			return fmt.Errorf("missing created field")
		}
		return nil
	})

	t.runStep("POST /v1/graph/search/hybrid (HybridSearch)", func() error {
		_, err := t.doJSON(http.MethodPost, "/v1/graph/search/hybrid", map[string]any{
			"query":          "REQ " + suffix,
			"top_k":          10,
			"alpha":          0.6,
			"beta":           0.2,
			"entity_types":   []string{t.entityType},
			"min_confidence": 0.0,
		}, true, nil, http.StatusOK)
		return err
	})

	t.runStep("GET /v1/graph/coverage/{domain} (GetCoverageReport)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/graph/coverage/payment", nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/traceability (GetTraceabilityMatrix)", func() error {
		_, err := t.doJSON(http.MethodPost, "/v1/graph/traceability", map[string]any{
			"source_types": []string{t.entityType},
			"target_types": []string{t.entityType},
			"max_hops":     3,
		}, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/views (CreateViewDefinition)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/views", map[string]any{
			"role_name":            "reader-" + suffix,
			"allowed_entity_types": []string{t.entityType},
			"allowed_fields":       []string{"id", "name", "priority"},
			"pii_mask_fields":      []string{"email", "phone"},
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		view, ok := resp["view"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing view object in response")
		}
		t.viewID = pickString(view, "view_id", "viewId")
		if t.viewID == "" {
			return fmt.Errorf("missing view_id")
		}
		return nil
	})

	t.runStep("GET /v1/graph/views/{view_id} (GetViewDefinition)", func() error {
		_, err := t.doJSON(http.MethodGet, "/v1/graph/views/"+url.PathEscape(t.viewID), nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("GET /v1/graph/views (ListViewDefinitions)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/views", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(asSlice(resp["views"])) == 0 {
			return fmt.Errorf("views list is empty")
		}
		return nil
	})

	t.runStep("DELETE /v1/graph/views/{view_id} (DeleteViewDefinition)", func() error {
		_, err := t.doJSON(http.MethodDelete, "/v1/graph/views/"+url.PathEscape(t.viewID), nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/overlays (CreateOverlay #1)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/overlays", map[string]any{
			"session_id":   "session-" + suffix,
			"base_version": "current",
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.overlay1ID = pickString(resp, "overlay_id", "overlayId")
		if t.overlay1ID == "" {
			return fmt.Errorf("missing overlay_id")
		}
		return nil
	})

	t.runStep("POST /v1/graph/nodes (CreateNode in Overlay)", func() error {
		_, err := t.doJSON(http.MethodPost, "/v1/graph/nodes", map[string]any{
			"label": t.entityType,
			"properties_json": mustJSONString(map[string]any{
				"name":       "REQ-OVERLAY-" + suffix,
				"priority":   "HIGH",
				"overlay_id": t.overlay1ID,
			}),
		}, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/overlays/{overlay_id}/commit (CommitOverlay)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/overlays/"+url.PathEscape(t.overlay1ID)+"/commit", map[string]any{
			"overlay_id":      t.overlay1ID,
			"conflict_policy": "KEEP_OVERLAY",
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		_ = pickString(resp, "new_version_id", "newVersionId")
		return nil
	})

	t.runStep("GET /v1/graph/versions (ListVersions)", func() error {
		resp, err := t.doJSON(http.MethodGet, "/v1/graph/versions", nil, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		versions := asSlice(resp["versions"])
		if len(versions) == 0 {
			return fmt.Errorf("no versions returned")
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
		return nil
	})

	t.runStep("GET /v1/graph/versions/diff (DiffVersions)", func() error {
		path := fmt.Sprintf(
			"/v1/graph/versions/diff?from_version_id=%s&to_version_id=%s",
			url.QueryEscape(t.versionFrom),
			url.QueryEscape(t.versionTo),
		)
		_, err := t.doJSON(http.MethodGet, path, nil, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/versions/{version_id}/rollback (RollbackVersion)", func() error {
		_, err := t.doJSON(http.MethodPost, "/v1/graph/versions/"+url.PathEscape(t.versionFrom)+"/rollback", map[string]any{
			"version_id": t.versionFrom,
			"reason":     "api tester rollback check",
		}, true, nil, http.StatusOK)
		return err
	})

	t.runStep("POST /v1/graph/overlays (CreateOverlay #2)", func() error {
		resp, err := t.doJSON(http.MethodPost, "/v1/graph/overlays", map[string]any{
			"session_id":   "discard-session-" + suffix,
			"base_version": "current",
		}, true, nil, http.StatusOK)
		if err != nil {
			return err
		}
		t.overlay2ID = pickString(resp, "overlay_id", "overlayId")
		if t.overlay2ID == "" {
			return fmt.Errorf("missing overlay_id for discard")
		}
		return nil
	})

	t.runStep("DELETE /v1/graph/overlays/{overlay_id} (DiscardOverlay)", func() error {
		_, err := t.doJSON(http.MethodDelete, "/v1/graph/overlays/"+url.PathEscape(t.overlay2ID), nil, true, nil, http.StatusOK)
		return err
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
		if b, ok := resp["success"].(bool); !ok || !b {
			return fmt.Errorf("revoke did not return success=true")
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
	t.results = append(t.results, res)

	if err != nil {
		fmt.Printf("FAIL  %-78s (%s)\n", name, res.Duration.Truncate(time.Millisecond))
		fmt.Printf("      %v\n", err)
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
		if t.namespace != "" {
			req.Header.Set("X-KG-Namespace", t.namespace)
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
	policy := fmt.Sprintf("package kgs\nimport rego.v1\n\ndefault allow := false\n\nallow if {\n  input.app_id == %q\n}\n", appID)
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
	for _, r := range t.results {
		if r.Err == nil {
			passed++
		} else {
			failed++
		}
	}
	fmt.Println()
	fmt.Println("==============================================================")
	fmt.Printf("Summary: total=%d, passed=%d, failed=%d\n", total, passed, failed)
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
