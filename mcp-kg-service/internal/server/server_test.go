package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/repository"
	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestKGHandlers(t *testing.T) {
	srv := newTestServer(t)

	t.Run("list documents", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/documents", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var summaries []repository.DocumentSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(summaries) != 1 {
			t.Fatalf("len(summaries) = %d, want 1", len(summaries))
		}
		if summaries[0].DocumentID != "doc-1" {
			t.Fatalf("document_id = %q, want doc-1", summaries[0].DocumentID)
		}
		if summaries[0].NodeCount != 4 {
			t.Fatalf("node_count = %d, want 4", summaries[0].NodeCount)
		}
		if summaries[0].EdgeCount != 3 {
			t.Fatalf("edge_count = %d, want 3", summaries[0].EdgeCount)
		}
	})

	t.Run("document subgraph", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/subgraph?max_nodes=10", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload struct {
			Nodes []repository.RequirementNode `json:"nodes"`
			Edges []repository.DependencyEdge  `json:"edges"`
			Stats struct {
				NodeCount int  `json:"node_count"`
				EdgeCount int  `json:"edge_count"`
				Truncated bool `json:"truncated"`
			} `json:"stats"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if payload.Stats.NodeCount != 4 || payload.Stats.EdgeCount != 3 || payload.Stats.Truncated {
			t.Fatalf("unexpected stats: %+v", payload.Stats)
		}
	})

	t.Run("feature subgraph", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/feature/F-001/subgraph", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload struct {
			Nodes []repository.RequirementNode `json:"nodes"`
			Edges []repository.DependencyEdge  `json:"edges"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(payload.Nodes) != 4 {
			t.Fatalf("len(nodes) = %d, want 4", len(payload.Nodes))
		}
		if len(payload.Edges) != 3 {
			t.Fatalf("len(edges) = %d, want 3", len(payload.Edges))
		}
	})

	t.Run("search features", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/features/search?q=login&document_id=doc-1", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var results []service.FeatureSearchResult
		if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
		if results[0].FeatureID != "F-001" {
			t.Fatalf("feature_id = %q, want F-001", results[0].FeatureID)
		}
	})

	t.Run("feature detail", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/feature/F-001/detail", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var detail service.FeatureDetail
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if detail.FeatureID != "F-001" {
			t.Fatalf("feature_id = %q, want F-001", detail.FeatureID)
		}
		if len(detail.FeatureDetails) != 1 {
			t.Fatalf("len(feature_details) = %d, want 1", len(detail.FeatureDetails))
		}
		if len(detail.BusinessRules) != 1 {
			t.Fatalf("len(business_rules) = %d, want 1", len(detail.BusinessRules))
		}
		if len(detail.UIScreens) != 1 {
			t.Fatalf("len(ui_screens) = %d, want 1", len(detail.UIScreens))
		}
	})

	t.Run("feature detail not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/feature/F-404/detail", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("feature subgraph not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/feature/F-404/subgraph", nil)

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("upsert nodes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"nodes":[{"id":"rule-2","type":"BUSINESS_RULE","properties":{"severity":"high"}}],"edges":[{"source":"detail-1","type":"CONTAINS","target":"rule-2"}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/kg/documents/doc-1/nodes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/subgraph?max_nodes=10", nil)
		srv.Handler.ServeHTTP(rec2, req2)

		var payload struct {
			Stats struct {
				NodeCount int `json:"node_count"`
				EdgeCount int `json:"edge_count"`
			} `json:"stats"`
		}
		if err := json.Unmarshal(rec2.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if payload.Stats.NodeCount != 5 || payload.Stats.EdgeCount != 4 {
			t.Fatalf("unexpected stats after upsert: %+v", payload.Stats)
		}
	})

	t.Run("save graph replaces snapshot", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"graph":{"nodes":[{"id":"feature-2","reference_id":"F-002","type":"FEATURE","summary":"Transfer","properties":{}}],"edges":[],"metadata":{"source":"test"}}}`
		req := httptest.NewRequest(http.MethodPut, "/v1/kg/documents/doc-1/graph", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/v1/kg/documents/doc-1/subgraph?max_nodes=10", nil)
		srv.Handler.ServeHTTP(rec2, req2)

		var payload struct {
			Stats struct {
				NodeCount int `json:"node_count"`
				EdgeCount int `json:"edge_count"`
			} `json:"stats"`
		}
		if err := json.Unmarshal(rec2.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if payload.Stats.NodeCount != 1 || payload.Stats.EdgeCount != 0 {
			t.Fatalf("unexpected stats after save graph: %+v", payload.Stats)
		}
	})

	t.Run("save document artifact", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"doc_kind":"product_index","content":"# Product Index"}`
		req := httptest.NewRequest(http.MethodPut, "/v1/kg/documents/doc-1/document", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var payload service.SaveDocumentResult
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal save document response: %v", err)
		}
		if payload.DocKind != "product_index" || !payload.Saved {
			t.Fatalf("unexpected save document payload: %+v", payload)
		}
	})
}

func newTestServer(t *testing.T) *http.Server {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&repository.RequirementNodeModel{}, &repository.DependencyEdgeModel{}, &repository.DocumentArtifactModel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	nodes := []repository.RequirementNodeModel{
		{ID: "feature-1", DocumentID: "doc-1", ReferenceID: "F-001", Type: "FEATURE", Summary: "Login", Description: "User login feature", CreatedAt: now, UpdatedAt: now},
		{ID: "detail-1", DocumentID: "doc-1", Type: "FEATURE_DETAIL", Summary: "Login detail", Metadata: []byte(`{"feature_id":"F-001"}`), CreatedAt: now, UpdatedAt: now},
		{ID: "br-1", DocumentID: "doc-1", Type: "BUSINESS_RULE", Summary: "Password length", CreatedAt: now, UpdatedAt: now},
		{ID: "ui-1", DocumentID: "doc-1", Type: "UI_SCREEN", Summary: "Login screen", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	edges := []repository.DependencyEdgeModel{
		{ID: "edge-1", DocumentID: "doc-1", SourceID: "feature-1", TargetID: "detail-1", Type: "CONTAINS", CreatedAt: now},
		{ID: "edge-2", DocumentID: "doc-1", SourceID: "detail-1", TargetID: "br-1", Type: "CONTAINS", CreatedAt: now},
		{ID: "edge-3", DocumentID: "doc-1", SourceID: "detail-1", TargetID: "ui-1", Type: "CONTAINS", CreatedAt: now},
	}
	if err := db.Create(&edges).Error; err != nil {
		t.Fatalf("seed edges: %v", err)
	}

	repo := repository.New(db)
	svc := service.New(repo)
	return New(0, svc, time.Second, time.Second)
}
