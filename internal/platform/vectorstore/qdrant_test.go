package vectorstore_test

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/vectorstore"
)

type qdrantTestServer struct {
	mu    sync.Mutex
	paths map[string]qdrantTestPoint
}

type qdrantRoundTripper struct {
	handler http.Handler
}

type qdrantTestPoint struct {
	Vector  []float64
	Payload map[string]any
}

func newQdrantTestClient(t *testing.T, state *qdrantTestServer) *http.Client {
	t.Helper()
	return &http.Client{
		Transport: qdrantRoundTripper{handler: http.HandlerFunc(state.handle)},
	}
}

func (rt qdrantRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	rt.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func (s *qdrantTestServer) handle(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/collections/kg_vectors/points") {
		http.NotFound(w, r)
		return
	}

	switch {
	case r.Method == http.MethodPut && r.URL.Path == "/collections/kg_vectors/points":
		s.handleUpsert(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/collections/kg_vectors/points/delete":
		s.handleDelete(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/collections/kg_vectors/points/search":
		s.handleSearch(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/collections/kg_vectors/points/scroll":
		s.handleScroll(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/collections/kg_vectors/points/"):
		s.handleGetPoint(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *qdrantTestServer) handleUpsert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Points []struct {
			ID      string         `json:"id"`
			Vector  []float64      `json:"vector"`
			Payload map[string]any `json:"payload"`
		} `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, point := range body.Points {
		s.paths[point.ID] = qdrantTestPoint{
			Vector:  append([]float64(nil), point.Vector...),
			Payload: cloneQdrantMap(point.Payload),
		}
	}
	writeJSON(w, map[string]any{"result": map[string]any{"status": "ok"}})
}

func (s *qdrantTestServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Points []string `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range body.Points {
		delete(s.paths, id)
	}
	writeJSON(w, map[string]any{"result": map[string]any{"status": "ok"}})
}

func (s *qdrantTestServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Vector []float64      `json:"vector"`
		Filter map[string]any `json:"filter"`
		Limit  int            `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]vectorstore.VectorResult, 0, len(s.paths))
	for id, point := range s.paths {
		if point.Payload == nil || !matchesQdrantFilter(point.Payload, body.Filter) {
			continue
		}
		doc := payloadToDocument(id, point)
		results = append(results, vectorstore.VectorResult{
			Document: doc,
			Score:    cosine(body.Vector, point.Vector),
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score > results[j].Score {
			return true
		}
		if results[i].Score < results[j].Score {
			return false
		}
		return results[i].Document.NodeID < results[j].Document.NodeID
	})
	if body.Limit > 0 && len(results) > body.Limit {
		results = results[:body.Limit]
	}
	points := make([]map[string]any, 0, len(results))
	for _, result := range results {
		points = append(points, map[string]any{
			"id":      result.Document.NodeID,
			"score":   result.Score,
			"payload": result.Document,
			"vector":  result.Document.Embedding,
		})
	}
	writeJSON(w, map[string]any{"result": points})
}

func (s *qdrantTestServer) handleScroll(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.paths))
	for id := range s.paths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	points := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		point := s.paths[id]
		doc := payloadToDocument(id, point)
		points = append(points, map[string]any{
			"id":      id,
			"payload": doc,
			"vector":  doc.Embedding,
		})
	}
	writeJSON(w, map[string]any{
		"result": map[string]any{
			"points":           points,
			"next_page_offset": nil,
		},
	})
}

func (s *qdrantTestServer) handleGetPoint(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/collections/kg_vectors/points/")
	s.mu.Lock()
	defer s.mu.Unlock()
	point, ok := s.paths[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	doc := payloadToDocument(id, point)
	writeJSON(w, map[string]any{
		"result": map[string]any{
			"id":      id,
			"payload": doc,
			"vector":  doc.Embedding,
		},
	})
}

func payloadToDocument(id string, point qdrantTestPoint) vectorstore.VectorDocument {
	payload := cloneQdrantMap(point.Payload)
	payload["node_id"] = id
	raw, _ := json.Marshal(payload)
	doc := vectorstore.VectorDocument{}
	_ = json.Unmarshal(raw, &doc)
	if len(doc.Embedding) == 0 {
		doc.Embedding = append([]float64(nil), point.Vector...)
	}
	return doc
}

func matchesQdrantFilter(payload map[string]any, filter map[string]any) bool {
	if len(filter) == 0 {
		return true
	}
	must, _ := filter["must"].([]any)
	for _, clause := range must {
		clauseMap, _ := clause.(map[string]any)
		key, _ := clauseMap["key"].(string)
		match, _ := clauseMap["match"].(map[string]any)
		anyValues, _ := match["any"].([]any)
		if len(anyValues) == 0 {
			return false
		}
		got := payload[key]
		if !filterMatchesAny(got, anyValues) {
			return false
		}
	}
	return true
}

func filterMatchesAny(value any, options []any) bool {
	switch v := value.(type) {
	case []string:
		for _, candidate := range v {
			for _, option := range options {
				if candidate == fmtAny(option) {
					return true
				}
			}
		}
	case []any:
		for _, candidate := range v {
			for _, option := range options {
				if fmtAny(candidate) == fmtAny(option) {
					return true
				}
			}
		}
	case string:
		for _, option := range options {
			if v == fmtAny(option) {
				return true
			}
		}
	default:
		for _, option := range options {
			if fmtAny(value) == fmtAny(option) {
				return true
			}
		}
	}
	return false
}

func fmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func cloneQdrantMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

func TestQdrantVectorAdapterConformance(t *testing.T) {
	state := &qdrantTestServer{paths: map[string]qdrantTestPoint{}}
	client := newQdrantTestClient(t, state)

	adapter := vectorstore.NewQdrantVectorAdapter(vectorstore.QdrantConfig{
		Endpoint:   "http://qdrant.test",
		Collection: "kg_vectors",
		Client:     client,
	})
	conformance.AssertVectorAdapterConformance(t, adapter)
}
