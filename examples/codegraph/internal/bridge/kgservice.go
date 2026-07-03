package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type KGServiceClient interface {
	Ping(context.Context) error
	CreateNode(context.Context, NodeRequest) (NodeCreateResponse, error)
	CreateNodesBulk(context.Context, NodeBulkCreateRequest) (NodeBulkCreateResponse, error)
	UpdateNode(context.Context, string, NodeUpdateRequest) (NodeUpdateResponse, error)
	DeleteNode(context.Context, string) error
	DeleteNodeWithVersion(context.Context, string, string) error
	DeleteNodesByExternalRefPrefix(context.Context, string) error
	DeleteNodesByExternalRefPrefixWithVersion(context.Context, string, string) error
	CreateRelationship(context.Context, RelationshipRequest) (RelationshipCreateResponse, error)
	CreateRelationshipsBulk(context.Context, RelationshipBulkCreateRequest) (RelationshipBulkCreateResponse, error)
	DeleteRelationshipsBulk(context.Context, RelationshipBulkDeleteRequest) (RelationshipBulkDeleteResponse, error)
	OpenSyncSession(context.Context, OpenSyncSessionRequest) (SyncSessionResponse, error)
	CommitSyncSession(context.Context, string) error
	AbandonSyncSession(context.Context, string) error
	SemanticSearch(context.Context, SemanticSearchRequest) (SemanticSearchResponse, error)
	FullTextSearch(context.Context, FullTextSearchRequest) (FullTextSearchResponse, error)
	TemplateQuery(context.Context, string, string, map[string]any) (TemplateExecutionResponse, error)
}

type KGServiceAdapter = KGServiceClient

func NewClient(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.KGServiceURL, "/"),
		apiKey:  cfg.KGAPIKey,
		http: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("healthz returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
}

func (c *Client) CreateNode(ctx context.Context, req NodeRequest) (NodeCreateResponse, error) {
	var resp NodeCreateResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/write/nodes", req, &resp); err != nil {
		return NodeCreateResponse{}, err
	}
	return resp, nil
}

func (c *Client) CreateNodesBulk(ctx context.Context, req NodeBulkCreateRequest) (NodeBulkCreateResponse, error) {
	var resp NodeBulkCreateResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/write/nodes/bulk", req, &resp); err != nil {
		return NodeBulkCreateResponse{}, err
	}
	return resp, nil
}

func (c *Client) UpdateNode(ctx context.Context, nodeID string, req NodeUpdateRequest) (NodeUpdateResponse, error) {
	var resp NodeUpdateResponse
	if err := c.doJSON(ctx, http.MethodPut, "/v1/kg/write/nodes/"+url.PathEscape(nodeID), req, &resp); err != nil {
		return NodeUpdateResponse{}, err
	}
	return resp, nil
}

func (c *Client) DeleteNode(ctx context.Context, nodeID string) error {
	var resp map[string]any
	if err := c.doJSON(ctx, http.MethodDelete, "/v1/kg/write/nodes/"+url.PathEscape(nodeID), nil, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteNodeWithVersion(ctx context.Context, nodeID, graphVersionID string) error {
	var resp map[string]any
	reqPath := "/v1/kg/write/nodes/" + url.PathEscape(nodeID)
	if strings.TrimSpace(graphVersionID) != "" {
		reqPath += "?graph_version_id=" + url.QueryEscape(graphVersionID)
	}
	if err := c.doJSON(ctx, http.MethodDelete, reqPath, nil, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteNodesByExternalRefPrefix(ctx context.Context, prefix string) error {
	var resp map[string]any
	if err := c.doJSON(ctx, http.MethodDelete, "/v1/kg/write/nodes:by-external-ref-prefix", map[string]any{
		"external_ref_prefix": prefix,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteNodesByExternalRefPrefixWithVersion(ctx context.Context, prefix, graphVersionID string) error {
	var resp map[string]any
	reqPath := "/v1/kg/write/nodes:by-external-ref-prefix"
	if strings.TrimSpace(graphVersionID) != "" {
		reqPath += "?graph_version_id=" + url.QueryEscape(graphVersionID)
	}
	if err := c.doJSON(ctx, http.MethodDelete, reqPath, map[string]any{
		"external_ref_prefix": prefix,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateRelationship(ctx context.Context, req RelationshipRequest) (RelationshipCreateResponse, error) {
	var resp RelationshipCreateResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/write/relationships", req, &resp); err != nil {
		return RelationshipCreateResponse{}, err
	}
	return resp, nil
}

func (c *Client) CreateRelationshipsBulk(ctx context.Context, req RelationshipBulkCreateRequest) (RelationshipBulkCreateResponse, error) {
	var resp RelationshipBulkCreateResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/write/relationships/bulk", req, &resp); err != nil {
		return RelationshipBulkCreateResponse{}, err
	}
	return resp, nil
}

func (c *Client) DeleteRelationshipsBulk(ctx context.Context, req RelationshipBulkDeleteRequest) (RelationshipBulkDeleteResponse, error) {
	var resp RelationshipBulkDeleteResponse
	if err := c.doJSON(ctx, http.MethodDelete, "/v1/kg/write/relationships/bulk", req, &resp); err != nil {
		return RelationshipBulkDeleteResponse{}, err
	}
	return resp, nil
}

func (c *Client) OpenSyncSession(ctx context.Context, req OpenSyncSessionRequest) (SyncSessionResponse, error) {
	var resp SyncSessionResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/write/sync-sessions", req, &resp); err != nil {
		return SyncSessionResponse{}, err
	}
	return resp, nil
}

func (c *Client) CommitSyncSession(ctx context.Context, sessionID string) error {
	var resp map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/write/sync-sessions/"+url.PathEscape(sessionID)+"/commit", nil, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) AbandonSyncSession(ctx context.Context, sessionID string) error {
	var resp map[string]any
	if err := c.doJSON(ctx, http.MethodDelete, "/v1/kg/write/sync-sessions/"+url.PathEscape(sessionID), nil, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) SemanticSearch(ctx context.Context, req SemanticSearchRequest) (SemanticSearchResponse, error) {
	var resp SemanticSearchResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/search/hybrid", req, &resp); err != nil {
		return SemanticSearchResponse{}, err
	}
	return resp, nil
}

func (c *Client) FullTextSearch(ctx context.Context, req FullTextSearchRequest) (FullTextSearchResponse, error) {
	var resp FullTextSearchResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/kg/search/fulltext", req, &resp); err != nil {
		return FullTextSearchResponse{}, err
	}
	return resp, nil
}

func (c *Client) TemplateQuery(ctx context.Context, domainID, templateName string, params map[string]any) (TemplateExecutionResponse, error) {
	body := map[string]any{"params": params}
	var resp TemplateExecutionResponse
	reqPath := path.Join("/v1/kg/read/template", domainID, templateName)
	if err := c.doJSON(ctx, http.MethodPost, reqPath, body, &resp); err != nil {
		return TemplateExecutionResponse{}, err
	}
	return resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, relPath string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+relPath, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseHTTPError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response from %s %s: %w", method, relPath, err)
	}
	return nil
}

func parseHTTPError(statusCode int, body []byte) error {
	var envelope struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Message != "" {
		return fmt.Errorf("kg-service %d %s: %s", statusCode, envelope.Error.Code, envelope.Error.Message)
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = strconv.Itoa(statusCode)
	}
	return errors.New("kg-service " + strconv.Itoa(statusCode) + ": " + msg)
}

type NodeRequest struct {
	DomainID    string         `json:"domain_id"`
	NodeType    string         `json:"node_type"`
	Properties  map[string]any `json:"properties"`
	Visibility  string         `json:"visibility"`
	ExternalRef string         `json:"external_ref"`
}

type NodeUpdateRequest struct {
	Properties     map[string]any `json:"properties,omitempty"`
	Visibility     string         `json:"visibility,omitempty"`
	ExternalRef    string         `json:"external_ref,omitempty"`
	GraphVersionID string         `json:"graph_version_id,omitempty"`
}

type RelationshipRequest struct {
	RelType    string         `json:"rel_type"`
	FromNodeID string         `json:"from_node_id"`
	ToNodeID   string         `json:"to_node_id"`
	DomainID   string         `json:"domain_id"`
	Properties map[string]any `json:"properties"`
}

type NodeCreateResponse struct {
	NodeID        string `json:"node_id"`
	DomainVersion int    `json:"domain_version"`
	Status        string `json:"status"`
	SyncETAMs     int    `json:"sync_eta_ms"`
}

type BulkItemError struct {
	Index          int    `json:"index"`
	ExternalRef    string `json:"external_ref,omitempty"`
	RelationshipID string `json:"relationship_id,omitempty"`
	Error          string `json:"error"`
}

type NodeBulkCreateRequest struct {
	Nodes          []NodeRequest `json:"nodes"`
	GraphVersionID string        `json:"graph_version_id,omitempty"`
}

type NodeBulkCreateResponse struct {
	Succeeded []NodeCreateResponse `json:"succeeded"`
	Failed    []BulkItemError      `json:"failed"`
}

type RelationshipBulkCreateRequest struct {
	Relationships  []RelationshipRequest `json:"relationships"`
	GraphVersionID string                `json:"graph_version_id,omitempty"`
}

type RelationshipBulkCreateResponse struct {
	Succeeded []RelationshipCreateResponse `json:"succeeded"`
	Failed    []BulkItemError              `json:"failed"`
}

type RelationshipBulkDeleteRequest struct {
	RelationshipIDs []string `json:"relationship_ids"`
	GraphVersionID  string   `json:"graph_version_id,omitempty"`
}

type RelationshipBulkDeleteResponse struct {
	RelationshipIDs []string `json:"relationship_ids"`
	Count           int      `json:"count"`
}

type OpenSyncSessionRequest struct {
	DomainID   string `json:"domain_id"`
	GraphScope string `json:"graph_scope"`
}

type SyncSessionResponse struct {
	SessionID          string `json:"session_id"`
	GraphVersionID     string `json:"graph_version_id"`
	GraphIdentifierID  string `json:"graph_identifier_id"`
	GraphVersionNumber int64  `json:"graph_version_number"`
}

type NodeUpdateResponse struct {
	NodeID        string `json:"node_id"`
	DomainVersion int    `json:"domain_version"`
	Status        string `json:"status"`
}

type RelationshipCreateResponse struct {
	RelationshipID string `json:"relationship_id"`
	Status         string `json:"status"`
}

type SearchResult struct {
	NodeID        string         `json:"node_id"`
	NodeType      string         `json:"node_type"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	ACLVisibleTo  []string       `json:"acl_visible_to"`
	IsDeleted     bool           `json:"is_deleted"`
	StatusValue   string         `json:"status_value,omitempty"`
	Content       string         `json:"content"`
	Score         float64        `json:"score"`
	DomainProps   map[string]any `json:"domain_props"`
}

type SemanticSearchRequest struct {
	Query          string   `json:"query"`
	DomainIDs      []string `json:"domain_ids"`
	TopK           int      `json:"top_k"`
	FTSOperator    string   `json:"fts_operator,omitempty"`
	SemanticWeight float64  `json:"semantic_weight,omitempty"`
}

type SemanticSearchResponse struct {
	Results      []SearchResult `json:"results"`
	SearchTimeMs int            `json:"search_time_ms"`
}

type FullTextSearchRequest struct {
	Query     string   `json:"query"`
	DomainIDs []string `json:"domain_ids"`
	TopK      int      `json:"top_k"`
	Mode      string   `json:"mode"`
	Fields    []string `json:"fields,omitempty"`
}

type FullTextSearchResponse struct {
	Results      []SearchResult `json:"results"`
	SearchTimeMs int            `json:"search_time_ms"`
}

type TemplateExecutionResponse struct {
	Results     []map[string]any `json:"results"`
	QueryTimeMs int              `json:"query_time_ms"`
}
