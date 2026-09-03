package domain

type ActionStatus string

const (
    ActionPending   ActionStatus = "pending"
    ActionActive    ActionStatus = "active"
    ActionBlocked   ActionStatus = "blocked"
    ActionDone      ActionStatus = "done"
    ActionCancelled ActionStatus = "cancelled"
    ActionFailed    ActionStatus = "failed"
)

// validTransitions defines allowed state machine transitions
var validTransitions = map[ActionStatus][]ActionStatus{
    ActionPending:   {ActionActive, ActionBlocked, ActionCancelled},
    ActionActive:    {ActionDone, ActionBlocked, ActionCancelled, ActionFailed},
    ActionBlocked:   {ActionActive, ActionCancelled},
    ActionDone:      {},
    ActionCancelled: {},
    ActionFailed:    {},
}

func (current ActionStatus) CanTransitionTo(next ActionStatus) bool {
    allowed := validTransitions[current]
    for _, a := range allowed { if a == next { return true } }
    return false
}
