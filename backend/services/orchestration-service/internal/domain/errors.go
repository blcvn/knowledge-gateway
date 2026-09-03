package domain

import "errors"

var (
    ErrInvalidTransition = errors.New("invalid action status transition")
    ErrLeaseConflict     = errors.New("lease conflict: action already locked by another agent")
    ErrLeaseExpired      = errors.New("lease has expired")
    ErrActionNotFound    = errors.New("action not found")
    ErrSignalNotFound    = errors.New("signal not found")
    ErrSketchNotFound    = errors.New("sketch not found")
)

type ErrLeaseConflictDetail struct {
    ActiveLease *Lease
}
func (e ErrLeaseConflictDetail) Error() string { return "lease conflict" }
