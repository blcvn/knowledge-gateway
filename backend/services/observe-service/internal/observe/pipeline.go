package observe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/pkg/privacy"
	"github.com/vnp-memory/services/observe-service/internal/domain"
	"github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

const maxObsPerSession = 500

type PipelineConfig struct {
	MaxObsPerSession int
	DedupTTL         time.Duration
	InjectContext    bool
	TokenBudget      int
}

type Pipeline struct {
	dedup     *DedupMap
	kvStore   port.IKVStore
	search    port.ISearchIndexer
	publisher port.IEventPublisher
	stream    *StreamBroker
	privacy   *privacy.Redactor
	mu        sync.Map // per-session mutex
	config    PipelineConfig
}

func NewPipeline(dedup *DedupMap, kvStore port.IKVStore, search port.ISearchIndexer,
	publisher port.IEventPublisher, stream *StreamBroker, priv *privacy.Redactor,
	cfg PipelineConfig) *Pipeline {
	return &Pipeline{dedup: dedup, kvStore: kvStore, search: search,
		publisher: publisher, stream: stream, privacy: priv, config: cfg}
}

type ObserveRequest struct {
	SessionID         string
	HookType          string
	ToolName          string
	ToolInput         []byte
	ToolOutput        []byte
	UserPrompt        string
	AssistantResponse string
	AgentID           string
	TenantID          string
	Project           string
	Timestamp         time.Time
}

type ObserveResponse struct {
	ObservationID   string
	Deduplicated    bool
	Compressed      domain.CompressedObservation
	InjectedContext string
	ContextTokens   int
}

func (p *Pipeline) Execute(ctx context.Context, req ObserveRequest) (*ObserveResponse, error) {
	// Step 1: Validate
	if req.SessionID == "" || req.HookType == "" {
		return nil, domain.ErrMissingFields
	}

	// Step 2: Dedup check
	hash := sha256.Sum256([]byte(req.SessionID + req.ToolName + fmt.Sprint(req.ToolInput)))
	if p.dedup.IsSeen(hash) {
		return &ObserveResponse{Deduplicated: true}, nil
	}

	// Step 3: Privacy redaction
	reqJSON, _ := json.Marshal(req)
	stripped := p.privacy.Strip(string(reqJSON))
	json.Unmarshal([]byte(stripped), &req)

	// Step 4: Build RawObservation
	raw := domain.RawObservation{
		ID:                uuid.New().String(),
		SessionID:         req.SessionID,
		TenantID:          req.TenantID,
		HookType:          req.HookType,
		ToolName:          req.ToolName,
		ToolInput:         req.ToolInput,
		ToolOutput:        req.ToolOutput,
		UserPrompt:        req.UserPrompt,
		AssistantResponse: req.AssistantResponse,
		AgentID:           req.AgentID,
		Timestamp:         req.Timestamp,
	}

	// Step 5: Image detection
	raw.Modality = detectModality(req.ToolInput)

	// Step 6: Per-session keyed mutex
	mu := p.getOrCreateMu(req.SessionID)
	mu.Lock()
	defer mu.Unlock()

	// Step 7: Session limit check
	count := p.kvStore.GetSessionObsCount(ctx, req.SessionID)
	if count >= maxObsPerSession {
		return nil, domain.ErrSessionLimitExceeded
	}

	// Step 8: AgentID inheritance
	if raw.AgentID == "" {
		raw.AgentID = p.kvStore.GetSessionAgentID(ctx, req.SessionID)
	}

	// Step 9: Persist RawObservation
	if err := p.kvStore.SaveRawObservation(ctx, raw); err != nil {
		return nil, err
	}

	// Step 10: Mark dedup hash seen
	p.dedup.MarkSeen(hash, p.config.DedupTTL)

	// Step 11: SSE broadcast
	p.stream.Broadcast(StreamEvent{
		Type:      "raw_observation",
		SessionID: req.SessionID,
		Data:      raw,
	})

	// Step 12: Increment session obs count
	p.kvStore.IncrementObsCount(ctx, req.SessionID)

	// Step 13: Synthetic compression
	compressed := syntheticCompress(raw)
	compressed.ID = uuid.New().String()
	p.kvStore.SaveCompressedObservation(ctx, compressed)

	// Step 14: Async index (non-blocking)
	go p.search.IndexObservation(context.Background(), compressed)

	// Publish NATS event
	p.publisher.Publish(ctx, "agentmemory.observation.captured", map[string]any{
		"observation_id": raw.ID,
		"session_id":     req.SessionID,
		"tenant_id":      req.TenantID,
		"hook_type":      req.HookType,
	})

	return &ObserveResponse{
		ObservationID: raw.ID,
		Compressed:    compressed,
	}, nil
}

func (p *Pipeline) getOrCreateMu(sessionID string) *sync.Mutex {
	mu, _ := p.mu.LoadOrStore(sessionID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}
