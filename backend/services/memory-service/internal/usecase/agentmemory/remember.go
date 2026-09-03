package agentmemory

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

const JaccardThreshold = 0.7

// RememberRequest carries data for storing a new agent memory.
type RememberRequest struct {
	TenantID    string
	Project     string
	AgentID     string
	SessionID   string
	Type        agentmemory.MemoryType
	Title       string
	Content     string
	Concepts    []string
	Files       []string
	Strength    float64
	ForgetAfter *time.Time
}

// RememberResult is returned after a successful remember operation.
type RememberResult struct {
	MemoryID   string
	Version    int
	Superseded []string
}

// RememberAgentUseCase implements Jaccard-based memory versioning.
type RememberAgentUseCase struct {
	repo         port.IMemoryRepo
	searchClient port.ISearchNotifier
	publisher    port.IEventPublisher
}

func NewRememberAgentUseCase(repo port.IMemoryRepo, searchClient port.ISearchNotifier, publisher port.IEventPublisher) *RememberAgentUseCase {
	return &RememberAgentUseCase{repo: repo, searchClient: searchClient, publisher: publisher}
}

func (uc *RememberAgentUseCase) Execute(ctx context.Context, req RememberRequest) (*RememberResult, error) {
	existing, err := uc.repo.ListLatestByType(ctx, req.TenantID, req.Project, string(req.Type))
	if err != nil {
		return nil, err
	}

	bestMatch, bestScore := findBestJaccardMatch(req.Concepts, existing)

	var superseded []string
	version := 1
	if bestScore >= JaccardThreshold && bestMatch != nil {
		if err := uc.repo.SetNotLatest(ctx, bestMatch.ID); err != nil {
			return nil, err
		}
		superseded = append(superseded, bestMatch.ID)
		version = bestMatch.Version + 1
	}

	strength := req.Strength
	if strength == 0 {
		strength = 0.7
	}

	mem := agentmemory.AgentMemory{
		ID:                   uuid.NewString(),
		TenantID:             req.TenantID,
		Project:              req.Project,
		Type:                 req.Type,
		Title:                req.Title,
		Content:              req.Content,
		Concepts:             req.Concepts,
		Files:                req.Files,
		SessionIDs:           []string{req.SessionID},
		Strength:             strength,
		Version:              version,
		Supersedes:           superseded,
		IsLatest:             true,
		ForgetAfter:          req.ForgetAfter,
		AgentID:              req.AgentID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := uc.repo.Save(ctx, mem); err != nil {
		return nil, err
	}

	go func() {
		_ = uc.searchClient.IndexMemory(ctx, mem)
	}()

	return &RememberResult{MemoryID: mem.ID, Version: mem.Version, Superseded: superseded}, nil
}

func findBestJaccardMatch(concepts []string, existing []agentmemory.AgentMemory) (*agentmemory.AgentMemory, float64) {
	var best *agentmemory.AgentMemory
	bestScore := 0.0
	for i := range existing {
		score := jaccardSimilarity(concepts, existing[i].Concepts)
		if score > bestScore {
			bestScore = score
			best = &existing[i]
		}
	}
	return best, bestScore
}

func jaccardSimilarity(a, b []string) float64 {
	setA := toLowerSet(a)
	setB := toLowerSet(b)
	intersection := 0
	for k := range setA {
		if setB[k] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func toLowerSet(tokens []string) map[string]bool {
	s := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		s[strings.ToLower(t)] = true
	}
	return s
}
