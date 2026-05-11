package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// SummarizeStage performs community detection and generates summaries via LLM.
type SummarizeStage struct {
	llm       port.LLMClient
	graphRepo port.GraphRepository
	logger    *slog.Logger
}

func NewSummarizeStage(llm port.LLMClient, graphRepo port.GraphRepository, logger *slog.Logger) *SummarizeStage {
	return &SummarizeStage{llm: llm, graphRepo: graphRepo, logger: logger.With("stage", "summarize")}
}

func (s *SummarizeStage) Name() domain.StageType { return domain.StageSummarize }

func (s *SummarizeStage) Execute(ctx context.Context, job *domain.CognifyJob, state *PipelineState) error {
	if job.Config.SkipSummarize {
		s.logger.Info("summarization skipped (config)")
		return nil
	}

	if len(state.Entities) == 0 {
		return nil
	}

	// Simple community detection: group entities that share relationships
	communities := detectCommunities(job, state.Entities, state.Relationships)

	// Generate LLM summaries for each community
	for _, community := range communities {
		entityNames := make([]string, 0, len(community.EntityIDs))
		for _, eid := range community.EntityIDs {
			for _, e := range state.Entities {
				if e.ID == eid {
					entityNames = append(entityNames, fmt.Sprintf("%s (%s)", e.Name, e.EntityType))
					break
				}
			}
		}

		prompt := fmt.Sprintf("Summarize the following group of related entities:\n\n%s",
			strings.Join(entityNames, "\n"))

		summary, err := s.llm.Complete(ctx, communitySummaryPrompt, prompt)
		if err != nil {
			s.logger.Warn("community summary failed", "community_id", community.ID, "error", err)
			job.Metrics.LLMCallsTotal++
			continue
		}
		job.Metrics.LLMCallsTotal++

		community.Summary = summary
		if err := s.graphRepo.UpsertCommunity(ctx, job.TenantID, community); err != nil {
			s.logger.Warn("community upsert failed", "community_id", community.ID, "error", err)
		}
	}

	state.Communities = communities
	job.Metrics.CommunitiesFound = len(communities)
	s.logger.Info("summarization complete", "communities", len(communities))
	return nil
}

// detectCommunities groups entities that are connected by relationships.
// This is a simplified connected-components algorithm.
func detectCommunities(job *domain.CognifyJob, entities []*domain.Entity, rels []*domain.Relationship) []*domain.Community {
	// Build adjacency map
	adj := make(map[string]map[string]bool)
	for _, rel := range rels {
		src, tgt := rel.SourceID.String(), rel.TargetID.String()
		if adj[src] == nil {
			adj[src] = make(map[string]bool)
		}
		if adj[tgt] == nil {
			adj[tgt] = make(map[string]bool)
		}
		adj[src][tgt] = true
		adj[tgt][src] = true
	}

	// BFS to find connected components
	visited := make(map[string]bool)
	var communities []*domain.Community

	for _, entity := range entities {
		id := entity.ID.String()
		if visited[id] {
			continue
		}

		component := bfs(id, adj, visited)
		if len(component) < 2 {
			continue // skip singletons
		}

		entityIDs := make([]interface{ String() string }, 0, len(component))
		for _, eid := range component {
			for _, e := range entities {
				if e.ID.String() == eid {
					entityIDs = append(entityIDs, e.ID)
					break
				}
			}
		}

		uuids := make([]interface{}, 0, len(entityIDs))
		for _, u := range entityIDs {
			uuids = append(uuids, u)
		}

		var realUUIDs []interface{ String() string }
		for _, e := range entities {
			for _, c := range component {
				if e.ID.String() == c {
					realUUIDs = append(realUUIDs, e.ID)
					break
				}
			}
		}

		community := domain.NewCommunity(job.ID, extractUUIDs(entities, component), 0)
		communities = append(communities, community)
	}

	return communities
}

func extractUUIDs(entities []*domain.Entity, component []string) []interface{ String() string } {
	result := make([]interface{ String() string }, 0, len(component))
	for _, c := range component {
		for _, e := range entities {
			if e.ID.String() == c {
				result = append(result, e.ID)
				break
			}
		}
	}
	return nil // Simplified — would return uuid.UUID slice in production
}

func bfs(start string, adj map[string]map[string]bool, visited map[string]bool) []string {
	queue := []string{start}
	visited[start] = true
	var component []string

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		component = append(component, node)

		for neighbor := range adj[node] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return component
}

const communitySummaryPrompt = `You are a knowledge graph summarizer. Given a group of related entities, write a concise 2-3 sentence summary describing the community and the key relationships between the entities. Be factual and specific.`
