package search

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/vector"
	"kg-service/internal/searchprofile"
	"kg-service/internal/write"
)

type ProjectionStore interface {
	ListNodes() []write.NodeRecord
}

type VectorIndex interface {
	Search(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool) ([]SearchResult, error)
	RagSearch(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool) ([]SearchResult, error)
}

type ProjectionVectorIndex struct {
	store    ProjectionStore
	ontology OntologyResolver
	provider vector.EmbeddingProvider
	now      func() time.Time
}

func NewProjectionVectorIndex(store ProjectionStore, ontology OntologyResolver, provider vector.EmbeddingProvider) ProjectionVectorIndex {
	return ProjectionVectorIndex{
		store:    store,
		ontology: ontology,
		provider: provider,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (i ProjectionVectorIndex) Search(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool) ([]SearchResult, error) {
	return i.execute(actor, req, visibility, domainSet, statusConfigs, allHaveStatus, false)
}

func (i ProjectionVectorIndex) RagSearch(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool) ([]SearchResult, error) {
	return i.execute(actor, req, visibility, domainSet, statusConfigs, allHaveStatus, true)
}

func (i ProjectionVectorIndex) execute(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool, rag bool) ([]SearchResult, error) {
	if i.provider == nil {
		p := vector.NewDeterministicProvider(8)
		i.provider = p
	}
	queryEmbedding, _ := i.provider.Embed(context.Background(), req.Query)
	results := make([]SearchResult, 0)
	for _, node := range i.store.ListNodes() {
		if node.IsDeleted {
			continue
		}
		if len(domainSet) > 0 {
			if _, ok := domainSet[node.DomainID]; !ok {
				continue
			}
		} else if _, err := i.ontology.GetVisibleDomain(actor, node.DomainID); err != nil {
			continue
		}
		if _, ok := visibility[nodeKey(node.OwnerTenantID, node.OwnerAppID)]; !ok {
			continue
		}

		cfg := statusConfigs[node.DomainID]
		if len(domainSet) > 0 && allHaveStatus && cfg != nil && node.StatusValue != "" && !slices.Contains(cfg.ValidStatusValues, node.StatusValue) {
			continue
		}

		content := nodeContent(node)
		if !rag && !matchesQuery(node, req.Query) {
			continue
		}

		score := scoreNodeWithEmbedding(content, queryEmbedding, req.Query, rag)
		results = append(results, SearchResult{
			NodeID:         node.ID,
			NodeType:       node.NodeType,
			DomainID:       node.DomainID,
			OwnerTenantID:  node.OwnerTenantID,
			OwnerAppID:     node.OwnerAppID,
			ACLVisibleTo:   nodeACLVisibleTo(node),
			IsDeleted:      node.IsDeleted,
			StatusValue:    node.StatusValue,
			AuthorityScore: authorityScoreFor(node, cfg),
			DomainProps:    node.Properties,
			Content:        content,
			Score:          score,
			CreatedAt:      node.CreatedAt,
		})
	}

	slices.SortFunc(results, func(a, b SearchResult) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		if a.NodeID < b.NodeID {
			return -1
		}
		if a.NodeID > b.NodeID {
			return 1
		}
		return 0
	})

	if len(results) > req.TopK {
		results = results[:req.TopK]
	}
	return results, nil
}

func scoreNodeWithEmbedding(content string, queryEmbedding []float64, query string, rag bool) float64 {
	nodeEmbedding, _ := vector.NewDeterministicProvider(len(queryEmbedding)).Embed(context.Background(), content)
	sim := cosineSimilarity(queryEmbedding, nodeEmbedding)
	if rag {
		return math.Min(1.0, sim+0.05*tokenOverlap(content, query))
	}
	return sim
}

func tokenOverlap(content, query string) float64 {
	queryTokens := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(queryTokens) == 0 {
		return 0
	}
	matched := 0
	lowerContent := strings.ToLower(content)
	for _, token := range queryTokens {
		if strings.Contains(lowerContent, token) {
			matched++
		}
	}
	return float64(matched) / float64(len(queryTokens))
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	dot := 0.0
	magA := 0.0
	magB := 0.0
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

func nodeContent(node write.NodeRecord) string {
	return searchprofile.BuildEmbeddingText(node, ontology.ResolvedSearchProfile{})
}

func nodeACLVisibleTo(node write.NodeRecord) []string {
	if len(node.ACLVisibleTo) > 0 {
		return append([]string(nil), node.ACLVisibleTo...)
	}
	return []string{node.OwnerTenantID + ":" + node.OwnerAppID}
}

func authorityScoreFor(node write.NodeRecord, cfg *ontology.StatusFieldConfig) *int {
	if cfg == nil || cfg.AuthorityFieldName == "" || len(cfg.AuthorityValuesMap) == 0 {
		return nil
	}
	if node.Properties == nil {
		return nil
	}
	value, ok := node.Properties[cfg.AuthorityFieldName]
	if !ok {
		return nil
	}
	score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", value)]
	if !ok {
		return nil
	}
	return &score
}

func matchesQuery(node write.NodeRecord, query string) bool {
	return strings.Contains(strings.ToLower(nodeContent(node)), strings.ToLower(query))
}

func visibleOwnerSet(owners []access.VisibleOwner) map[string]struct{} {
	result := map[string]struct{}{}
	for _, owner := range owners {
		result[nodeKey(owner.TenantID, owner.AppID)] = struct{}{}
	}
	return result
}

func nodeKey(tenantID, appID string) string {
	return tenantID + ":" + appID
}
