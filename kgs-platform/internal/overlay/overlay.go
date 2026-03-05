package overlay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kgs-platform/internal/data"
	"kgs-platform/internal/observability"
	"kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const defaultOverlayTTL = time.Hour

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

type Manager struct {
	store      Store
	versionMgr version.VersionManager
	publisher  EventPublisher
	log        *log.Helper
}

type OverlayManager interface {
	Create(ctx context.Context, namespace, sessionID, baseVersionID string) (*OverlayGraph, error)
	Get(ctx context.Context, overlayID string) (*OverlayGraph, error)
	Commit(ctx context.Context, overlayID, conflictPolicy string) (*CommitResult, error)
	Discard(ctx context.Context, overlayID string) error
	DiscardBySession(ctx context.Context, sessionID string) error
}

func NewManager(store Store, versionMgr version.VersionManager, publisher EventPublisher, logger log.Logger) *Manager {
	return &Manager{
		store:      store,
		versionMgr: versionMgr,
		publisher:  publisher,
		log:        log.NewHelper(logger),
	}
}

func (m *Manager) Create(ctx context.Context, namespace, sessionID, baseVersionID string) (*OverlayGraph, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	baseVersionID = m.resolveBaseVersion(ctx, namespace, baseVersionID)
	now := time.Now().UTC()
	overlay := &OverlayGraph{
		OverlayID:     uuid.NewString(),
		Namespace:     namespace,
		SessionID:     sessionID,
		BaseVersionID: baseVersionID,
		Status:        StatusActive,
		EntitiesDelta: []EntityDelta{},
		EdgesDelta:    []EdgeDelta{},
		CreatedAt:     now,
		ExpiresAt:     now.Add(defaultOverlayTTL),
	}
	if err := m.store.Save(ctx, overlay, defaultOverlayTTL); err != nil {
		return nil, err
	}
	if err := m.store.BindSession(ctx, sessionID, overlay.OverlayID, defaultOverlayTTL); err != nil {
		return nil, err
	}
	observability.IncOverlayActive(namespace)
	return overlay, nil
}

func (m *Manager) Get(ctx context.Context, overlayID string) (*OverlayGraph, error) {
	return m.store.Get(ctx, overlayID)
}

func (m *Manager) Discard(ctx context.Context, overlayID string) error {
	overlay, err := m.store.Get(ctx, overlayID)
	if err != nil {
		return err
	}
	overlay.Status = StatusDiscarded
	if err := m.store.Delete(ctx, overlayID); err != nil {
		return err
	}
	if overlay.SessionID != "" {
		_ = m.store.UnbindSession(ctx, overlay.SessionID)
	}
	observability.DecOverlayActive(overlay.Namespace)
	if m.publisher != nil {
		payload, _ := json.Marshal(map[string]any{
			"overlay_id": overlay.OverlayID,
			"namespace":  overlay.Namespace,
			"session_id": overlay.SessionID,
			"status":     overlay.Status,
		})
		_ = m.publisher.Publish(ctx, data.TopicOverlayDiscarded(overlay.Namespace), payload)
	}
	return nil
}

func (m *Manager) DiscardBySession(ctx context.Context, sessionID string) error {
	overlayID, err := m.store.FindBySession(ctx, sessionID)
	if err != nil || overlayID == "" {
		return err
	}
	return m.Discard(ctx, overlayID)
}

func (m *Manager) resolveBaseVersion(ctx context.Context, namespace, baseVersionID string) string {
	if baseVersionID != "" && baseVersionID != "current" {
		return baseVersionID
	}
	if m.versionMgr == nil {
		return ""
	}
	versions, err := m.versionMgr.ListVersions(ctx, namespace)
	if err != nil || len(versions) == 0 {
		return ""
	}
	return versions[0].ID
}

var _ OverlayManager = (*Manager)(nil)
