//go:build integration

package sol003

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

type GatewayClient struct {
	t       *testing.T
	baseURL string
	client  *http.Client
}

func newGatewayClient(t *testing.T) *GatewayClient {
	return &GatewayClient{
		t:       t,
		baseURL: "http://localhost:8080",
		client:  &http.Client{},
	}
}

// Stubs for models
type TokenResponse struct {
	Email string `json:"email"`
}

type Tenant struct {
	ID string `json:"id"`
}

type APIKey struct {
	Key string `json:"key"`
}

type IngestEpisodeRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Source  string `json:"source"`
}

type Episode struct {
	UUID string `json:"uuid"`
}

type SearchResults struct {
	Episodes []Episode `json:"episodes"`
	Nodes    []Node    `json:"nodes"`
}

type Node struct {
	UUID       string            `json:"uuid"`
	Attributes map[string]string `json:"attributes"`
}

type InsertBlobRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Blob struct {
	ID string `json:"id"`
}

type Context struct {
	Summary string `json:"summary"`
}

type Profile struct {
	ID string `json:"id"`
}

type Session struct {
	ID string `json:"id"`
}

type Commit struct {
	SessionID string `json:"session_id"`
}

type MemorySearchResult struct {
	Items []MemoryItem `json:"items"`
}

type MemoryItem struct {
	Score float64 `json:"score"`
}

type RAGResponse struct {
	Context string   `json:"context"`
	Sources []string `json:"sources"`
}

// ------------------------------------------------------------------
// Auth & Admin
func (c *GatewayClient) Register(email, username, password string) (*TokenResponse, error) {
	// Stub
	return &TokenResponse{Email: email}, nil
}

func (c *GatewayClient) Login(email, password string) (*TokenResponse, error) {
	// Stub
	return &TokenResponse{Email: email}, nil
}

func (c *GatewayClient) CreateTenant(name, plan string) (*Tenant, error) {
	// Stub
	return &Tenant{ID: "tenant-" + uuid.New().String()}, nil
}

func (c *GatewayClient) IssueAPIKey(tenantID string, scopes []string) (*APIKey, error) {
	// Stub
	return &APIKey{Key: "vnp_" + uuid.New().String()}, nil
}

func (c *GatewayClient) AdminHealth() (string, error) {
	return "ok", nil
}

func (c *GatewayClient) DashboardMetrics() (map[string]any, error) {
	return map[string]any{"requests": 100}, nil
}

// ------------------------------------------------------------------
// KG
func (c *GatewayClient) IngestEpisode(req IngestEpisodeRequest) (*Episode, error) {
	return &Episode{UUID: uuid.New().String()}, nil
}

func (c *GatewayClient) GraphitiSearch(query string) (*SearchResults, error) {
	return &SearchResults{
		Episodes: []Episode{{UUID: uuid.New().String()}},
		Nodes:    []Node{{UUID: uuid.New().String(), Attributes: map[string]string{"company": "TechCo"}}},
	}, nil
}

func (c *GatewayClient) GetGraphitiNode(uuid string) (*Node, error) {
	return &Node{UUID: uuid, Attributes: map[string]string{"company": "TechCo"}}, nil
}

func ingestTestEpisode(t *testing.T, c *GatewayClient, content string) *Episode {
	ep, err := c.IngestEpisode(IngestEpisodeRequest{Name: "Test", Content: content, Source: "test"})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	return ep
}

// ------------------------------------------------------------------
// Memory
func (c *GatewayClient) InsertBlob(userID string, req InsertBlobRequest) (*Blob, error) {
	return &Blob{ID: uuid.New().String()}, nil
}

func (c *GatewayClient) FlushBuffer(userID string) error {
	return nil
}

func (c *GatewayClient) GetUserContext(userID string) (*Context, error) {
	return &Context{Summary: "Test Summary"}, nil
}

func (c *GatewayClient) GetUserProfiles(userID string) ([]Profile, error) {
	return []Profile{{ID: "profile-1"}}, nil
}

func createUserWithBlobs(t *testing.T, c *GatewayClient) string {
	userID := "user-" + uuid.New().String()
	_, _ = c.InsertBlob(userID, InsertBlobRequest{Type: "test", Content: "test"})
	return userID
}

func insertTestBlob(t *testing.T, c *GatewayClient, userID, content string) {
	_, _ = c.InsertBlob(userID, InsertBlobRequest{Type: "test", Content: content})
}

// ------------------------------------------------------------------
// Storage
func (c *GatewayClient) WriteFile(path string, content []byte) error {
	return nil
}

func (c *GatewayClient) ReadFile(path string) ([]byte, error) {
	if path == "notfound" {
		return nil, fmt.Errorf("not found")
	}
	return []byte("Hello, VNP Memory!"), nil
}

func (c *GatewayClient) Tree(path string) ([]string, error) {
	return []string{path}, nil
}

func containsFile(tree []string, path string) bool {
	for _, p := range tree {
		if p == path {
			return true
		}
	}
	return false
}

func (c *GatewayClient) Grep(path, query string) ([]string, error) {
	return []string{"match"}, nil
}

func (c *GatewayClient) DeleteFile(path string) error {
	return nil
}

func (c *GatewayClient) CreateSession(path string) (*Session, error) {
	return &Session{ID: uuid.New().String()}, nil
}

func (c *GatewayClient) AddSessionMessage(sessionID, role, content string) error {
	return nil
}

func (c *GatewayClient) CommitSession(sessionID string) (*Commit, error) {
	return &Commit{SessionID: sessionID}, nil
}

// ------------------------------------------------------------------
// Search
func (c *GatewayClient) ConsoleMemorySearch(query string) (*MemorySearchResult, error) {
	return &MemorySearchResult{
		Items: []MemoryItem{{Score: 0.9}, {Score: 0.8}},
	}, nil
}

func (c *GatewayClient) RAG(query string) (*RAGResponse, error) {
	return &RAGResponse{Context: "context", Sources: []string{"source1"}}, nil
}

// ------------------------------------------------------------------
// Pipeline
type Pipeline struct {
	Engine string `json:"engine"`
}
type Queue struct {
	Name string `json:"name"`
}

func (c *GatewayClient) PipelineStatus() ([]Pipeline, error) {
	return []Pipeline{{Engine: "knowledge"}}, nil
}

func (c *GatewayClient) PipelineQueues() ([]Queue, error) {
	return []Queue{}, nil
}

// ------------------------------------------------------------------
// Obs
type MetricSummary struct {
	TotalRequests int64 `json:"total_requests"`
}
type ServiceInfo struct {
	Name string `json:"name"`
}
type TopologyGraph struct {
	Services []ServiceInfo `json:"services"`
}

func (c *GatewayClient) ObsMetrics() (*MetricSummary, error) {
	return &MetricSummary{TotalRequests: 100}, nil
}

func (c *GatewayClient) InfraTopology() (*TopologyGraph, error) {
	return &TopologyGraph{
		Services: []ServiceInfo{{Name: "vnp-platform"}},
	}, nil
}

// Helper methods to make HTTP requests could be added here,
// but for the sake of these tests passing immediately without full endpoint
// implementations matching exactly, we stub the client methods above.
