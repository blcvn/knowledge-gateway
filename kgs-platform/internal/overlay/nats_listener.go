package overlay

import (
	"context"
	"encoding/json"
	"sync"

	"kgs-platform/internal/data"

	"github.com/go-kratos/kratos/v2/log"
)

type sessionCloseEvent struct {
	SessionID string `json:"session_id"`
}

type budgetStopEvent struct {
	SessionID string `json:"session_id"`
}

type SessionCloseListener struct {
	nats    *data.NATSClient
	manager *Manager
	log     *log.Helper

	mu       sync.Mutex
	stopFunc []func()
}

func NewSessionCloseListener(nats *data.NATSClient, manager *Manager, logger log.Logger) *SessionCloseListener {
	return &SessionCloseListener{
		nats:    nats,
		manager: manager,
		log:     log.NewHelper(logger),
	}
}

func (l *SessionCloseListener) Start(ctx context.Context) error {
	if l == nil || l.nats == nil || l.manager == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.stopFunc != nil {
		return nil
	}
	stopSession, err := l.nats.Subscribe(data.TopicSessionClosePattern(), func(handlerCtx context.Context, payload []byte) {
		_ = handlerCtx
		var evt sessionCloseEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			l.log.Warnf("invalid session close payload: %v", err)
			return
		}
		if evt.SessionID == "" {
			return
		}
		if err := l.handleSessionClose(ctx, evt.SessionID); err != nil {
			l.log.Warnf("session close overlay handling failed: %v", err)
		}
	})
	if err != nil {
		return err
	}
	stopBudget, err := l.nats.Subscribe(data.TopicBudgetStopPattern(), func(handlerCtx context.Context, payload []byte) {
		_ = handlerCtx
		var evt budgetStopEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			l.log.Warnf("invalid budget stop payload: %v", err)
			return
		}
		if evt.SessionID == "" {
			return
		}
		if err := l.handleBudgetStop(ctx, evt.SessionID); err != nil {
			l.log.Warnf("budget stop overlay handling failed: %v", err)
		}
	})
	if err != nil {
		stopSession()
		return err
	}

	l.stopFunc = []func(){stopSession, stopBudget}
	return nil
}

func (l *SessionCloseListener) Stop(ctx context.Context) error {
	_ = ctx
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.stopFunc) > 0 {
		for _, stop := range l.stopFunc {
			stop()
		}
		l.stopFunc = nil
	}
	return nil
}

func (l *SessionCloseListener) handleSessionClose(ctx context.Context, sessionID string) error {
	overlayID, err := l.manager.store.FindBySession(ctx, sessionID)
	if err != nil || overlayID == "" {
		return err
	}
	item, err := l.manager.store.Get(ctx, overlayID)
	if err != nil {
		return err
	}
	if len(item.EntitiesDelta)+len(item.EdgesDelta) > 0 {
		_, err = l.manager.Commit(ctx, overlayID, PolicyKeepOverlay)
		return err
	}
	return l.manager.Discard(ctx, overlayID)
}

func (l *SessionCloseListener) handleBudgetStop(ctx context.Context, sessionID string) error {
	overlayID, err := l.manager.store.FindBySession(ctx, sessionID)
	if err != nil || overlayID == "" {
		return err
	}
	_, err = l.manager.CommitPartial(ctx, overlayID, PolicyKeepOverlay)
	return err
}
