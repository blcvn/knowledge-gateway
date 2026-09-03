package orchestration

import (
    "context"

    "vnp-memory/services/orchestration-service/internal/domain"
    "vnp-memory/services/orchestration-service/internal/usecase/port"
)

type SentinelService struct {
    repo       port.ISentinelRepo
    actionRepo port.IActionRepo
    publisher  port.IEventPublisher
}

// EvaluateAll runs every 30s to check all "watching" sentinels
func (s *SentinelService) EvaluateAll(ctx context.Context) {
    sentinels, _ := s.repo.ListWatching(ctx)
    for _, sentinel := range sentinels {
        if s.conditionMet(ctx, sentinel.Condition) {
            s.trigger(ctx, sentinel)
        }
    }
}

func (s *SentinelService) conditionMet(ctx context.Context, cond domain.SentinelCondition) bool {
    switch cond.Type {
    case "action_done":
        action, _ := s.actionRepo.Get(ctx, cond.Target)
        return action != nil && action.Status == domain.ActionDone
    case "signal_received":
        return false // TODO: integrate with signal service
    case "time":
        return cronMatches(cond.Target)  // cron expression evaluation
    }
    return false
}

func (s *SentinelService) trigger(ctx context.Context, sentinel domain.Sentinel) {
    s.repo.SetStatus(ctx, sentinel.ID, "triggered")

    if sentinel.ActionID != "" {
        s.actionRepo.TransitionStatus(ctx, sentinel.ActionID, domain.ActionPending, domain.ActionActive)
    }

    s.publisher.Publish(ctx, "agentmemory.sentinel.triggered", map[string]any{
        "sentinel_id": sentinel.ID, "condition": sentinel.Condition,
    })
}

// cronMatches: simple time-based condition check
func cronMatches(cronExpr string) bool {
    // Basic implementation: use time.Now() vs cron expression
    // For prod: use robfig/cron or similar
    return false
}
